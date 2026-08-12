package ai.open.right.workflow.flow.fork.impl;

import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.condition.ConditionUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.fork.ForkService;
import ai.open.right.workflow.flow.fork.ForkTarget;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import lombok.Builder;
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
import org.springframework.util.CollectionUtils;

import java.util.ArrayList;
import java.util.List;

@Slf4j
@Setter
@Getter
// 多路分发
public class ForkServiceImpl implements ForkService {

    protected NotifierService notifierService;

    // 调用下游思考链（Workflow）的超时
    protected Integer timeout;

    @Override
    public void fork(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        // 通知数量
        Integer notified = 0;
        if (!CollectionUtils.isEmpty(workflowConfig.getForkConfig().getTarget())) {
            List<ForkFuture> forkFutures = null;
            for (ForkTarget target : workflowConfig.getForkConfig().getTarget()) {
                if (target.hasCondition()) {
                    // 需要条件分支Condition（可选）
                    if (forkFutures == null) {
                        forkFutures = new ArrayList<ForkFuture>();
                    }
                    SyncConfig syncConfig = SyncConfig.builder()
                            .timeout(workflowConfig.getForkConfig().getTimeout(this.timeout))
                            .workflow(target.getCondition())
                            .reQuery(workTask.getQuery())
                            .workTask(workTask)
                            .build();
                    forkFutures.add(ForkFuture.builder()
                            .conditionTask(SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig))
                            .target(target)
                            .build());
                } else {
                    // 不需要条件分支
                    this.fork2Target(workTask, target.getDynamic());
                    notified++;
                }
            }
            // 检查条件状态并分发
            notified += this.checkAndFork(workflowConfig, workTask, forkFutures);
        }
        if (log.isDebugEnabled()) {
            log.debug("Fork notification size={}", notified);
        }
        // 没有通知者则使用Chain
        if (notified == 0 && workflowConfig.hasChain()) {
            if (log.isInfoEnabled()) {
                log.info("Fork notification through the default target={}", workflowConfig.getChain());
            }
            this.fork2Target(workTask, workflowConfig.getChain());
        }
    }

    // 检查条件状态并分发
    protected Integer checkAndFork(WorkflowConfig workflowConfig, WorkflowTask workTask, List<ForkFuture> forkFutures) throws Exception {
        Integer notified = 0;
        if (!CollectionUtils.isEmpty(forkFutures)) {
            for (ForkFuture forkFuture : forkFutures) {
                try {
                    // True: True/true/Yes/Y/1
                    // False: False/false/No/N/0 and Other
                    // Json: {...,"condition":true/false/0/1}
                    String condition = StringUtils.lowerCase(forkFuture.getConditionTask().get());
                    if (ConditionUtils.checkCondition(condition).print().getCondition()) {
                        this.fork2Target(workTask, forkFuture.getTarget().getDynamic());
                        notified++;
                    }
                } catch (Exception e) {
                    // 任一失败是否终止
                    if (workflowConfig.getForkConfig().getStopOnFailed()) {
                        throw e;
                    } else {
                        if (log.isInfoEnabled()) {
                            log.info(e.getMessage(), e);
                        }
                    }
                }
            }
        }
        return notified;
    }

    // 分发
    protected void fork2Target(WorkflowTask workTask, String workflow) throws Exception {
        Segment.SegmentConfig segmentConfig = Segment.SegmentConfig.builder()
                .content(workTask.getQuery() != null ? new StringBuffer(workTask.getQuery()) : null)
                .metadata(workTask.getMetadata())
                // 固定为Localhost
                .notifier(Notifier.LOCALHOST)
                .code(ProtocolCode.C200)
                .workflow(workflow)
                .build();
        Segment segment = Segment.build(workTask, segmentConfig);
        this.notifierService.notify(segment, workTask, workTask);
    }

    @Builder
    @Setter
    @Getter
    public static class ForkFuture {

        protected SyncWorkflowTask conditionTask;

        protected ForkTarget target;
    }

    @ConditionalOnProperty(name = "fork.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected NotifierService notifierService;

        @Value("${fork.timeout:1800000}")
        // 调用下游思考链（Workflow）的超时
        protected Integer timeout;

        @Bean
        @ConditionalOnMissingBean(value = ForkService.class)
        public ForkService forkService() throws Exception {
            ForkServiceImpl forkService = new ForkServiceImpl();
            BeanUtils.copyProperties(this, forkService);
            log.info("ForkServiceImpl inited: timeout={}", forkService.getTimeout());
            return forkService;
        }
    }
}
