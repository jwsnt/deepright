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
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.McpResult;
import ai.open.right.workflow.mcp.client.dimension.McpDimensionService;
import ai.open.right.workflow.mcp.client.impl.McpClientServiceImpl;
import ai.open.right.workflow.mcp.client.rewrtier.impl.McpRewriteServiceImpl;
import ai.open.right.workflow.mcp.client.trigger.impl.McpTriggerServiceImpl;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class McpToolsCallAssistantTest {

    @Test
    public void test() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("{}");
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        List<Map<String, Object>> maps = new ArrayList<>();
        maps.add(new HashMap<>());
        McpResult mcpToolsResult = new McpResult();
        mcpToolsResult.setResult(maps);
        mcpToolsResult.setClient("CLIENT");
        mcpToolsResult.setName("NAME");
        McpDimension mcpDimension = McpDimension.builder().build();
        EasyMock.expect(mcpClientService.toolsCall("CLIENT", "NAME", JsonUtils.read(workflowTask.getQuery(), Map.class), mcpDimension)).andReturn(mcpToolsResult).anyTimes();
        EasyMock.replay(mcpClientService);
        McpToolsCallAssistant mcpToolsCallAssistant = new McpToolsCallAssistant() {

            @Override
            protected McpDimension buildMcpDimension(String workflow, McpConfig mcpConfig, WorkflowTask workTask) {
                mcpDimension.bind(new String[]{"CLIENT", "NAME"});
                return mcpDimension;
            }

            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
            }
        };
        mcpToolsCallAssistant.setMcpClientService(mcpClientService);
        mcpToolsCallAssistant.setWorkflowConfigService(new WorkflowConfigServiceImpl() {
            @Override
            public WorkflowConfig buildMcpConfig(String workflow) {
                return new WorkflowConfig();
            }
        });
        mcpToolsCallAssistant.setMcpRewriteService(new McpRewriteServiceImpl());
        mcpToolsCallAssistant.setMcpTriggerService(new McpTriggerServiceImpl());
        mcpToolsCallAssistant.execute(new WorkflowConfig(), workflowTask);
        EasyMock.verify(mcpClientService);
    }

    @Test
    public void testWithConfig() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("{}");
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        List<Map<String, Object>> maps = new ArrayList<>();
        maps.add(new HashMap<>());
        McpResult mcpToolsResult = new McpResult();
        mcpToolsResult.setResult(maps);
        mcpToolsResult.setClient("CLIENT");
        mcpToolsResult.setName("NAME");
        McpDimension mcpDimension = McpDimension.builder().build();
        EasyMock.expect(mcpClientService.toolsCall("HELLO", "WORLD", JsonUtils.read(workflowTask.getQuery(), Map.class), mcpDimension)).andReturn(mcpToolsResult).anyTimes();
        EasyMock.replay(mcpClientService);
        McpToolsCallAssistant mcpToolsCallAssistant = new McpToolsCallAssistant() {

            @Override
            protected McpDimension buildMcpDimension(String workflow, McpConfig mcpConfig, WorkflowTask workTask) {
                mcpDimension.bind(new String[]{"HELLO", "WORLD"});
                return mcpDimension;
            }

            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
            }
        };
        mcpToolsCallAssistant.setMcpClientService(mcpClientService);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        McpConfig mcpConfig = new McpConfig();
        mcpConfig.setClient("HELLO");
        mcpConfig.setName("WORLD");
        workflowConfig.setMcpConfig(mcpConfig);
        mcpToolsCallAssistant.setMcpRewriteService(new McpRewriteServiceImpl());
        mcpToolsCallAssistant.setMcpTriggerService(new McpTriggerServiceImpl());
        mcpToolsCallAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(mcpClientService);
    }

    @Test
    public void testToolsCallWithClient() throws Exception {
        WorkflowConfig _workflowConfig = new WorkflowConfig();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        _workflowTask.setWorkflow("HELLO");
        _workflowTask.setUpstream("WORLD");
        _workflowTask.setQuery("{}");
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        List<Map<String, Object>> maps = new ArrayList<>();
        maps.add(new HashMap<>());
        McpResult mcpToolsResult = new McpResult();
        mcpToolsResult.setResult(maps);
        mcpToolsResult.setClient("CLIENT");
        mcpToolsResult.setName("NAME");
        McpDimension mcpDimension = McpDimension.builder().build();
        EasyMock.expect(mcpClientService.toolsCall("HELLO", "WORLD", JsonUtils.read(_workflowTask.getQuery(), Map.class), mcpDimension)).andReturn(mcpToolsResult).anyTimes();
        EasyMock.replay(mcpClientService);
        WorkflowConfigServiceImpl workflowConfigService = new WorkflowConfigServiceImpl() {

            @Override
            public WorkflowConfig config(WorkflowTask workflowTask, String workflow) throws Exception {
                return _workflowConfig;
            }

            @Override
            public WorkflowConfig config(WorkflowTask workflowTask) throws Exception {
                return _workflowConfig;
            }
        };
        McpToolsCallAssistant mcpToolsCallAssistant = new McpToolsCallAssistant() {
            @Override
            protected McpDimension buildMcpDimension(String workflow, McpConfig mcpConfig, WorkflowTask workTask) {
                mcpDimension.bind(new String[]{"HELLO", "WORLD"});
                return mcpDimension;
            }

            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
                Assert.assertEquals(_workflowConfig, workflowConfig);
                Assert.assertEquals(_workflowTask, workTask);
                Assert.assertEquals("[{}]", content);
            }
        };
        workflowConfigService.setNamesService(ObjectBuilder.buildNamesService());
        mcpToolsCallAssistant.setWorkflowConfigService(workflowConfigService);
        mcpToolsCallAssistant.setMcpClientService(mcpClientService);
        mcpToolsCallAssistant.setMcpRewriteService(new McpRewriteServiceImpl());
        mcpToolsCallAssistant.setMcpTriggerService(new McpTriggerServiceImpl());
        mcpToolsCallAssistant.execute(_workflowConfig, _workflowTask);
        EasyMock.verify(mcpClientService);
    }

    @Test
    public void testToolsCallWithClientAndName() throws Exception {
        WorkflowConfig _workflowConfig = new WorkflowConfig();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        _workflowTask.setWorkflow("HELLO");
        _workflowTask.setUpstream("WORLD");
        _workflowTask.setQuery("{}");
        McpConfig mcpConfig = new McpConfig();
        mcpConfig.setClient("CLIENT");
        mcpConfig.setName("NAME");
        _workflowConfig.setMcpConfig(mcpConfig);
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        List<Map<String, Object>> maps = new ArrayList<>();
        maps.add(new HashMap<>());
        McpResult mcpToolsResult = new McpResult();
        mcpToolsResult.setResult(maps);
        mcpToolsResult.setClient("CLIENT");
        mcpToolsResult.setName("NAME");
        McpDimension mcpDimension = McpDimension.builder().build();
        EasyMock.expect(mcpClientService.toolsCall("CLIENT", "NAME", JsonUtils.read(_workflowTask.getQuery(), Map.class), mcpDimension)).andReturn(mcpToolsResult).anyTimes();
        EasyMock.replay(mcpClientService);
        McpToolsCallAssistant mcpToolsCallAssistant = new McpToolsCallAssistant() {
            @Override
            protected McpDimension buildMcpDimension(String workflow, McpConfig mcpConfig, WorkflowTask workTask) {
                mcpDimension.bind(new String[]{"CLIENT", "NAME"});
                return mcpDimension;
            }

            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
                Assert.assertEquals(_workflowConfig, workflowConfig);
                Assert.assertEquals(_workflowTask, workTask);
                Assert.assertEquals("[{}]", content);
            }
        };
        mcpToolsCallAssistant.setMcpClientService(mcpClientService);
        mcpToolsCallAssistant.setMcpRewriteService(new McpRewriteServiceImpl());
        mcpToolsCallAssistant.setMcpTriggerService(new McpTriggerServiceImpl());
        mcpToolsCallAssistant.execute(_workflowConfig, _workflowTask);
        EasyMock.verify(mcpClientService);
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
        McpDimensionService mcpDimensionService = EasyMock.createMock(McpDimensionService.class);
        EasyMock.replay(mcpDimensionService, workflowConfigService, mcpListenerManager, mcpTriggerManager, mcpClientService, notifierManager, signalFactory);
        McpToolsCallAssistant.InitConfig mcpAssistant = new McpToolsCallAssistant.InitConfig();
        mcpAssistant.setWorkflowConfigService(workflowConfigService);
        mcpAssistant.setMcpDimensionService(mcpDimensionService);
        mcpAssistant.setMcpRewriteService(mcpListenerManager);
        mcpAssistant.setMcpTriggerService(mcpTriggerManager);
        mcpAssistant.setLlmQueryService(llmQueryServices);
        mcpAssistant.setMcpClientService(mcpClientService);
        mcpAssistant.setNotifierService(notifierManager);
        mcpAssistant.setSignalFactory(signalFactory);
        McpToolsCallAssistant empty = mcpAssistant.mcpToolsCallAssistant();
        Assert.assertEquals(empty.getWorkflowConfigService(), workflowConfigService);
        Assert.assertEquals(empty.getMcpTriggerService(), mcpTriggerManager);
        Assert.assertEquals(empty.getMcpRewriteService(), mcpListenerManager);
        Assert.assertEquals(empty.getLlmQueryService(), llmQueryServices);
        Assert.assertEquals(empty.getMcpClientService(), mcpClientService);
        Assert.assertEquals(empty.getNotifierService(), notifierManager);
        Assert.assertEquals(empty.getSignalFactory(), signalFactory);
        Assert.assertEquals(empty.getMcpDimensionService(), mcpDimensionService);
        EasyMock.verify(mcpDimensionService, mcpClientService, notifierManager, signalFactory);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = McpToolsCallAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = McpToolsCallAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
