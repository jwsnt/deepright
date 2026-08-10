package ai.open.right.workflow.flow.assistant.pubsub;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.assistant.PassageAssistant;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.pubsub.PubSubService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
@Setter
@Getter
// 发布/订阅
public class SubEventAssistant extends PassageAssistant {

    public static final String WORKFLOW_NAME = "def-subEvent";

    protected PubSubService pubSubService;

    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        String content = this.pubSubService.sub(workflowConfig.getPubSubConfig(), workTask);
        this.chainOr2Endpoint(workflowConfig, workTask, content);
    }


    @ConditionalOnProperty(name = "pubsub.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected PubSubService pubSubService;

        @Bean(SubEventAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = SubEventAssistant.WORKFLOW_NAME)
        public SubEventAssistant subEventAssistant() throws Exception {
            SubEventAssistant subEventAssistant = new SubEventAssistant();
            BeanUtils.copyProperties(this, subEventAssistant);
            log.info("SubEventAssistant inited");
            return subEventAssistant;
        }
    }
}
