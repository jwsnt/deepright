package ai.open.right.workflow.flow.llm.rag.mcp;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.mcp.client.McpResult;
import ai.open.right.workflow.mcp.client.McpRuntime;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.dimension.McpDimensionService;
import ai.open.right.workflow.mcp.client.impl.McpClientServiceImpl;
import ai.open.right.workflow.mcp.client.rewrtier.impl.McpRewriteServiceImpl;
import ai.open.right.workflow.mcp.client.trigger.impl.McpTriggerServiceImpl;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

public class RagMcpServiceTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = RagMcpService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = RagMcpService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testWithOutMcp() throws Exception {
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        McpRuntime mcpRuntime = McpRuntime.builder().build();
        McpResult<String> mcpResult = new McpResult<String>();
        mcpResult.setResult("WORLD3");
        McpDimension mcpDimension = McpDimension.builder().build();
        EasyMock.expect(mcpClientService.resourcesRead("WORLD1", "WORLD2", mcpRuntime, mcpDimension)).andReturn(mcpResult).anyTimes();
        EasyMock.replay(mcpClientService);
        RagMcpService ragMcpService = new RagMcpService();
        RagConfig ragConfig = new RagConfig();
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        ragMcpService.setMcpClientService(mcpClientService);
        ragMcpService.rag(ragConfig, ragData);
        Assert.assertEquals("HELLO", ragData.getPrompt());
        EasyMock.verify(mcpClientService);
    }

    @Test
    public void testWithMcp() throws Exception {
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        McpRuntime mcpRuntime = McpRuntime.builder().build();
        McpResult<String> mcpResult = new McpResult<String>();
        mcpResult.setResult("WORLD3");
        McpDimension mcpDimension = McpDimension.builder().build();
        EasyMock.expect(mcpClientService.resourcesRead("HELLO", "WORLD2", mcpRuntime, mcpDimension)).andReturn(mcpResult).anyTimes();
        EasyMock.replay(mcpClientService);
        RagMcpService ragMcpService = new RagMcpService() {
            @Override
            protected McpDimension buildMcpDimension(RagMcpConfig ragMcpConfig, RagData ragData) {
                // WORLD应该不被使用
                mcpDimension.bind(new String[]{"HELLO", "WORLD"});
                return mcpDimension;
            }

            @Override
            protected McpRuntime buildMcpRuntime(RagMcpConfig ragMcpConfig, RagData ragData) {
                return mcpRuntime;
            }
        };
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        ragMcpService.setExecutorService(executorService);
        RagConfig ragConfig = new RagConfig();
        RagMcpConfig ragMcpConfig = new RagMcpConfig();
        ragMcpConfig.setClient("WORLD1");
        ragMcpConfig.setName("WORLD2");
        ragConfig.setRagMcpConfig(ragMcpConfig);
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        ragMcpService.setMcpClientService(mcpClientService);
        ragMcpService.setMcpTriggerService(new McpTriggerServiceImpl());
        ragMcpService.setMcpRewriteService(new McpRewriteServiceImpl());
        ragMcpService.setTimeout(1000);
        ragMcpService.rag(ragConfig, ragData).run();
        Assert.assertEquals("UNKNOWNWORLD3", ragData.getQuery().getQuery());
        EasyMock.verify(mcpClientService);
        executorService.close();
    }

    @Test
    public void testWithException() throws Exception {
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        McpRuntime mcpRuntime = McpRuntime.builder().build();
        McpDimension mcpDimension = McpDimension.builder().build();
        EasyMock.expect(mcpClientService.resourcesRead("WORLD1", "WORLD2", mcpRuntime, mcpDimension)).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(mcpClientService);
        RagMcpService ragMcpService = new RagMcpService() {
            @Override
            protected McpDimension buildMcpDimension(RagMcpConfig ragMcpConfig, RagData ragData) {
                return mcpDimension;
            }

            protected McpRuntime buildMcpRuntime(RagMcpConfig ragMcpConfig, RagData ragData) {
                return mcpRuntime;
            }
        };
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        ragMcpService.setExecutorService(executorService);
        RagConfig ragConfig = new RagConfig();
        RagMcpConfig ragMcpConfig = new RagMcpConfig();
        ragMcpConfig.setClient("WORLD1");
        ragMcpConfig.setName("WORLD2");
        ragConfig.setRagMcpConfig(ragMcpConfig);
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        ragMcpService.setMcpClientService(mcpClientService);
        ragMcpService.setMcpRewriteService(new McpRewriteServiceImpl());
        ragMcpService.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN", ragData.getQuery().getQuery());
        EasyMock.verify(mcpClientService);
        executorService.close();
    }

    @Test
    public void testBuildMpcRuntime() throws Exception {
        RagMcpService ragMcpService = new RagMcpService();
        RagConfig ragConfig = new RagConfig();
        RagMcpConfig ragMcpConfig = new RagMcpConfig();
        ragMcpConfig.setDynamic("DYNAMIC");
        ragMcpConfig.setClient("WORLD1");
        ragMcpConfig.setName("WORLD2");
        ragMcpConfig.setTimeout(1000);
        ragConfig.setRagMcpConfig(ragMcpConfig);
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        McpRuntime mcpRuntime = ragMcpService.buildMcpRuntime(ragMcpConfig, ragData);
        Assert.assertEquals(ragData.getQuery(), mcpRuntime.getWorkTask());
        Assert.assertEquals("DYNAMIC", mcpRuntime.getDynamic());
        Assert.assertEquals(Integer.valueOf(1000), mcpRuntime.getTimeout());
        Assert.assertEquals(Integer.valueOf(1000), mcpRuntime.getTimeout(20000));
    }

    @Test(expected = NullPointerException.class)
    public void testWithStopOnFailed() throws Exception {
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        McpRuntime mcpRuntime = McpRuntime.builder().build();
        McpDimension mcpDimension = McpDimension.builder().build();
        EasyMock.expect(mcpClientService.resourcesRead("WORLD1", "WORLD2", mcpRuntime, mcpDimension)).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(mcpClientService);
        RagMcpService ragMcpService = new RagMcpService() {
            @Override
            protected McpDimension buildMcpDimension(RagMcpConfig ragMcpConfig, RagData ragData) {
                return mcpDimension;
            }

            @Override
            protected McpRuntime buildMcpRuntime(RagMcpConfig ragMcpConfig, RagData ragData) {
                return mcpRuntime;
            }
        };
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        ragMcpService.setExecutorService(executorService);
        RagConfig ragConfig = new RagConfig();
        ragConfig.setStopOnFailed(true);
        RagMcpConfig ragMcpConfig = new RagMcpConfig();
        ragMcpConfig.setClient("WORLD1");
        ragMcpConfig.setName("WORLD2");
        ragConfig.setRagMcpConfig(ragMcpConfig);
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        ragMcpService.setMcpClientService(mcpClientService);
        try {
            ragMcpService.rag(ragConfig, ragData).run();
        } finally {
            EasyMock.verify(mcpClientService);
            executorService.close();
        }
    }

    @Test
    public void testWithDynamicURI() throws Exception {
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        McpRuntime mcpRuntime = McpRuntime.builder().build();
        McpResult<String> mcpResult = new McpResult<String>();
        mcpResult.setResult("WORLD3");
        McpDimension mcpDimension = McpDimension.builder().build();
        EasyMock.expect(mcpClientService.resourcesRead("HELLO", "DYNAMIC", mcpRuntime, mcpDimension)).andReturn(mcpResult).anyTimes();
        EasyMock.replay(mcpClientService);
        RagMcpService ragMcpService = new RagMcpService() {
            @Override
            protected McpDimension buildMcpDimension(RagMcpConfig ragMcpConfig, RagData ragData) {
                // WORLD应该不被使用
                mcpDimension.bind(new String[]{"HELLO", "WORLD"});
                return mcpDimension;
            }

            @Override
            protected McpRuntime buildMcpRuntime(RagMcpConfig ragMcpConfig, RagData ragData) {
                return mcpRuntime;
            }
        };
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        ragMcpService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("DYNAMIC"));
        ragMcpService.setExecutorService(executorService);
        RagConfig ragConfig = new RagConfig();
        RagMcpConfig ragMcpConfig = new RagMcpConfig();
        ragMcpConfig.setDynamic("DYNAMIC");
        ragMcpConfig.setClient("WORLD1");
        ragConfig.setRagMcpConfig(ragMcpConfig);
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        ragMcpService.setMcpClientService(mcpClientService);
        ragMcpService.setMcpTriggerService(new McpTriggerServiceImpl());
        ragMcpService.setMcpRewriteService(new McpRewriteServiceImpl());
        ragMcpService.setTimeout(1000);
        ragMcpService.rag(ragConfig, ragData).run();
        Assert.assertEquals("UNKNOWNWORLD3", ragData.getQuery().getQuery());
        EasyMock.verify(mcpClientService);
        executorService.close();
    }

    @Test
    public void testBuildMcpDimension() throws Exception {
        RagMcpService ragMcpService = new RagMcpService() {
            protected McpDimension buildMcpDimension(McpDimension mcpDimension, RagData ragData) throws Exception {
                mcpDimension.bind(new String[]{"A", "B"});
                return mcpDimension;
            }
        };
        RagMcpConfig ragMcpConfig = new RagMcpConfig();
        ragMcpConfig.setDynamic("DYNAMIC");
        ragMcpConfig.setClient("WORLD1");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        McpDimension mcpDimension = ragMcpService.buildMcpDimension(ragMcpConfig, ragData);
        Assert.assertEquals("UNKNOWN-UNKNOWN-UNKNOWN", mcpDimension.getDimension());
        Assert.assertEquals(ragMcpConfig, mcpDimension.getMcpConfig());
        Assert.assertEquals("UNKNOWN", mcpDimension.getDevice());
        Assert.assertEquals("UNKNOWN", mcpDimension.getBiz());
        Assert.assertEquals("UNKNOWN", mcpDimension.getChat());
        Assert.assertEquals("UNKNOWN", mcpDimension.getWorkflow());
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        ExecutorService executorService = EasyMock.createMock(ExecutorService.class);
        McpClientServiceImpl clientService = EasyMock.createMock(McpClientServiceImpl.class);
        McpRewriteServiceImpl listenerManager = EasyMock.createMock(McpRewriteServiceImpl.class);
        McpTriggerServiceImpl triggerManager = EasyMock.createMock(McpTriggerServiceImpl.class);
        McpDimensionService mcpDimensionService = EasyMock.createMock(McpDimensionService.class);
        EasyMock.replay(mcpDimensionService, executorService, clientService, listenerManager, triggerManager);
        RagMcpService.InitConfig service = new RagMcpService.InitConfig();
        service.setNotifierService(notifierManager);
        service.setExecutorService(executorService);
        service.setMcpDimensionService(mcpDimensionService);
        service.setMcpRewriteService(listenerManager);
        service.setMcpTriggerService(triggerManager);
        service.setMcpClientService(clientService);
        service.setTimeout4Condition(10086);
        service.setTimeout(1000);
        service.setTimeout4Llm(1000);
        RagMcpService empty = service.ragMcpService();
        Assert.assertEquals(Integer.valueOf(10086), empty.getTimeout4Condition());
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(executorService, empty.getExecutorService());
        Assert.assertEquals(Integer.valueOf(1000), empty.getTimeout4Llm());
        Assert.assertEquals(Integer.valueOf(1000), empty.getTimeout());
        Assert.assertEquals(clientService, empty.getMcpClientService());
        Assert.assertEquals(listenerManager, empty.getMcpRewriteService());
        Assert.assertEquals(triggerManager, empty.getMcpTriggerService());
        Assert.assertEquals(mcpDimensionService, empty.getMcpDimensionService());
        EasyMock.verify(mcpDimensionService, executorService, clientService, listenerManager, triggerManager);
    }

    @Test
    public void testBuildMcpDimension2() throws Exception {
        RagMcpService ragMcpService = new RagMcpService();
        RagMcpConfig ragMcpConfig = new RagMcpConfig();
        ragMcpConfig.setDynamic("DYNAMIC");
        ragMcpConfig.setClient("WORLD1");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        McpDimension md = McpDimension.builder().build();
        McpDimensionService mcpDimensionService = EasyMock.createMock(McpDimensionService.class);
        EasyMock.expect(mcpDimensionService.buildDimension(md, ragData.getQuery())).andReturn(md).anyTimes();
        EasyMock.replay(mcpDimensionService);
        ragMcpService.setMcpDimensionService(mcpDimensionService);
        McpDimension mcpDimension = ragMcpService.buildMcpDimension(md, ragData);
        Assert.assertNotNull(mcpDimension);
        EasyMock.verify(mcpDimensionService);
    }
    @Test
    public void testAllowedNoMcp() throws Exception {
        RagMcpService service = new RagMcpService();
        RagConfig config = new RagConfig();
        Assert.assertFalse(service.allowed(config, null));
    }

    @Test(expected = Exception.class)
    public void testMcpFutureCallEmptyName() throws Exception {
        RagMcpService service = new RagMcpService();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        McpDimensionService mcpDimensionService = EasyMock.createMock(McpDimensionService.class);
        EasyMock.expect(mcpDimensionService.buildDimension(EasyMock.anyObject(McpDimension.class), llmQuery)).andReturn(McpDimension.builder().build()).anyTimes();
        service.setMcpDimensionService(mcpDimensionService);
        EasyMock.replay(mcpDimensionService);
        RagConfig config = new RagConfig();
        config.setRagMcpConfig(new RagMcpConfig());
        RagData data = RagData.builder().query(llmQuery).build();
        RagMcpService.McpFuture future = service.new McpFuture(config, data);
        future.call();
        EasyMock.verify(mcpDimensionService);
    }
}
