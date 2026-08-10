package ai.open.right.workflow.mcp.client.trigger;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.mcp.client.McpContext;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class BaseTriggerTest {

    @Test
    public void toolsCall() throws Exception {
        BaseTrigger baseTrigger = new BaseTrigger();
        McpDimension mcpDimension = McpDimension.builder().build();
        baseTrigger.beforeToolsCall(mcpDimension, new HashMap<>(), ObjectBuilder.buildWorkflowTask());
    }

    @Test
    public void testPromptGet() throws Exception {
        BaseTrigger baseTrigger = new BaseTrigger();
        McpDimension mcpDimension = McpDimension.builder().build();
        baseTrigger.beforePromptGet(mcpDimension, new HashMap<>(), ObjectBuilder.buildWorkflowTask());
    }

    @Test
    public void testResourcesRead() throws Exception {
        BaseTrigger baseTrigger = new BaseTrigger();
        McpDimension mcpDimension = McpDimension.builder().build();
        baseTrigger.beforeResourcesRead(mcpDimension, "URI", ObjectBuilder.buildWorkflowTask());
    }

    @Test
    public void testEndpoint1() throws Exception {
        McpContext<?> context = new McpContext<>();
        context.setWorkTask(ObjectBuilder.buildWorkflowTask());
        Map<String, Object> _metadata = new HashMap<>();
        String _content = "QUERY";
        Integer _code = 250;
        BaseTrigger baseTrigger = new BaseTrigger() {
            public void endpoint(WorkflowTask workTask, Map<String, Object> metadata, String content, Integer code) throws Exception {
                Assert.assertEquals(workTask, context.getWorkTask());
                Assert.assertEquals(_metadata, metadata);
                Assert.assertEquals(_content, content);
                Assert.assertEquals(_code, code);
            }
        };
        baseTrigger.endpoint(context, _metadata, _content, _code);
    }

    @Test
    public void testEndpoint2() throws Exception {
        McpContext<?> context = new McpContext<>();
        context.setWorkTask(ObjectBuilder.buildWorkflowTask());
        Map<String, Object> _metadata = new HashMap<>();
        String _content = "QUERY";
        BaseTrigger baseTrigger = new BaseTrigger() {
            public void endpoint(WorkflowTask workTask, Map<String, Object> metadata, String content, Integer code) throws Exception {
                Assert.assertEquals(workTask, context.getWorkTask());
                Assert.assertEquals(_metadata, metadata);
                Assert.assertEquals(_content, content);
            }
        };
        baseTrigger.endpoint(context, _metadata, _content);
    }

    @Test
    public void testEndpoint3() throws Exception {
        McpContext<?> context = new McpContext<>();
        context.setWorkTask(ObjectBuilder.buildWorkflowTask());
        Map<String, Object> _metadata = new HashMap<>();
        String _content = "QUERY";
        Integer _code = 250;
        BaseTrigger baseTrigger = new BaseTrigger() {
            public void endpoint(WorkflowTask workTask, Map<String, Object> metadata, String content, Integer code) throws Exception {
                Assert.assertEquals(workTask, context.getWorkTask());
                Assert.assertEquals(_metadata, metadata);
                Assert.assertEquals(_content, content);
                Assert.assertEquals(_code, code);
            }
        };
        baseTrigger.endpoint(context, _content, 250);
    }

    @Test
    public void testEndpoint4() throws Exception {
        McpContext<?> context = new McpContext<>();
        context.setWorkTask(ObjectBuilder.buildWorkflowTask());
        Map<String, Object> _metadata = new HashMap<>();
        String _content = "QUERY";
        BaseTrigger baseTrigger = new BaseTrigger() {
            public void endpoint(WorkflowTask workTask, Map<String, Object> metadata, String content, Integer code) throws Exception {
                Assert.assertEquals(workTask, context.getWorkTask());
                Assert.assertEquals(_metadata, metadata);
                Assert.assertEquals(_content, content);
            }
        };
        baseTrigger.endpoint(context, _content);
    }

    @Test
    public void testSource1() throws Exception {
        McpContext<?> context = new McpContext<>();
        context.setWorkTask(ObjectBuilder.buildWorkflowTask());
        Map<String, Object> _metadata = new HashMap<>();
        String _content = "QUERY";
        Integer _code = 250;
        BaseTrigger baseTrigger = new BaseTrigger() {
            public void source(WorkflowTask workTask, Map<String, Object> metadata, String content, Integer code) throws Exception {
                Assert.assertEquals(workTask, context.getWorkTask());
                Assert.assertEquals(_metadata, metadata);
                Assert.assertEquals(_content, content);
                Assert.assertEquals(_code, code);
            }
        };
        baseTrigger.source(context, _metadata, _content, _code);
    }

    @Test
    public void testSource2() throws Exception {
        McpContext<?> context = new McpContext<>();
        context.setWorkTask(ObjectBuilder.buildWorkflowTask());
        Map<String, Object> _metadata = new HashMap<>();
        String _content = "QUERY";
        BaseTrigger baseTrigger = new BaseTrigger() {
            public void source(WorkflowTask workTask, Map<String, Object> metadata, String content, Integer code) throws Exception {
                Assert.assertEquals(workTask, context.getWorkTask());
                Assert.assertEquals(_metadata, metadata);
                Assert.assertEquals(_content, content);
            }
        };
        baseTrigger.source(context, _metadata, _content);
    }

    @Test
    public void testSource3() throws Exception {
        McpContext<?> context = new McpContext<>();
        context.setWorkTask(ObjectBuilder.buildWorkflowTask());
        Map<String, Object> _metadata = new HashMap<>();
        String _content = "QUERY";
        Integer _code = 250;
        BaseTrigger baseTrigger = new BaseTrigger() {
            public void source(WorkflowTask workTask, Map<String, Object> metadata, String content, Integer code) throws Exception {
                Assert.assertEquals(workTask, context.getWorkTask());
                Assert.assertEquals(_metadata, metadata);
                Assert.assertEquals(_content, content);
                Assert.assertEquals(_code, code);
            }
        };
        baseTrigger.source(context, _content, 250);
    }

    @Test
    public void testSource4() throws Exception {
        McpContext<?> context = new McpContext<>();
        context.setWorkTask(ObjectBuilder.buildWorkflowTask());
        Map<String, Object> _metadata = new HashMap<>();
        String _content = "QUERY";
        BaseTrigger baseTrigger = new BaseTrigger() {
            public void source(WorkflowTask workTask, Map<String, Object> metadata, String content, Integer code) throws Exception {
                Assert.assertEquals(workTask, context.getWorkTask());
                Assert.assertEquals(_metadata, metadata);
                Assert.assertEquals(_content, content);
            }
        };
        baseTrigger.source(context, _content);
    }

    @Test
    public void testLocalhost() throws Exception {
        McpContext<?> context = new McpContext<>();
        context.setWorkTask(ObjectBuilder.buildWorkflowTask());
        SyncConfig _syncConfig = SyncConfig.builder().build();
        BaseTrigger baseTrigger = new BaseTrigger() {
            public SyncWorkflowTask localhost(WorkflowTask workTask, SyncConfig syncConfig) throws Exception {
                Assert.assertEquals(_syncConfig, syncConfig);
                Assert.assertEquals(context.getWorkTask(), workTask);
                return null;
            }
        };
        baseTrigger.localhost(context, _syncConfig);
    }

    @Test
    public void testSetGet() {
        McpContext<?> context = new McpContext<>();
        context.setWorkTask(ObjectBuilder.buildWorkflowTask());
        SyncConfig _syncConfig = SyncConfig.builder().build();
        BaseTrigger baseTrigger = new BaseTrigger() {
            public SyncWorkflowTask localhost(WorkflowTask workTask, SyncConfig syncConfig) throws Exception {
                Assert.assertEquals(_syncConfig, syncConfig);
                Assert.assertEquals(context.getWorkTask(), workTask);
                return null;
            }
        };
        Assert.assertNull(baseTrigger.getNotifierService());
        baseTrigger.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        Assert.assertNotNull(baseTrigger.getNotifierService());
    }
}
