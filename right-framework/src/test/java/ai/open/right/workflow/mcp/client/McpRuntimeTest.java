package ai.open.right.workflow.mcp.client;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.mcp.client.McpRuntime;
import org.junit.Assert;
import org.junit.Test;

public class McpRuntimeTest {

    @Test
    public void test() {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        McpRuntime mcpRuntime = McpRuntime.builder()
                .dynamic("DYNAMIC")
                .timeout(10000)
                .workTask(workflowTask)
                .build();
        Assert.assertEquals("DYNAMIC", mcpRuntime.getDynamic());
        Assert.assertEquals(Integer.valueOf(10000), mcpRuntime.getTimeout());
        Assert.assertEquals(workflowTask, mcpRuntime.getWorkTask());
        Assert.assertEquals(Integer.valueOf(10000), mcpRuntime.getTimeout(1200));
    }

    @Test
    public void testTimeout() {
        McpRuntime mcpRuntime = McpRuntime.builder()
                .build();
        Assert.assertEquals(Integer.valueOf(1200), mcpRuntime.getTimeout(1200));
    }
}
