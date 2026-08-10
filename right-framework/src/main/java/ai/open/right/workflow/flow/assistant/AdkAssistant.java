package ai.open.right.workflow.flow.assistant;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.adk.AdkService;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Setter
@Getter
@Slf4j
// ADK
public class AdkAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-adk";

    protected AdkService adkService;

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        String response = this.adkService.execute(workflowConfig, workTask);
        if (log.isDebugEnabled()) {
            log.debug("AdkAssistant response={}", response);
        }
        this.chainOr2Endpoint(workflowConfig, workTask, response);
    }

    @ConditionalOnProperty(name = "adk.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected AdkService adkService;

        @Bean(AdkAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = AdkAssistant.WORKFLOW_NAME)
        public AdkAssistant adkAssistant() throws Exception {
            AdkAssistant adkAssistant = new AdkAssistant();
            BeanUtils.copyProperties(this, adkAssistant);
            log.info("AdkAssistant inited");
            return adkAssistant;
        }
    }
}
