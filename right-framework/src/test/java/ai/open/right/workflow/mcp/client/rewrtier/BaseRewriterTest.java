package ai.open.right.workflow.mcp.client.rewrtier;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.mcp.client.McpContext;
import ai.open.right.workflow.mcp.client.McpResult;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class BaseRewriterTest {

    @Test
    public void toolsCall() throws Exception {
        McpContext<List<Map<String, Object>>> context = new McpContext<List<Map<String, Object>>>();
        McpResult<List<Map<String, Object>>> result = new McpResult<List<Map<String, Object>>>();
        List<Map<String, Object>> content = new ArrayList<>();
        result.setResult(content);
        context.setResult(result);
        BaseRewriter baseListener = new BaseRewriter();
        Assert.assertEquals(content, baseListener.toolsCall(context).getResult());
        Assert.assertNull(baseListener.getNotifierService());
        baseListener.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        Assert.assertNotNull(baseListener.getNotifierService());
    }

    @Test
    public void testPromptGet() throws Exception {
        McpContext<List<History>> context = new McpContext<List<History>>();
        McpResult<List<History>> result = new McpResult<List<History>>();
        List<History> content = new ArrayList<>();
        result.setResult(content);
        context.setResult(result);
        BaseRewriter baseListener = new BaseRewriter();
        Assert.assertEquals(content, baseListener.promptGet(context).getResult());
    }

    @Test
    public void testResourcesRead() throws Exception {
        McpContext<String> context = new McpContext<String>();
        McpResult<String> result = new McpResult<String>();
        result.setResult("HELLO");
        context.setResult(result);
        BaseRewriter baseListener = new BaseRewriter();
        Assert.assertEquals("HELLO", baseListener.resourcesRead(context).getResult());
    }

    @Test
    public void testEndpoint1() throws Exception {
        McpContext<?> context = new McpContext<>();
        context.setWorkTask(ObjectBuilder.buildWorkflowTask());
        Map<String, Object> _metadata = new HashMap<>();
        String _content = "QUERY";
        Integer _code = 250;
        BaseRewriter baseListener = new BaseRewriter() {
            public void endpoint(WorkflowTask workTask, Map<String, Object> metadata, String content, Integer code) throws Exception {
                Assert.assertEquals(workTask, context.getWorkTask());
                Assert.assertEquals(_metadata, metadata);
                Assert.assertEquals(_content, content);
                Assert.assertEquals(_code, code);
            }
        };
        baseListener.endpoint(context, _metadata, _content, _code);
    }

    @Test
    public void testEndpoint2() throws Exception {
        McpContext<?> context = new McpContext<>();
        context.setWorkTask(ObjectBuilder.buildWorkflowTask());
        Map<String, Object> _metadata = new HashMap<>();
        String _content = "QUERY";
        BaseRewriter baseListener = new BaseRewriter() {
            public void endpoint(WorkflowTask workTask, Map<String, Object> metadata, String content, Integer code) throws Exception {
                Assert.assertEquals(workTask, context.getWorkTask());
                Assert.assertEquals(_metadata, metadata);
                Assert.assertEquals(_content, content);
            }
        };
        baseListener.endpoint(context, _metadata, _content);
    }

    @Test
    public void testEndpoint3() throws Exception {
        McpContext<?> context = new McpContext<>();
        context.setWorkTask(ObjectBuilder.buildWorkflowTask());
        Map<String, Object> _metadata = new HashMap<>();
        String _content = "QUERY";
        Integer _code = 250;
        BaseRewriter baseListener = new BaseRewriter() {
            public void endpoint(WorkflowTask workTask, Map<String, Object> metadata, String content, Integer code) throws Exception {
                Assert.assertEquals(workTask, context.getWorkTask());
                Assert.assertEquals(_metadata, metadata);
                Assert.assertEquals(_content, content);
                Assert.assertEquals(_code, code);
            }
        };
        baseListener.endpoint(context, _content, 250);
    }

    @Test
    public void testEndpoint4() throws Exception {
        McpContext<?> context = new McpContext<>();
        context.setWorkTask(ObjectBuilder.buildWorkflowTask());
        Map<String, Object> _metadata = new HashMap<>();
        String _content = "QUERY";
        BaseRewriter baseListener = new BaseRewriter() {
            public void endpoint(WorkflowTask workTask, Map<String, Object> metadata, String content, Integer code) throws Exception {
                Assert.assertEquals(workTask, context.getWorkTask());
                Assert.assertEquals(_metadata, metadata);
                Assert.assertEquals(_content, content);
            }
        };
        baseListener.endpoint(context, _content);
    }

    @Test
    public void testSource1() throws Exception {
        McpContext<?> context = new McpContext<>();
        context.setWorkTask(ObjectBuilder.buildWorkflowTask());
        Map<String, Object> _metadata = new HashMap<>();
        String _content = "QUERY";
        Integer _code = 250;
        BaseRewriter baseListener = new BaseRewriter() {
            public void source(WorkflowTask workTask, Map<String, Object> metadata, String content, Integer code) throws Exception {
                Assert.assertEquals(workTask, context.getWorkTask());
                Assert.assertEquals(_metadata, metadata);
                Assert.assertEquals(_content, content);
                Assert.assertEquals(_code, code);
            }
        };
        baseListener.source(context, _metadata, _content, _code);
    }

    @Test
    public void testSource2() throws Exception {
        McpContext<?> context = new McpContext<>();
        context.setWorkTask(ObjectBuilder.buildWorkflowTask());
        Map<String, Object> _metadata = new HashMap<>();
        String _content = "QUERY";
        BaseRewriter baseListener = new BaseRewriter() {
            public void source(WorkflowTask workTask, Map<String, Object> metadata, String content, Integer code) throws Exception {
                Assert.assertEquals(workTask, context.getWorkTask());
                Assert.assertEquals(_metadata, metadata);
                Assert.assertEquals(_content, content);
            }
        };
        baseListener.source(context, _metadata, _content);
    }

    @Test
    public void testSource3() throws Exception {
        McpContext<?> context = new McpContext<>();
        context.setWorkTask(ObjectBuilder.buildWorkflowTask());
        Map<String, Object> _metadata = new HashMap<>();
        String _content = "QUERY";
        Integer _code = 250;
        BaseRewriter baseListener = new BaseRewriter() {
            public void source(WorkflowTask workTask, Map<String, Object> metadata, String content, Integer code) throws Exception {
                Assert.assertEquals(workTask, context.getWorkTask());
                Assert.assertEquals(_metadata, metadata);
                Assert.assertEquals(_content, content);
                Assert.assertEquals(_code, code);
            }
        };
        baseListener.source(context, _content, 250);
    }

    @Test
    public void testSource4() throws Exception {
        McpContext<?> context = new McpContext<>();
        context.setWorkTask(ObjectBuilder.buildWorkflowTask());
        Map<String, Object> _metadata = new HashMap<>();
        String _content = "QUERY";
        BaseRewriter baseListener = new BaseRewriter() {
            public void source(WorkflowTask workTask, Map<String, Object> metadata, String content, Integer code) throws Exception {
                Assert.assertEquals(workTask, context.getWorkTask());
                Assert.assertEquals(_metadata, metadata);
                Assert.assertEquals(_content, content);
            }
        };
        baseListener.source(context, _content);
    }

    @Test
    public void testLocalhost() throws Exception {
        McpContext<?> context = new McpContext<>();
        context.setWorkTask(ObjectBuilder.buildWorkflowTask());
        SyncConfig _syncConfig = SyncConfig.builder().build();
        BaseRewriter baseListener = new BaseRewriter() {
            public SyncWorkflowTask localhost(WorkflowTask workTask, SyncConfig syncConfig) throws Exception {
                Assert.assertEquals(_syncConfig, syncConfig);
                Assert.assertEquals(context.getWorkTask(), workTask);
                return null;
            }
        };
        baseListener.localhost(context, _syncConfig);
    }
}
