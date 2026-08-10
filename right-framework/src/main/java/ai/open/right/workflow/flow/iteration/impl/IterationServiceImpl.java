package ai.open.right.workflow.flow.iteration.impl;

import ai.open.right.workflow.condition.Condition;
import ai.open.right.workflow.condition.ConditionUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.iteration.IterationConfig;
import ai.open.right.workflow.flow.iteration.IterationService;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.track.TrackDimension;
import ai.open.right.workflow.flow.track.TrackFunCall;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import ai.open.right.workflow.sync.impl.NotifierCallable;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.util.List;

@Slf4j
@Setter
@Getter
public class IterationServiceImpl implements IterationService {

    public static final String KEY_TRACK = "_iteration_track";

    // 当前迭代次数（Metadata Key）
    // 在高迭代次数场景下，如果每轮响应很长，内存占用会很可观
    public static final String KEY_TIMES = "_iteration_times";

    public static final Integer STATUS_EXCEPTION = 0;

    public static final Integer STATUS_NORMAL = 1;

    protected TrackFunCallService trackFunCallService;

    protected NotifierService notifierService;

    protected HistoryStore historyStore;

    // 最大迭代次数
    protected Integer maxTimes;

    // 最大Query
    protected Integer maxSize;

    // 调用下游超时
    protected Integer timeout;

    // 迭代Query前缀（影响LLM输入格式）
    private String prefix = "##################" + System.lineSeparator();

    // 迭代Query后缀（影响LLM输入格式）
    private String suffix = "##################" + System.lineSeparator();

    protected String condition = "The check recommendations round";

    // 迭代响应前缀（影响LLM输入格式）
    protected String answer = "The answer round";

    protected String error = "The error round";

    // 迭代请求前缀（影响LLM输入格式）
    protected String query = "The query round";

    @Override
    public String iterate(IterationConfig iterationConfig, WorkflowTask workTask) throws Exception {
        // 最大迭代次数检查
        Assert.isTrue(this.maxTimes > iterationConfig.getTimes(), "The iteration times must be less than maxTimes: " + this.maxTimes);
        Assert.isTrue(iterationConfig.hasProcessor(), "The iteration must config the `processor`, please check the config");
        // 构建迭代过程记录
        StringBuffer history = new StringBuffer(this.buildInitial(iterationConfig, workTask)).append(this.appendQuery(1, workTask.getQuery()));
        // 当前Query
        String q = workTask.getQuery();
        // 当前Answer
        String a = "";
        try {
            // 当前迭代是否为正常
            int status = IterationServiceImpl.STATUS_NORMAL;
            // Store Track，开启Fun Call过程存储
            this.storeFunCalls(iterationConfig, workTask);
            for (int index = 1; index <= iterationConfig.getTimes(); index++) {
                try {
                    if (log.isDebugEnabled()) {
                        log.debug("Iteration round status={},index={},history={}", status, index, history);
                    }
                    // 在Metadata存放当前迭代次数
                    workTask.putMetadata(IterationServiceImpl.KEY_TIMES, index);
                    // First
                    if (index != 1) {
                        // Condition判断是否要继续迭代或者不需要反思
                        if (!this.checkStatus(iterationConfig, workTask, status) && !this.checkCondition(iterationConfig, workTask, history, a, index)) {
                            if (log.isInfoEnabled()) {
                                log.info("Break the iteration during round={},status={}", index, status);
                            }
                            break;
                        }
                        // 反思Query
                        q = this.buildRefection(iterationConfig, workTask, history, a, index);
                        if (log.isInfoEnabled()) {
                            log.info("Iteration round={},query={}", index, q);
                        }
                        history.append(this.appendQuery(index, q));
                    }
                    // 再次迭代，不会传递上次迭代的Fun Call
                    Assert.isTrue(!workTask.containMetadata(ProviderRequestService.KEY_FUN_MERGE), "Fun Call Data should be empty");
                    SyncConfig syncConfig = SyncConfig.builder()
                            // 迭代过程是否需要反馈Notifier（比如终端）
                            .syncCallable(iterationConfig.hasNotifierWithProcessor() ? new NotifierCallable(iterationConfig.getNotifier().getProcessor()) : null)
                            .reQuery(this.buildProcess(iterationConfig, workTask, history, a, index))
                            .timeout(iterationConfig.getTimeout(this.timeout))
                            .workflow(iterationConfig.getProcessor())
                            .workTask(workTask)
                            .build();
                    a = SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get();
                    if (log.isInfoEnabled()) {
                        log.info("Iteration round={},answer={}", index, a);
                    }
                    // 本次反思的响应追加入迭代过程记录
                    history.append(this.appendAnswer(index, a));
                    // 标记当前迭代为正常结束
                    status = this.buildSuccessStatus(iterationConfig, workTask, history, a, a);
                } catch (Exception e) {
                    if (log.isInfoEnabled()) {
                        log.info(e.getMessage(), e);
                    }
                    // 标记当前迭代为正常结束
                    status = this.buildExceptionStatus(iterationConfig, workTask, history, a, q);
                    // 如果是最后一次迭代，则抛出异常
                    if (index == (iterationConfig.getTimes() - 1)) {
                        throw e;
                    } else {
                        // 否则将异常加入到迭代过程记录，继续迭代
                        history.append(this.appendException(index, e));
                    }
                }
            }
            // Restore Track
            this.restoreFunCalls(iterationConfig, workTask);
            // Store History
            this.storeHistories(iterationConfig, workTask, history, a);
        } finally {
            workTask.delMetadata(IterationServiceImpl.KEY_TIMES);
            workTask.closeFunCallTrack();
        }
        // Last answer，返回最后结果
        return a;
    }

    // 迭代条件判断（不配置条件则默认False，不需要再次迭代）
    protected Boolean checkCondition(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer, Integer idx) throws Exception {
        if (iterationConfig.hasCondition()) {
            SyncConfig syncConfig = SyncConfig.builder()
                    .timeout(iterationConfig.getTimeout(this.timeout))
                    .workflow(iterationConfig.getCondition())
                    .reQuery(history.toString())
                    .workTask(workTask)
                    .build();
            // True: True/true/Yes/Y/1
            // False: False/false/No/N/0 and Other
            // Json: {...,"condition":true/false/0/1}
            String condition2string = StringUtils.lowerCase(SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get());
            Condition condition = ConditionUtils.checkCondition(condition2string);
            if (condition.getCondition()) {
                // 失败则将Condition追加到History
                history.append(this.buildCondition(iterationConfig, workTask, history, condition.getContent(), idx));
            }
            return condition.getCondition();
        }
        return false;
    }

    // 迭代状态判断
    protected Boolean checkStatus(IterationConfig iterationConfig, WorkflowTask workTask, Integer status) throws Exception {
        // 如果为Exception则强制迭代（返回True）
        return IterationServiceImpl.STATUS_EXCEPTION.equals(status);
    }

    protected String buildCondition(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String condition, Integer idx) throws Exception {
        return !StringUtils.isEmpty(condition) ? this.appendCondition(idx, condition) : "";
    }

    // 反思
    protected String buildRefection(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer, Integer idx) throws Exception {
        if (iterationConfig.hasRefection()) {
            // 使用模型反思
            SyncConfig syncConfig = SyncConfig.builder()
                    // 反思过程是否需要反馈Notifier（比如终端）
                    .syncCallable(iterationConfig.hasNotifierWithRefection() ? new NotifierCallable(iterationConfig.getNotifier().getRefection()) : null)
                    .timeout(iterationConfig.getTimeout(this.timeout))
                    .workflow(iterationConfig.getRefection())
                    .reQuery(history.toString())
                    .workTask(workTask)
                    .build();
            return StringUtils.defaultIfEmpty(SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get(), "");
        } else {
            // 使用静态反思
            return workTask.getQuery();
        }
    }

    // 执行
    protected String buildProcess(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer, Integer idx) throws Exception {
        return StringUtils.right(history.toString(), this.maxSize);
    }

    protected String buildInitial(IterationConfig iterationConfig, WorkflowTask workflowTask) throws Exception {
        return "The user's original query: " + workflowTask.getQuery() + System.lineSeparator() + this.prefix;
    }

    protected Integer buildExceptionStatus(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer, String query) throws Exception {
        return IterationServiceImpl.STATUS_EXCEPTION;
    }

    protected Integer buildSuccessStatus(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer, String query) throws Exception {
        return IterationServiceImpl.STATUS_NORMAL;
    }

    // 获取FunCall Track(Request/Response)
    protected void restoreFunCalls(IterationConfig iterationConfig, WorkflowTask workTask) throws Exception {
        if (iterationConfig.hasFunCallTrack()) {
            List<TrackFunCall> trackFunCalls = this.trackFunCallService.restore(this.buildTrackDimension(workTask));
            if (!CollectionUtils.isEmpty(trackFunCalls)) {
                workTask.putMetadata(IterationServiceImpl.KEY_TRACK, trackFunCalls);
                if (log.isDebugEnabled()) {
                    log.debug("Iteration track fun call={}", trackFunCalls.size());
                }
            }
        }
    }

    protected void storeFunCalls(IterationConfig iterationConfig, WorkflowTask workTask) throws Exception {
        // 记录FunCall Track(Request/Response)
        // @See ProviderStream.trackFunCall
        if (iterationConfig.hasFunCallTrack()) {
            if (log.isDebugEnabled()) {
                log.debug("Iteration start track fun call");
            }
            workTask.beginFunCallTrack();
        }
    }

    // 用于子类覆盖
    protected void storeHistories(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Iteration answer={}, history={}", answer, history);
        }
        if (iterationConfig.getContainHistories()) {
            List<String> repositories = this.buildRepositories(iterationConfig, workTask, history, answer);
            String historyAnswer = this.buildAnswer(iterationConfig, workTask, history, answer);
            String historyQuery = this.buildQuery(iterationConfig, workTask, history, answer);
            this.historyStore.store(workTask, repositories, historyQuery, historyAnswer, iterationConfig.getLlmConfig().getExpired(), iterationConfig.getLlmConfig().getHistories(), workTask.getCreated());
        }
    }

    public List<String> buildRepositories(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer) throws Exception {
        return iterationConfig.getLlmConfig().buildRepositories(workTask.getWorkflow());
    }

    protected String buildAnswer(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer) throws Exception {
        return answer;
    }

    protected String buildQuery(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer) throws Exception {
        return workTask.getQuery();
    }

    protected TrackDimension buildTrackDimension(WorkflowTask workTask) throws Exception {
        return new TrackDimension(workTask, workTask.getFunCallTrack());
    }

    protected String appendException(Integer index, Exception exception) throws Exception {
        return this.error + " " + index + ": " + exception.getMessage() + System.lineSeparator() + this.suffix;
    }

    protected String appendCondition(Integer index, String condition) throws Exception {
        return this.condition + " " + index + ": " + condition + System.lineSeparator() + this.suffix;
    }

    protected String appendAnswer(Integer index, String answer) throws Exception {
        return this.answer + " " + index + ": " + answer + System.lineSeparator() + this.suffix;
    }

    protected String appendQuery(Integer index, String query) throws Exception {
        return this.query + " " + index + ": " + query + System.lineSeparator();
    }

    @ConditionalOnProperty(name = "iteration.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected TrackFunCallService trackFunCallService;

        @Autowired
        protected NotifierService notifierService;

        @Autowired
        protected HistoryStore historyStore;

        @Value("${iteration.maxtimes:10}")
        // 最大迭代次数
        protected Integer maxTimes;

        @Value("${iteration.maxsize:102400}")
        protected Integer maxSize;

        @Value("${iteration.timeout:1800000}")
        // 调用下游超时
        protected Integer timeout;

        @Value("${iteration.prefix:##################\n}")
        // 迭代Query前缀（影响LLM输入格式）
        private String prefix = "##################" + System.lineSeparator();

        @Value("${iteration.line:##################\n}")
        // 迭代Query后缀（影响LLM输入格式）
        private String suffix = "##################" + System.lineSeparator();

        @Value("${iteration.answer:The answer round}")
        // 迭代响应前缀（影响LLM输入格式）
        protected String answer = "The answer round";

        @Value("${iteration.query:The query round}")
        // 迭代请求前缀（影响LLM输入格式）
        protected String query = "The query round";

        @Bean
        @ConditionalOnMissingBean(value = IterationService.class)
        public IterationService iterationService() throws Exception {
            IterationServiceImpl iterationService = new IterationServiceImpl();
            BeanUtils.copyProperties(this, iterationService);
            log.info("IterationServiceImpl inited, maxTimes={}, maxSize={}, timeout={}", iterationService.getMaxTimes(), iterationService.getMaxSize(), iterationService.getTimeout());
            return iterationService;
        }
    }
}
