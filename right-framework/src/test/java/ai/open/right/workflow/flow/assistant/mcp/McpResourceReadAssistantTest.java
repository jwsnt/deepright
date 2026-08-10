package ai.open.right.workflow.flow.assistant.mcp;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.McpConfig;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.config.WorkflowConfigService;
import ai.open.right.workflow.flow.config.impl.WorkflowConfigServiceImpl;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.mcp.client.impl.McpClientServiceImpl;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.McpResult;
import ai.open.right.workflow.mcp.client.McpRuntime;
import ai.open.right.workflow.mcp.client.rewrtier.impl.McpRewriteServiceImpl;
import ai.open.right.workflow.mcp.client.trigger.impl.McpTriggerServiceImpl;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class McpResourceReadAssistantTest {

    @Test
    public void test() throws Exception {

        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("{}");
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        McpResult<String> mcpResult = new McpResult<String>();
        mcpResult.setResult("HELLO");
        McpDimension mcpDimension = McpDimension.builder().build();
        EasyMock.expect(mcpClientService.resourcesRead(workflowTask.getWorkflow(), "http://", mcpDimension)).andReturn(mcpResult).anyTimes();
        EasyMock.replay(mcpClientService);
        McpResourceReadAssistant mcpResourceGetAssistant = new McpResourceReadAssistant() {

            @Override
            protected McpDimension buildMcpDimension(String workflow, McpConfig mcpConfig, WorkflowTask workTask) {
                return mcpDimension;
            }

            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
            }
        };
        mcpResourceGetAssistant.setMcpClientService(mcpClientService);
        workflowTask.setQuery("{\"uri\":\"http://\"}");
        WorkflowConfigServiceImpl workflowConfigService = new WorkflowConfigServiceImpl() {
            @Override
            public WorkflowConfig config(WorkflowTask workflowTask, String workflow) throws Exception {
                return new WorkflowConfig();
            }

            @Override
            public WorkflowConfig config(WorkflowTask workflowTask) throws Exception {
                return new WorkflowConfig();
            }
        };
        workflowConfigService.setNamesService(ObjectBuilder.buildNamesService());
        mcpResourceGetAssistant.setWorkflowConfigService(workflowConfigService);
        mcpResourceGetAssistant.setMcpRewriteService(new McpRewriteServiceImpl());
        mcpResourceGetAssistant.setMcpTriggerService(new McpTriggerServiceImpl());
        mcpResourceGetAssistant.execute(new WorkflowConfig(), workflowTask);
        EasyMock.verify(mcpClientService);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testWithOutUri() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("{}");
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        McpResult<String> mcpResult = new McpResult<String>();
        mcpResult.setResult("HELLO");
        McpDimension mcpDimension = McpDimension.builder().build();
        EasyMock.expect(mcpClientService.resourcesRead(workflowTask.getWorkflow(), null, mcpDimension)).andReturn(mcpResult).anyTimes();
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        EasyMock.expect(workflowConfigService.config(workflowTask, "UNKNOWN")).andReturn(workflowConfig).anyTimes();
        EasyMock.replay(mcpClientService, workflowConfigService);
        McpResourceReadAssistant mcpResourceGetAssistant = new McpResourceReadAssistant() {

            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
            }

            @Override
            protected McpDimension buildMcpDimension(McpDimension mcpDimension, WorkflowTask workTask) throws Exception {
                return mcpDimension;
            }
        };
        mcpResourceGetAssistant.setWorkflowConfigService(workflowConfigService);
        mcpResourceGetAssistant.setMcpClientService(mcpClientService);
        mcpResourceGetAssistant.setMcpRewriteService(new McpRewriteServiceImpl());
        mcpResourceGetAssistant.setMcpTriggerService(new McpTriggerServiceImpl());
        mcpResourceGetAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(mcpClientService, workflowConfigService);
    }

    @Test
    public void testWithConfig() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("{}");
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        McpRuntime mcpRuntime = McpRuntime.builder().build();
        McpResult<String> mcpResult = new McpResult<String>();
        mcpResult.setResult("HELLO WORLD");
        McpDimension mcpDimension = McpDimension.builder().build();
        EasyMock.expect(mcpClientService.resourcesRead("HELLO", "WORLD", mcpRuntime, mcpDimension)).andReturn(mcpResult).anyTimes();
        EasyMock.replay(mcpClientService);
        McpResourceReadAssistant mcpResourceReadAssistant = new McpResourceReadAssistant() {

            @Override
            protected McpDimension buildMcpDimension(String workflow, McpConfig mcpConfig, WorkflowTask workTask) {
                mcpDimension.bind(new String[]{"HELLO", "WORLD"});
                return mcpDimension;
            }

            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
            }

            @Override
            protected McpRuntime buildMcpRuntime(McpConfig mcpConfig, WorkflowTask workTask) {
                return mcpRuntime;
            }
        };
        mcpResourceReadAssistant.setMcpClientService(mcpClientService);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        McpConfig mcpConfig = new McpConfig();
        mcpConfig.setClient("HELLO");
        mcpConfig.setName("WORLD");
        workflowConfig.setMcpConfig(mcpConfig);
        mcpResourceReadAssistant.setMcpClientService(mcpClientService);
        mcpResourceReadAssistant.setMcpRewriteService(new McpRewriteServiceImpl());
        mcpResourceReadAssistant.setMcpTriggerService(new McpTriggerServiceImpl());
        mcpResourceReadAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(mcpClientService);
    }

    @Test
    public void testWithMcpRuntime() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        McpResourceReadAssistant mcpResourceReadAssistant = new McpResourceReadAssistant() {
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
            }
        };
        McpConfig mcpConfig = new McpConfig();
        mcpConfig.setTimeout(1000);
        mcpConfig.setDynamic("DYNAMIC");
        McpRuntime mcpRuntime = mcpResourceReadAssistant.buildMcpRuntime(mcpConfig, workflowTask);
        Assert.assertEquals(workflowTask, mcpRuntime.getWorkTask());
        Assert.assertEquals("DYNAMIC", mcpRuntime.getDynamic());
        Assert.assertEquals(Integer.valueOf(1000), mcpRuntime.getTimeout());
    }

    @Test
    public void testInit() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        McpRewriteServiceImpl mcpListenerManager = EasyMock.createMock(McpRewriteServiceImpl.class);
        McpTriggerServiceImpl mcpTriggerManager = EasyMock.createMock(McpTriggerServiceImpl.class);
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);
        Map<String, LLMQueryService> llmQueryServices = new HashMap<>();
        EasyMock.replay(workflowConfigService, mcpListenerManager, mcpTriggerManager, mcpClientService, notifierManager, signalFactory);
        McpResourceReadAssistant.InitConfig mcpAssistant = new McpResourceReadAssistant.InitConfig();
        mcpAssistant.setWorkflowConfigService(workflowConfigService);
        mcpAssistant.setMcpRewriteService(mcpListenerManager);
        mcpAssistant.setMcpTriggerService(mcpTriggerManager);
        mcpAssistant.setLlmQueryService(llmQueryServices);
        mcpAssistant.setMcpClientService(mcpClientService);
        mcpAssistant.setNotifierService(notifierManager);
        mcpAssistant.setSignalFactory(signalFactory);
        McpResourceReadAssistant empty = mcpAssistant.mcpResourceReadAssistant();
        Assert.assertEquals(empty.getWorkflowConfigService(), workflowConfigService);
        Assert.assertEquals(empty.getMcpTriggerService(), mcpTriggerManager);
        Assert.assertEquals(empty.getMcpRewriteService(), mcpListenerManager);
        Assert.assertEquals(empty.getLlmQueryService(), llmQueryServices);
        Assert.assertEquals(empty.getMcpClientService(), mcpClientService);
        Assert.assertEquals(empty.getNotifierService(), notifierManager);
        Assert.assertEquals(empty.getSignalFactory(), signalFactory);
        EasyMock.verify(mcpClientService, notifierManager, signalFactory);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = McpResourceReadAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = McpResourceReadAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
