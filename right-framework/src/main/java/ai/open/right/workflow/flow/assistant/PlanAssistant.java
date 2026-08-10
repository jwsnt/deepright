package ai.open.right.workflow.flow.assistant;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.plan.PlanService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

@Setter
@Getter
@Slf4j
// Plan-Execute
public class PlanAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-plan";

    protected PlanService planService;

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        Assert.isTrue(workflowConfig.hasPlan() && workflowConfig.getPlanConfig().hasIteration(), "Plan/Iteration config can not be empty, please check config");
        String planContent = this.planService.plan(workflowConfig.getPlanConfig(), workTask);
        this.chainOr2Endpoint(workflowConfig, workTask, planContent);
    }

    @ConditionalOnProperty(name = "plan.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected PlanService planService;

        @Bean(PlanAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = PlanAssistant.WORKFLOW_NAME)
        public PlanAssistant planAssistant() throws Exception {
            PlanAssistant planAssistant = new PlanAssistant();
            BeanUtils.copyProperties(this, planAssistant);
            log.info("PlanAssistant inited");
            return planAssistant;
        }
    }
}
