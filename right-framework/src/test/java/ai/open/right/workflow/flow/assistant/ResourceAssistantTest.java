package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.resource.ResourceConfig;
import ai.open.right.workflow.flow.resource.ResourceFetcher;
import ai.open.right.workflow.flow.resource.ResourceResponse;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

public class ResourceAssistantTest {

    private ResourceAssistant resourceAssistant;
    private ResourceFetcher resourceFetcher;

    @BeforeEach
    public void setUp() {
        resourceAssistant = new ResourceAssistant();
        resourceFetcher = mock(ResourceFetcher.class);
        resourceAssistant.setResourceFetcher(resourceFetcher);
    }

    @Test
    public void testExecute() throws Exception {
        WorkflowConfig config = new WorkflowConfig();
        ResourceConfig resourceConfig = new ResourceConfig();
        config.setResourceConfig(resourceConfig);
        
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ResourceResponse response = new ResourceResponse();
        response.setContent("{\"status\":\"success\"}");
        
        when(resourceFetcher.fetch(resourceConfig, task)).thenReturn(response);

        final boolean[] called = {false};
        ResourceAssistant assistant = new ResourceAssistant() {
            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
                called[0] = true;
                Assertions.assertNotNull(content);
            }
        };
        assistant.setResourceFetcher(resourceFetcher);
        
        assistant.execute(config, task);
        
        Assertions.assertTrue(called[0]);
    }

    @Test
    public void testInitConfig() throws Exception {
        ResourceAssistant.InitConfig initConfig = new ResourceAssistant.InitConfig();
        initConfig.setResourceFetcher(resourceFetcher);
        ResourceAssistant assistant = initConfig.resourceAssistant();
        Assertions.assertNotNull(assistant);
        Assertions.assertEquals(resourceFetcher, assistant.getResourceFetcher());
    }
}
