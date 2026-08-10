package ai.open.right.workflow.flow.assistant;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.resource.ResourceFetcher;
import ai.open.right.workflow.flow.resource.ResourceResponse;
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

@Slf4j
@Setter
@Getter
public class ResourceAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-resource";

    protected ResourceFetcher resourceFetcher;

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        ResourceResponse response = this.resourceFetcher.fetch(workflowConfig.getResourceConfig(), workTask);
        Assert.notNull(response, "The response can not be empty");
        this.chainOr2Endpoint(workflowConfig, workTask, JsonUtils.write(response));
    }

    @ConditionalOnProperty(name = "resource.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected ResourceFetcher resourceFetcher;

        @Bean(ResourceAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = ResourceAssistant.WORKFLOW_NAME)
        public ResourceAssistant resourceAssistant() throws Exception {
            ResourceAssistant resourceAssistant = new ResourceAssistant();
            BeanUtils.copyProperties(this, resourceAssistant);
            log.info("ResourceAssistant inited");
            return resourceAssistant;
        }
    }
}
