package ai.open.right.workflow.flow.assistant.mcp;

import ai.open.right.ObjectBuilder;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.McpConfig;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.config.WorkflowConfigService;
import ai.open.right.workflow.flow.config.impl.WorkflowConfigServiceImpl;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.mcp.client.McpResult;
import ai.open.right.workflow.mcp.client.McpRuntime;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.impl.McpClientServiceImpl;
import ai.open.right.workflow.mcp.client.rewrtier.impl.McpRewriteServiceImpl;
import ai.open.right.workflow.mcp.client.trigger.impl.McpTriggerServiceImpl;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class McpPromptGetAssistantTest {

    @Test
    public void test() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("{}");
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        McpResult<List<History>> result = new McpResult<List<History>>();
        result.setResult(Arrays.asList(new History()));
        McpDimension mcpDimension = McpDimension.builder().build();
        EasyMock.expect(mcpClientService.promptGet("A", "B", JsonUtils.read(workflowTask.getQuery(), Map.class), mcpDimension)).andReturn(result).anyTimes();
        McpPromptGetAssistant mcpPromptGetAssistant = new McpPromptGetAssistant() {

            @Override
            protected McpDimension buildMcpDimension(String workflow, McpConfig mcpConfig, WorkflowTask workTask) throws Exception {
                mcpDimension.bind(new String[]{"A", "B"});
                return mcpDimension;
            }

            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
            }
        };
        mcpPromptGetAssistant.setMcpClientService(mcpClientService);
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
        EasyMock.replay(mcpClientService);
        workflowConfigService.setNamesService(ObjectBuilder.buildNamesService());
        mcpPromptGetAssistant.setWorkflowConfigService(workflowConfigService);
        mcpPromptGetAssistant.setMcpRewriteService(new McpRewriteServiceImpl());
        mcpPromptGetAssistant.setMcpTriggerService(new McpTriggerServiceImpl());
        mcpPromptGetAssistant.setMcpClientService(mcpClientService);
        mcpPromptGetAssistant.execute(new WorkflowConfig(), workflowTask);
        EasyMock.verify(mcpClientService);
    }

    @Test
    public void testWithConfig() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("{}");
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        McpRuntime mcpRuntime = McpRuntime.builder().build();
        McpResult<List<History>> result = new McpResult<List<History>>();
        result.setResult(Arrays.asList(new History()));
        McpDimension mcpDimension = McpDimension.builder().build();
        EasyMock.expect(mcpClientService.promptGet("HELLO", "WORLD", JsonUtils.read(workflowTask.getQuery(), Map.class), mcpRuntime, mcpDimension)).andReturn(result).anyTimes();
        EasyMock.replay(mcpClientService);
        McpPromptGetAssistant mcpPromptGetAssistant = new McpPromptGetAssistant() {

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
        mcpPromptGetAssistant.setMcpClientService(mcpClientService);
        mcpPromptGetAssistant.setMcpRewriteService(new McpRewriteServiceImpl());
        mcpPromptGetAssistant.setMcpTriggerService(new McpTriggerServiceImpl());
        WorkflowConfig workflowConfig = new WorkflowConfig();
        McpConfig mcpConfig = new McpConfig();
        mcpConfig.setClient("HELLO");
        mcpConfig.setName("WORLD");
        workflowConfig.setMcpConfig(mcpConfig);
        mcpPromptGetAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(mcpClientService);
    }

    @Test
    public void testWithMcpRuntime() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        McpPromptGetAssistant mcpPromptGetAssistant = new McpPromptGetAssistant() {
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
            }
        };
        McpConfig mcpConfig = new McpConfig();
        mcpConfig.setTimeout(1000);
        mcpConfig.setDynamic("DYNAMIC");
        McpRuntime mcpRuntime = mcpPromptGetAssistant.buildMcpRuntime(mcpConfig, workflowTask);
        Assert.assertEquals(workflowTask, mcpRuntime.getWorkTask());
        Assert.assertEquals("DYNAMIC", mcpRuntime.getDynamic());
        Assert.assertEquals(Integer.valueOf(1000), mcpRuntime.getTimeout());
    }

    @Test
    public void testBuildMcpDimension() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        McpConfig mcpConfig = new McpConfig();
        McpPromptGetAssistant mcpAssistant = new McpPromptGetAssistant() {
            protected McpDimension buildMcpDimension(McpDimension mcpDimension, WorkflowTask workTask) throws Exception {
                return mcpDimension;
            }
        };
        McpDimension mcpDimension = mcpAssistant.buildMcpDimension("W1", mcpConfig, workflowTask);
        Assert.assertEquals("W1", mcpDimension.getWorkflow());
        Assert.assertEquals(workflowTask.getChat(), mcpDimension.getChat());
        Assert.assertEquals(workflowTask.getBiz(), mcpDimension.getBiz());
        Assert.assertEquals(workflowTask.getUserContext().getDevice(), mcpDimension.getDevice());
        Assert.assertEquals(workflowTask.getDimension(), mcpDimension.getDimension());
        Assert.assertEquals(mcpConfig, mcpDimension.getMcpConfig());
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
        McpPromptGetAssistant.InitConfig mcpAssistant = new McpPromptGetAssistant.InitConfig();
        mcpAssistant.setWorkflowConfigService(workflowConfigService);
        mcpAssistant.setMcpRewriteService(mcpListenerManager);
        mcpAssistant.setMcpTriggerService(mcpTriggerManager);
        mcpAssistant.setLlmQueryService(llmQueryServices);
        mcpAssistant.setMcpClientService(mcpClientService);
        mcpAssistant.setNotifierService(notifierManager);
        mcpAssistant.setSignalFactory(signalFactory);
        McpPromptGetAssistant empty = mcpAssistant.mcpPromptGetAssistant();
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
        Object object = McpPromptGetAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = McpPromptGetAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
