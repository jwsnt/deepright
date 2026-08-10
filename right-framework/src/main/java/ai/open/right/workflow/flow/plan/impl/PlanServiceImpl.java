package ai.open.right.workflow.flow.plan.impl;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.iteration.IterationService;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.plan.PlanConfig;
import ai.open.right.workflow.flow.plan.PlanService;
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

import java.util.List;

@Slf4j
@Setter
@Getter
public class PlanServiceImpl implements PlanService {

    protected IterationService iterationService;

    protected NotifierService notifierService;

    protected HistoryStore historyStore;

    // Plan调用下游思考链条（Workflow）超时
    protected Integer timeout4Llm;

    @Override
    public String plan(PlanConfig planConfig, WorkflowTask workTask) throws Exception {
        try {
            workTask.setQuery(this.buildPlan(planConfig, workTask));
            String content = this.buildIteration(planConfig, workTask);
            if (planConfig.hasSummary()) {
                return this.buildSummary(planConfig, workTask, content);
            } else {
                this.storeIteration(planConfig, workTask, content);
                return content;
            }
        } catch (Exception e) {
            if (planConfig.hasException()) {
                return this.buildException(planConfig, workTask, e.getMessage());
            } else {
                throw e;
            }
        }
    }

    protected String buildIteration(PlanConfig planConfig, WorkflowTask workTask) throws Exception {
        return this.iterationService.iterate(planConfig.getIterationConfig(), workTask);
    }

    protected String buildPlan(PlanConfig planConfig, WorkflowTask workTask) throws Exception {
        SyncConfig syncConfig = SyncConfig.builder()
                .syncCallable(planConfig.hasNotifierWithPlan() ? new NotifierCallable(planConfig.getNotifier().getPlan()) : null)
                .timeout(planConfig.getTimeout4Llm(this.timeout4Llm))
                .workflow(planConfig.getPlan())
                .reQuery(workTask.getQuery())
                .workTask(workTask)
                .build();
        String plan = SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get();
        if (log.isDebugEnabled()) {
            log.debug("Plan={}", plan);
        }
        Assert.hasText(plan, "Plan can not be empty");
        return plan;
    }

    // 异常处理
    protected String buildException(PlanConfig planConfig, WorkflowTask workTask, String query) throws Exception {
        SyncConfig syncConfig = SyncConfig.builder()
                // 指定通知方式
                .syncCallable(planConfig.hasNotifierWithException() ? new NotifierCallable(planConfig.getNotifier().getException()) : null)
                .timeout(planConfig.getTimeout4Llm(this.timeout4Llm))
                .workflow(planConfig.getException())
                .workTask(workTask)
                .reQuery(query)
                .build();
        String exception = SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get();
        if (log.isInfoEnabled()) {
            log.info("Plan exception={}", exception);
        }
        this.storeException(planConfig, workTask, exception);
        return exception;
    }

    // 总结
    protected String buildSummary(PlanConfig planConfig, WorkflowTask workTask, String query) throws Exception {
        SyncConfig syncConfig = SyncConfig.builder()
                // 指定通知方式
                .syncCallable(planConfig.hasNotifierWithSummary() ? new NotifierCallable(planConfig.getNotifier().getSummary()) : null)
                .timeout(planConfig.getTimeout4Llm(this.timeout4Llm))
                .workflow(planConfig.getSummary())
                .workTask(workTask)
                .reQuery(query)
                .build();
        String summary = SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get();
        if (log.isInfoEnabled()) {
            log.info("Plan summary={}", summary);
        }
        this.storeSummary(planConfig, workTask, summary);
        return summary;
    }

    // 用于子类覆盖
    protected void storeHistories(PlanConfig planConfig, WorkflowTask workTask, String answer) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Plan query={}, answer={}", workTask.getQuery(), answer);
        }
        if (planConfig.getContainHistories()) {
            List<String> repositories = this.buildRepositories(planConfig, workTask, answer);
            this.historyStore.store(workTask, repositories, workTask.getQuery(), answer, planConfig.getLlmConfig().getExpired(), planConfig.getLlmConfig().getHistories(), workTask.getCreated());
        }
    }

    // 用于子类覆盖
    protected void storeIteration(PlanConfig planConfig, WorkflowTask workTask, String answer) throws Exception {
        this.storeHistories(planConfig, workTask, answer);
    }

    // 用于子类覆盖
    protected void storeException(PlanConfig planConfig, WorkflowTask workTask, String answer) throws Exception {
        this.storeHistories(planConfig, workTask, answer);
    }

    // 用于子类覆盖
    protected void storeSummary(PlanConfig planConfig, WorkflowTask workTask, String answer) throws Exception {
        this.storeHistories(planConfig, workTask, answer);
    }

    public List<String> buildRepositories(PlanConfig planConfig, WorkflowTask workTask, String answer) throws Exception {
        return planConfig.getLlmConfig().buildRepositories(workTask.getWorkflow());
    }

    @ConditionalOnProperty(name = "plan.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected IterationService iterationService;

        @Autowired
        protected NotifierService notifierService;

        @Autowired
        protected HistoryStore historyStore;

        @Value("${plan.timeout.llm:1800000}")
        // Plan调用下游思考链条（Workflow）超时
        protected Integer timeout4Llm;

        @Bean
        @ConditionalOnMissingBean(value = PlanService.class)
        public PlanService planService() throws Exception {
            PlanServiceImpl planService = new PlanServiceImpl();
            BeanUtils.copyProperties(this, planService);
            log.info("PlanServiceImpl inited: timeout4Llm={}", planService.getTimeout4Llm());
            return planService;
        }
    }
}
