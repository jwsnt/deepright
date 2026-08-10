package ai.open.right.workflow.mcp.client;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import org.junit.Assert;
import org.junit.Test;

public class McpDimensionTest {

    @Test
    public void test() {
        McpDimension mcpDimension = McpDimension.builder().device("D1").workflow("W1").chat("C1").biz("B1").build();
        Assert.assertEquals("D1", mcpDimension.getDevice());
        Assert.assertEquals("W1", mcpDimension.getWorkflow());
        Assert.assertEquals("C1", mcpDimension.getChat());
        Assert.assertEquals("B1", mcpDimension.getBiz());
    }

    @Test
    public void testMerge() {
        McpDimension mcpDimension = McpDimension.builder().device("D1").chat("C1").build();
        FunctionContext functionContext = FunctionContext.builder().workTask(ObjectBuilder.buildWorkflowTask()).build();
        mcpDimension.merge(functionContext);
        Assert.assertEquals(functionContext.getWorkTask().getWorkflow(), mcpDimension.getWorkflow());
        Assert.assertEquals("D1", mcpDimension.getDevice());
        Assert.assertEquals("C1", mcpDimension.getChat());
        Assert.assertEquals(functionContext.getWorkTask().getBiz(), mcpDimension.getBiz());
    }

    @Test
    public void testBind() {
        McpDimension mcpDimension = McpDimension.builder().build();
        mcpDimension.bind(new String[]{"A", "B"});
        Assert.assertEquals("A", mcpDimension.getClient());
        Assert.assertEquals("B", mcpDimension.getName());
    }

    @Test
    public void testToString() {
        McpDimension mcpDimension = McpDimension.builder().client("client").name("name").device("D1").chat("C1").build();
        String expect = "McpDimension(headers=null, mcpConfig=null, workflow=null, device=D1, client=client, name=name, chat=C1, biz=null)";
        Assert.assertEquals(expect, mcpDimension.toString());
    }
}
