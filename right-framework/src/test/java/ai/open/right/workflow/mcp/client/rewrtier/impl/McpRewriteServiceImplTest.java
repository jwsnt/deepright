package ai.open.right.workflow.mcp.client.rewrtier.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.McpConfig;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.mcp.client.McpContext;
import ai.open.right.workflow.mcp.client.McpResult;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.rewrtier.BaseRewriter;
import ai.open.right.workflow.mcp.client.rewrtier.McpRewriter;
import org.junit.Assert;
import org.junit.Test;

import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.atomic.AtomicInteger;

public class McpRewriteServiceImplTest {

    @Test
    public void testListener() throws Exception {
        McpRewriteServiceImpl mcpListenerManager = new McpRewriteServiceImpl();
        mcpListenerManager.setRewriter(Collections.singletonMap("HELLO", new BaseRewriter()));
        McpConfig mcpConfig = new McpConfig();
        mcpConfig.setRewriter("HELLO");
        McpDimension _mcpDimension = McpDimension.builder().mcpConfig(mcpConfig).build();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        McpResult mcpToolsResult = new McpResult();
        mcpToolsResult.setClient("CLIENT");
        mcpToolsResult.setName("NAME");
        McpResult result = mcpListenerManager.toolsCall(_mcpDimension, new HashMap<>(), workflowTask, mcpToolsResult);
        Assert.assertEquals(mcpToolsResult, result);
    }

    @Test
    public void testListenerWithPromptGet() throws Exception {
        McpRewriteServiceImpl mcpListenerManager = new McpRewriteServiceImpl();
        mcpListenerManager.setRewriter(Collections.singletonMap("HELLO", new BaseRewriter()));
        McpConfig mcpConfig = new McpConfig();
        mcpConfig.setRewriter("HELLO");
        McpDimension _mcpDimension = McpDimension.builder().mcpConfig(mcpConfig).build();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        McpResult mcpToolsResult = new McpResult();
        mcpToolsResult.setClient("CLIENT");
        mcpToolsResult.setName("NAME");
        McpResult result = mcpListenerManager.promptGet(_mcpDimension, new HashMap<>(), workflowTask, mcpToolsResult);
        Assert.assertEquals(mcpToolsResult, result);
    }

    @Test
    public void testListenerWithResourcesRead() throws Exception {
        McpRewriteServiceImpl mcpListenerManager = new McpRewriteServiceImpl();
        mcpListenerManager.setRewriter(Collections.singletonMap("HELLO", new BaseRewriter()));
        McpConfig mcpConfig = new McpConfig();
        mcpConfig.setRewriter("HELLO");
        McpDimension _mcpDimension = McpDimension.builder().mcpConfig(mcpConfig).build();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        McpResult mcpToolsResult = new McpResult();
        mcpToolsResult.setClient("CLIENT");
        mcpToolsResult.setName("NAME");
        McpResult result = mcpListenerManager.resourcesRead(_mcpDimension, "URI", workflowTask, mcpToolsResult);
        Assert.assertEquals(mcpToolsResult, result);
    }

    @Test
    public void testListener2() throws Exception {
        McpConfig _mcpConfig = new McpConfig();
        _mcpConfig.setRewriter("HELLO");
        McpDimension _mcpDimension = McpDimension.builder().mcpConfig(_mcpConfig).build();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        McpResult _mcpToolsResult = new McpResult();
        _mcpToolsResult.setClient("CLIENT");
        _mcpToolsResult.setName("NAME");
        McpRewriteServiceImpl mcpListenerManager = new McpRewriteServiceImpl();
        Map<String, McpRewriter> mcpToolsListener = new HashMap<>();
        mcpToolsListener.put("HELLO", new BaseRewriter() {

            @Override
            public McpResult toolsCall(McpContext<List<Map<String, Object>>> context) throws Exception {
                Assert.assertEquals(context.getMcpConfig(), _mcpConfig);
                Assert.assertEquals(context.getDimension(), _mcpDimension);
                Assert.assertEquals(context.getWorkTask(), _workflowTask);
                Assert.assertEquals(_mcpToolsResult, context.getResult());
                return context.getResult();
            }
        });
        mcpListenerManager.setRewriter(mcpToolsListener);
        McpResult result = mcpListenerManager.toolsCall(_mcpDimension, new HashMap<>(), _workflowTask, _mcpToolsResult);
        Assert.assertEquals(_mcpToolsResult, result);
    }

    @Test
    public void testListener2WithPromptGet() throws Exception {
        McpConfig _mcpConfig = new McpConfig();
        _mcpConfig.setRewriter("HELLO");
        McpDimension _mcpDimension = McpDimension.builder().mcpConfig(_mcpConfig).build();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        McpResult _mcpToolsResult = new McpResult();
        _mcpToolsResult.setClient("CLIENT");
        _mcpToolsResult.setName("NAME");
        McpRewriteServiceImpl mcpListenerManager = new McpRewriteServiceImpl();
        Map<String, McpRewriter> mcpToolsListener = new HashMap<>();
        mcpToolsListener.put("HELLO", new BaseRewriter() {

            @Override
            public McpResult promptGet(McpContext<List<History>> context) throws Exception {
                Assert.assertEquals(context.getMcpConfig(), _mcpConfig);
                Assert.assertEquals(context.getDimension(), _mcpDimension);
                Assert.assertEquals(context.getWorkTask(), _workflowTask);
                Assert.assertEquals(_mcpToolsResult, context.getResult());
                return context.getResult();
            }
        });
        mcpListenerManager.setRewriter(mcpToolsListener);
        McpResult result = mcpListenerManager.promptGet(_mcpDimension, new HashMap<>(), _workflowTask, _mcpToolsResult);
        Assert.assertEquals(_mcpToolsResult, result);
    }

    @Test
    public void testListener3WithResourcesRead() throws Exception {
        McpConfig _mcpConfig = new McpConfig();
        _mcpConfig.setRewriter("HELLO");
        McpDimension _mcpDimension = McpDimension.builder().mcpConfig(_mcpConfig).build();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        McpResult _mcpToolsResult = new McpResult();
        _mcpToolsResult.setClient("CLIENT");
        _mcpToolsResult.setName("NAME");
        McpRewriteServiceImpl mcpListenerManager = new McpRewriteServiceImpl();
        Map<String, McpRewriter> mcpToolsListener = new HashMap<>();
        mcpToolsListener.put("HELLO", new BaseRewriter() {

            @Override
            public McpResult resourcesRead(McpContext<String> context) throws Exception {
                Assert.assertEquals(context.getMcpConfig(), _mcpConfig);
                Assert.assertEquals(context.getDimension(), _mcpDimension);
                Assert.assertEquals(context.getWorkTask(), _workflowTask);
                Assert.assertEquals(_mcpToolsResult, context.getResult());
                return context.getResult();
            }
        });
        mcpListenerManager.setRewriter(mcpToolsListener);
        McpResult result = mcpListenerManager.resourcesRead(_mcpDimension, "URI", _workflowTask, _mcpToolsResult);
        Assert.assertEquals(_mcpToolsResult, result);
    }

    @Test
    public void testWithOutListener() throws Exception {
        McpConfig _mcpConfig = new McpConfig();
        McpDimension _mcpDimension = McpDimension.builder().mcpConfig(_mcpConfig).build();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        McpResult _mcpToolsResult = new McpResult();
        _mcpToolsResult.setClient("CLIENT");
        _mcpToolsResult.setName("NAME");
        McpRewriteServiceImpl mcpListenerManager = new McpRewriteServiceImpl();
        McpResult result = mcpListenerManager.toolsCall(_mcpDimension, new HashMap<>(), _workflowTask, _mcpToolsResult);
        Assert.assertEquals(_mcpToolsResult, result);
    }

    @Test
    public void testWithOutListenerWithPromptGet() throws Exception {
        McpConfig _mcpConfig = new McpConfig();
        McpDimension _mcpDimension = McpDimension.builder().mcpConfig(_mcpConfig).build();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        McpResult _mcpToolsResult = new McpResult();
        _mcpToolsResult.setClient("CLIENT");
        _mcpToolsResult.setName("NAME");
        McpRewriteServiceImpl mcpListenerManager = new McpRewriteServiceImpl();
        McpResult result = mcpListenerManager.promptGet(_mcpDimension, new HashMap<>(), _workflowTask, _mcpToolsResult);
        Assert.assertEquals(_mcpToolsResult, result);
    }

    @Test
    public void testWithOutListenerWithResourcesRead() throws Exception {
        McpConfig _mcpConfig = new McpConfig();
        McpDimension _mcpDimension = McpDimension.builder().mcpConfig(_mcpConfig).build();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        McpResult _mcpToolsResult = new McpResult();
        _mcpToolsResult.setClient("CLIENT");
        _mcpToolsResult.setName("NAME");
        McpRewriteServiceImpl mcpListenerManager = new McpRewriteServiceImpl();
        McpResult result = mcpListenerManager.resourcesRead(_mcpDimension, "URI", _workflowTask, _mcpToolsResult);
        Assert.assertEquals(_mcpToolsResult, result);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testListenerWithOutListenerAndException() throws Exception {
        McpConfig _mcpConfig = new McpConfig();
        _mcpConfig.setRewriter("HELLO1");
        McpDimension _mcpDimension = McpDimension.builder().mcpConfig(_mcpConfig).build();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        McpResult _mcpToolsResult = new McpResult();
        _mcpToolsResult.setClient("CLIENT");
        _mcpToolsResult.setName("NAME");
        McpRewriteServiceImpl mcpListenerManager = new McpRewriteServiceImpl();
        mcpListenerManager.setRewriter(new HashMap<>());
        mcpListenerManager.toolsCall(_mcpDimension, new HashMap<>(), _workflowTask, _mcpToolsResult);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testListenerWithOutListenerAndExceptionWithPromptGet() throws Exception {
        McpConfig _mcpConfig = new McpConfig();
        _mcpConfig.setRewriter("HELLO1");
        McpDimension _mcpDimension = McpDimension.builder().mcpConfig(_mcpConfig).build();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        McpResult _mcpToolsResult = new McpResult();
        _mcpToolsResult.setClient("CLIENT");
        _mcpToolsResult.setName("NAME");
        McpRewriteServiceImpl mcpListenerManager = new McpRewriteServiceImpl();
        mcpListenerManager.setRewriter(new HashMap<>());
        mcpListenerManager.promptGet(_mcpDimension, new HashMap<>(), _workflowTask, _mcpToolsResult);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testListenerWithOutListenerAndExceptionWithResourcesRead() throws Exception {
        McpConfig _mcpConfig = new McpConfig();
        _mcpConfig.setRewriter("HELLO1");
        McpDimension _mcpDimension = McpDimension.builder().mcpConfig(_mcpConfig).build();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        McpResult _mcpToolsResult = new McpResult();
        _mcpToolsResult.setClient("CLIENT");
        _mcpToolsResult.setName("NAME");
        McpRewriteServiceImpl mcpListenerManager = new McpRewriteServiceImpl();
        mcpListenerManager.setRewriter(new HashMap<>());
        mcpListenerManager.resourcesRead(_mcpDimension, "URI", _workflowTask, _mcpToolsResult);
    }

    @Test
    public void testBuild() {
        Map<String, McpRewriter> listener = new HashMap<>();
        McpRewriteServiceImpl.InitConfig empty = new McpRewriteServiceImpl.InitConfig();
        empty.setRewriter(listener);
        Assert.assertEquals(listener, empty.getRewriter());
    }

    @Test
    public void testBuild2() throws Exception {
        Map<String, McpRewriter> listener = new HashMap<>();
        McpRewriter mcpRewriter = new BaseRewriter();
        listener.put(McpRewriter.NAME, mcpRewriter);
        McpRewriteServiceImpl.InitConfig empty = new McpRewriteServiceImpl.InitConfig();
        empty.setRewriter(listener);
        empty.setGlobal(mcpRewriter);
        McpRewriteServiceImpl mcpRewriteService = (McpRewriteServiceImpl) empty.mcpRewriteService();
        Assert.assertTrue(mcpRewriteService.getRewriter().isEmpty());
    }

    @Test
    public void testGetSet() throws Exception {
        Map<String, McpRewriter> listener = new HashMap<>();
        McpRewriteServiceImpl.InitConfig empty = new McpRewriteServiceImpl.InitConfig();
        empty.setRewriter(listener);
        McpRewriteServiceImpl service = (McpRewriteServiceImpl) empty.mcpRewriteService();
        Assert.assertEquals(listener, service.getRewriter());
    }

    @Test
    public void testGlobal() throws Exception {
        McpRewriteServiceImpl mcpRewriteService = new McpRewriteServiceImpl();
        McpConfig mcpConfig = new McpConfig();
        McpDimension _mcpDimension = McpDimension.builder().mcpConfig(mcpConfig).build();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        McpResult mcpToolsResult = new McpResult();
        Map<String, Object> arg = new HashMap<>();
        mcpToolsResult.setClient("CLIENT");
        mcpToolsResult.setName("NAME");
        AtomicInteger integer = new AtomicInteger();
        McpResult result = new McpResult<>();
        mcpRewriteService.setGlobal(new McpRewriter() {
            @Override
            public McpResult<List<Map<String, Object>>> toolsCall(McpContext<List<Map<String, Object>>> context) throws Exception {
                integer.incrementAndGet();
                Assert.assertEquals(context.getWorkTask(), workflowTask);
                Assert.assertEquals(context.getArguments(), arg);
                Assert.assertEquals(context.getDimension(), _mcpDimension);
                Assert.assertEquals(context.getMcpConfig(), mcpConfig);
                Assert.assertEquals(context.getResult(), mcpToolsResult);
                return result;
            }

            @Override
            public McpResult<List<History>> promptGet(McpContext<List<History>> context) throws Exception {
                integer.incrementAndGet();
                Assert.assertEquals(context.getWorkTask(), workflowTask);
                Assert.assertEquals(context.getArguments(), arg);
                Assert.assertEquals(context.getDimension(), _mcpDimension);
                Assert.assertEquals(context.getMcpConfig(), mcpConfig);
                Assert.assertEquals(context.getResult(), mcpToolsResult);
                return result;
            }

            @Override
            public McpResult<String> resourcesRead(McpContext<String> context) throws Exception {
                integer.incrementAndGet();
                Assert.assertEquals(context.getWorkTask(), workflowTask);
                Assert.assertEquals(context.getUri(), "URI");
                Assert.assertEquals(context.getDimension(), _mcpDimension);
                Assert.assertEquals(context.getMcpConfig(), mcpConfig);
                Assert.assertEquals(context.getResult(), mcpToolsResult);
                return result;
            }
        });
        Assert.assertEquals(result, mcpRewriteService.toolsCall(_mcpDimension, arg, workflowTask, mcpToolsResult));
        Assert.assertEquals(result, mcpRewriteService.promptGet(_mcpDimension, arg, workflowTask, mcpToolsResult));
        Assert.assertEquals(result, mcpRewriteService.resourcesRead(_mcpDimension, "URI", workflowTask, mcpToolsResult));
    }

    @Test
    public void testInitConfigRewritersNull() throws Exception {
        McpRewriteServiceImpl.InitConfig config = new McpRewriteServiceImpl.InitConfig();
        config.setRewriter(null);
        Assert.assertNotNull(config.mcpRewriteService());
    }
}
