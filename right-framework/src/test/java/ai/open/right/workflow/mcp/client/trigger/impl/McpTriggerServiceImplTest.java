package ai.open.right.workflow.mcp.client.trigger.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.McpConfig;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.McpResult;
import ai.open.right.workflow.mcp.client.trigger.BaseTrigger;
import ai.open.right.workflow.mcp.client.trigger.McpTrigger;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.Collections;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.atomic.AtomicInteger;

public class McpTriggerServiceImplTest {

    @Test
    public void testTrigger() throws Exception {
        McpTriggerServiceImpl mcpTriggerManager = new McpTriggerServiceImpl();
        mcpTriggerManager.setTriggers(Collections.singletonMap("HELLO", new BaseTrigger()));
        McpConfig mcpConfig = new McpConfig();
        mcpConfig.setTrigger("HELLO");
        McpDimension _mcpDimension = McpDimension.builder()
                .mcpConfig(mcpConfig)
                .build();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        mcpTriggerManager.beforeToolsCall(_mcpDimension, new HashMap<>(), workflowTask);
    }

    @Test
    public void testTriggerWithPromptGet() throws Exception {
        McpTriggerServiceImpl mcpTriggerManager = new McpTriggerServiceImpl();
        mcpTriggerManager.setTriggers(Collections.singletonMap("HELLO", new BaseTrigger()));
        McpConfig mcpConfig = new McpConfig();
        mcpConfig.setTrigger("HELLO");
        McpDimension _mcpDimension = McpDimension.builder()
                .mcpConfig(mcpConfig)
                .build();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        mcpTriggerManager.beforePromptGet(_mcpDimension, new HashMap<>(), workflowTask);
    }

    @Test
    public void testTriggerWithResourcesRead() throws Exception {
        McpTriggerServiceImpl mcpTriggerManager = new McpTriggerServiceImpl();
        mcpTriggerManager.setTriggers(Collections.singletonMap("HELLO", new BaseTrigger()));
        McpConfig mcpConfig = new McpConfig();
        mcpConfig.setTrigger("HELLO");
        McpDimension _mcpDimension = McpDimension.builder()
                .mcpConfig(mcpConfig)
                .build();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        mcpTriggerManager.beforeResourcesRead(_mcpDimension, "URI", workflowTask);
    }

    @Test
    public void testTrigger2() throws Exception {
        McpConfig _mcpConfig = new McpConfig();
        _mcpConfig.setRewriter("HELLO");
        McpDimension _mcpDimension = McpDimension.builder()
                .mcpConfig(_mcpConfig)
                .build();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        McpResult _mcpToolsResult = new McpResult();
        _mcpToolsResult.setClient("CLIENT");
        _mcpToolsResult.setName("NAME");
        McpTriggerServiceImpl mcpTriggerManager = new McpTriggerServiceImpl();
        Map<String, McpTrigger> mcpToolsTrigger = new HashMap<>();
        mcpToolsTrigger.put("HELLO", new BaseTrigger() {

            @Override
            public void beforeToolsCall(McpDimension mcpDimension, Map<String, Object> arg, WorkflowTask workTask) throws Exception {
                Assert.assertEquals(mcpDimension.getMcpConfig(), _mcpConfig);
                Assert.assertEquals(mcpDimension.getChat(), "UNKNOWN");
                Assert.assertEquals(mcpDimension.getWorkflow(), _workflowTask);
                Assert.assertEquals(mcpDimension.getBiz(), "BIZ");
                Assert.assertEquals(_mcpToolsResult, workTask);
            }
        });
        mcpTriggerManager.setTriggers(mcpToolsTrigger);
        mcpTriggerManager.beforeToolsCall(_mcpDimension, new HashMap<>(), _workflowTask);
    }

    @Test
    public void testTrigger2WithPromptGet() throws Exception {
        McpConfig _mcpConfig = new McpConfig();
        _mcpConfig.setRewriter("HELLO");
        McpDimension _mcpDimension = McpDimension.builder()
                .mcpConfig(_mcpConfig)
                .build();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        McpResult _mcpToolsResult = new McpResult();
        _mcpToolsResult.setClient("CLIENT");
        _mcpToolsResult.setName("NAME");
        McpTriggerServiceImpl mcpTriggerManager = new McpTriggerServiceImpl();
        Map<String, McpTrigger> mcpToolsTrigger = new HashMap<>();
        mcpToolsTrigger.put("HELLO", new BaseTrigger() {
            @Override
            public void beforePromptGet(McpDimension mcpDimension, Map<String, Object> args, WorkflowTask workTask) throws Exception {
                Assert.assertEquals(mcpDimension.getMcpConfig(), _mcpConfig);
                Assert.assertEquals(mcpDimension.getChat(), "UNKNOWN");
                Assert.assertEquals(mcpDimension.getWorkflow(), _workflowTask);
                Assert.assertEquals(mcpDimension.getBiz(), "BIZ");
                Assert.assertEquals(_mcpToolsResult, workTask);
            }
        });
        mcpTriggerManager.setTriggers(mcpToolsTrigger);
        mcpTriggerManager.beforePromptGet(_mcpDimension, new HashMap<>(), _workflowTask);
    }

    @Test
    public void testTrigger3WithResourcesRead() throws Exception {
        McpConfig _mcpConfig = new McpConfig();
        _mcpConfig.setRewriter("HELLO");
        McpDimension _mcpDimension = McpDimension.builder()
                .mcpConfig(_mcpConfig)
                .build();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        McpResult _mcpToolsResult = new McpResult();
        _mcpToolsResult.setClient("CLIENT");
        _mcpToolsResult.setName("NAME");
        McpTriggerServiceImpl mcpTriggerManager = new McpTriggerServiceImpl();
        Map<String, McpTrigger> mcpToolsTrigger = new HashMap<>();
        mcpToolsTrigger.put("HELLO", new BaseTrigger() {

            public void beforeResourcesRead(McpDimension mcpDimension, WorkflowTask workTask) throws Exception {
                Assert.assertEquals(mcpDimension.getMcpConfig(), _mcpConfig);
                Assert.assertEquals(mcpDimension.getChat(), "UNKNOWN");
                Assert.assertEquals(mcpDimension.getWorkflow(), _workflowTask);
                Assert.assertEquals(mcpDimension.getBiz(), "BIZ");
                Assert.assertEquals(_mcpToolsResult, workTask);
            }
        });
        mcpTriggerManager.setTriggers(mcpToolsTrigger);
        mcpTriggerManager.beforeResourcesRead(_mcpDimension, "URI", _workflowTask);
    }

    @Test
    public void testWithOutTrigger() throws Exception {
        McpConfig _mcpConfig = new McpConfig();
        McpDimension _mcpDimension = McpDimension.builder()
                .mcpConfig(_mcpConfig)
                .build();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        McpTriggerServiceImpl mcpTriggerManager = new McpTriggerServiceImpl();
        mcpTriggerManager.beforeToolsCall(_mcpDimension, new HashMap<>(), _workflowTask);
    }

    @Test
    public void testWithOutTriggerWithPromptGet() throws Exception {
        McpConfig _mcpConfig = new McpConfig();
        McpDimension _mcpDimension = McpDimension.builder()
                .mcpConfig(_mcpConfig)
                .build();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        McpTriggerServiceImpl mcpTriggerManager = new McpTriggerServiceImpl();
        mcpTriggerManager.beforePromptGet(_mcpDimension, new HashMap<>(), _workflowTask);
    }

    @Test
    public void testWithOutTriggerWithResourcesRead() throws Exception {
        McpConfig _mcpConfig = new McpConfig();
        McpDimension _mcpDimension = McpDimension.builder()
                .mcpConfig(_mcpConfig)
                .build();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        McpTriggerServiceImpl mcpTriggerManager = new McpTriggerServiceImpl();
        mcpTriggerManager.beforeResourcesRead(_mcpDimension, "URI", _workflowTask);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testTriggerWithOutListenerAndException() throws Exception {
        McpConfig _mcpConfig = new McpConfig();
        _mcpConfig.setTrigger("HELLO1");
        McpDimension _mcpDimension = McpDimension.builder()
                .mcpConfig(_mcpConfig)
                .build();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        McpTriggerServiceImpl mcpTriggerManager = new McpTriggerServiceImpl();
        mcpTriggerManager.setTriggers(new HashMap<>());
        mcpTriggerManager.beforeToolsCall(_mcpDimension, new HashMap<>(), _workflowTask);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testTriggerWithOutListenerAndExceptionWithPromptGet() throws Exception {
        McpConfig _mcpConfig = new McpConfig();
        _mcpConfig.setTrigger("HELLO1");
        McpDimension _mcpDimension = McpDimension.builder()
                .mcpConfig(_mcpConfig)
                .build();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        McpTriggerServiceImpl mcpTriggerManager = new McpTriggerServiceImpl();
        mcpTriggerManager.setTriggers(new HashMap<>());
        mcpTriggerManager.beforePromptGet(_mcpDimension, new HashMap<>(), _workflowTask);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testTriggerWithOutListenerAndExceptionWithResourcesRead() throws Exception {
        McpConfig _mcpConfig = new McpConfig();
        _mcpConfig.setTrigger("HELLO1");
        McpDimension _mcpDimension = McpDimension.builder()
                .mcpConfig(_mcpConfig)
                .build();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        McpTriggerServiceImpl mcpTriggerManager = new McpTriggerServiceImpl();
        mcpTriggerManager.setTriggers(new HashMap<>());
        mcpTriggerManager.beforeResourcesRead(_mcpDimension, "URI", _workflowTask);
    }

    @Test
    public void testBuild() throws Exception {
        Map<String, McpTrigger> triggers = new HashMap<>();
        McpTrigger mcpTrigger = EasyMock.createMock(McpTrigger.class);
        McpTriggerServiceImpl.InitConfig empty = new McpTriggerServiceImpl.InitConfig();
        empty.setTriggers(triggers);
        empty.setGlobal(mcpTrigger);
        McpTriggerServiceImpl mcpTriggerService = (McpTriggerServiceImpl) empty.mcpTriggerService();
        Assert.assertEquals(triggers, mcpTriggerService.getTriggers());
        Assert.assertEquals(mcpTrigger, mcpTriggerService.getGlobal());
    }

    @Test
    public void testBuild2() throws Exception {
        NamesService namesService = EasyMock.createMock(NamesService.class);
        EasyMock.replay(namesService);
        Map<String, McpTrigger> triggers = new HashMap<>();
        McpTrigger mcpTrigger = new BaseTrigger();
        triggers.put(McpTrigger.NAME, mcpTrigger);
        McpTriggerServiceImpl.InitConfig empty = new McpTriggerServiceImpl.InitConfig();
        empty.setNamesService(namesService);
        empty.setTriggers(triggers);
        empty.setGlobal(mcpTrigger);
        McpTriggerServiceImpl mcpTriggerService = (McpTriggerServiceImpl) empty.mcpTriggerService();
        Assert.assertTrue(mcpTriggerService.getTriggers().isEmpty());
        Assert.assertEquals(mcpTrigger, mcpTriggerService.getGlobal());
        Assert.assertEquals(namesService, mcpTriggerService.getNamesService());
        EasyMock.verify(namesService);
    }

    @Test
    public void testGlobalTrigger() throws Exception {
        McpTriggerServiceImpl mcpTriggerManager = new McpTriggerServiceImpl();
        McpConfig mcpConfig = new McpConfig();
        Map<String, Object> arg = new HashMap<>();
        McpDimension _mcpDimension = McpDimension.builder()
                .mcpConfig(mcpConfig)
                .build();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        AtomicInteger integer = new AtomicInteger();
        mcpTriggerManager.setGlobal(new McpTrigger() {
            @Override
            public void beforeToolsCall(McpDimension mcpDimension, Map<String, Object> arguments, WorkflowTask workTask) throws Exception {
                integer.incrementAndGet();
                Assert.assertEquals(mcpDimension, _mcpDimension);
                Assert.assertEquals(arguments, arg);
                Assert.assertEquals(workTask, _workflowTask);
            }

            @Override
            public void beforePromptGet(McpDimension mcpDimension, Map<String, Object> arguments, WorkflowTask workTask) throws Exception {
                integer.incrementAndGet();
                Assert.assertEquals(mcpDimension, _mcpDimension);
                Assert.assertEquals(arguments, arg);
                Assert.assertEquals(workTask, _workflowTask);
            }

            @Override
            public void beforeResourcesRead(McpDimension mcpDimension, String uri, WorkflowTask workTask) throws Exception {
                integer.incrementAndGet();
                Assert.assertEquals(mcpDimension, _mcpDimension);
                Assert.assertEquals("URI", uri);
                Assert.assertEquals(workTask, _workflowTask);
            }
        });
        mcpTriggerManager.beforeToolsCall(_mcpDimension, arg, _workflowTask);
        mcpTriggerManager.beforePromptGet(_mcpDimension, arg, _workflowTask);
        mcpTriggerManager.beforeResourcesRead(_mcpDimension, "URI", _workflowTask);
        Assert.assertEquals(3, integer.get());
    }
    @Test
    public void testTriggerMcpConfigNull() throws Exception {
        McpTriggerServiceImpl service = new McpTriggerServiceImpl();
        McpDimension dim = McpDimension.builder().mcpConfig(null).build();
        service.beforeToolsCall(dim, null, null); // Should not throw exception
    }

    @Test
    public void testInitConfigTriggersNull() throws Exception {
        McpTriggerServiceImpl.InitConfig config = new McpTriggerServiceImpl.InitConfig();
        config.setTriggers(null);
        Assert.assertNotNull(config.mcpTriggerService());
    }
}
