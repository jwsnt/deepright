package ai.open.right.workflow.flow.parallel.impl;

import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.parallel.ParallelConfig;
import ai.open.right.workflow.flow.parallel.ParallelFlow;
import ai.open.right.workflow.flow.parallel.ParallelService;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import ai.open.right.workflow.sync.impl.NotifierCallable;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.util.ArrayList;
import java.util.List;

@Slf4j
@Setter
@Getter
public class ParallelServiceImpl implements ParallelService {

    protected NotifierService notifierService;

    // 并行调用下游思考链（Workflow）超时
    protected Integer timeout4Llm;

    @Override
    public String execute(ParallelConfig parallelConfig, WorkflowTask workTask) throws Exception {
        Assert.isTrue(parallelConfig.hasParallelFlow(), "Parallel flow can not be empty, please check config");
        List<SyncWorkflowTask> syncWorkflowTasks = new ArrayList<SyncWorkflowTask>();
        for (ParallelFlow parallelFlow : parallelConfig.getParallelFlow()) {
            SyncConfig syncConfig = SyncConfig.builder()
                    // 指定通知方式
                    .syncCallable(parallelConfig.hasNotifier() ? new NotifierCallable(parallelConfig.getNotifier()) : null)
                    .timeout(parallelConfig.getTimeout4Llm(this.timeout4Llm))
                    .workflow(parallelFlow.getDynamic())
                    .reQuery(workTask.getQuery())
                    .workTask(workTask)
                    .build();
            syncWorkflowTasks.add(SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig));
        }
        return this.getParallelResponse(parallelConfig, syncWorkflowTasks);
    }

    protected String getParallelResponse(ParallelConfig parallelConfig, List<SyncWorkflowTask> syncWorkflowTasks) throws Exception {
        StringBuffer buffer = new StringBuffer();
        for (int idx = 0; idx < syncWorkflowTasks.size(); idx++) {
            try {
                String response = syncWorkflowTasks.get(idx).get();
                if (log.isDebugEnabled()) {
                    log.debug("Parallel each response={}", response);
                }
                Assert.hasText(response, "Parallel response can not be empty");
                buffer.append(response);
            } catch (Exception e) {
                if (!parallelConfig.getParallelFlow().get(idx).getStopOnFailed()) {
                    WorkflowException.dolog(e);
                } else {
                    throw e;
                }
            }
        }
        String response = buffer.toString();
        if (log.isInfoEnabled()) {
            log.info("Parallel response={}", response);
        }
        Assert.hasText(response, "Parallel response can not be empty");
        return response;
    }

    @ConditionalOnProperty(name = "parallel.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected NotifierService notifierService;

        @Value("${parallel.timeout.llm:1800000}")
        // 并行调用下游思考链（Workflow）超时
        protected Integer timeout4Llm;

        @Bean
        @ConditionalOnMissingBean(value = ParallelService.class)
        public ParallelService parallelService() throws Exception {
            ParallelServiceImpl parallelService = new ParallelServiceImpl();
            BeanUtils.copyProperties(this, parallelService);
            log.info("ParallelServiceImpl inited, timeout4Llm={}", parallelService.getTimeout4Llm());
            return parallelService;
        }
    }
}
