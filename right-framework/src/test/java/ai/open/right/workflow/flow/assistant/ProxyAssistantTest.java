package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.config.WorkflowConfigService;
import com.google.common.collect.ImmutableMap;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.List;

public class ProxyAssistantTest {

    @Test
    public void test() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        EasyMock.expect(workflowConfigService.config("BIZ", "WORKFLOW")).andReturn(new WorkflowConfig()).anyTimes();
        EasyMock.replay(workflowConfigService);
        ProxyAssistant proxyAssistant = new ProxyAssistant() {

            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
                Assert.assertEquals("BIZ@WORKFLOW", workflowConfig.getChain());
                Assert.assertEquals("[A, B, C]", content);
            }
        };
        proxyAssistant.setWorkflowConfigService(workflowConfigService);
        ProxyAssistant.ProxyTask task = new ProxyAssistant.ProxyTask();
        task.setQuery(List.of("A", "B", "C").toString());
        task.setWorkflow("WORKFLOW");
        task.setBiz("BIZ");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(JsonUtils.write(task));
        proxyAssistant.execute(new WorkflowConfig(), workflowTask);
        EasyMock.verify(workflowConfigService);
    }

    @Test
    public void testWithMetadata() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        EasyMock.expect(workflowConfigService.config("BIZ", "WORKFLOW")).andReturn(new WorkflowConfig()).anyTimes();
        EasyMock.replay(workflowConfigService);
        ProxyAssistant proxyAssistant = new ProxyAssistant() {

            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
                Assert.assertEquals("BIZ@WORKFLOW", workflowConfig.getChain());
                Assert.assertEquals("[A, B, C]", content);
                Assert.assertEquals("B", workTask.getMetadata().get("A"));
            }
        };
        proxyAssistant.setWorkflowConfigService(workflowConfigService);
        ProxyAssistant.ProxyTask task = new ProxyAssistant.ProxyTask();
        task.setMetadata(ImmutableMap.of("A", "B"));
        task.setQuery(List.of("A", "B", "C").toString());
        task.setWorkflow("WORKFLOW");
        task.setBiz("BIZ");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(JsonUtils.write(task));
        proxyAssistant.execute(new WorkflowConfig(), workflowTask);
        EasyMock.verify(workflowConfigService);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testWithNull() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        EasyMock.expect(workflowConfigService.config("BIZ", "WORKFLOW")).andReturn(null).anyTimes();
        EasyMock.replay(workflowConfigService);
        ProxyAssistant proxyAssistant = new ProxyAssistant() {

            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
                Assert.assertEquals("BIZ@WORKFLOW", workflowConfig.getChain());
                Assert.assertEquals("[A, B, C]", content);
            }
        };
        proxyAssistant.setWorkflowConfigService(workflowConfigService);
        ProxyAssistant.ProxyTask task = new ProxyAssistant.ProxyTask();
        task.setQuery(List.of("A", "B", "C").toString());
        task.setWorkflow("WORKFLOW");
        task.setBiz("BIZ");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(JsonUtils.write(task));
        proxyAssistant.execute(new WorkflowConfig(), workflowTask);
        EasyMock.verify(workflowConfigService);
    }

    @Test
    public void testInit() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        ProxyAssistant.InitConfig proxyAssistant = new ProxyAssistant.InitConfig();
        proxyAssistant.setWorkflowConfigService(workflowConfigService);
        Assert.assertNotNull(proxyAssistant.proxyAssistant());
        Assert.assertEquals(workflowConfigService, proxyAssistant.proxyAssistant().getWorkflowConfigService());
    }
}
