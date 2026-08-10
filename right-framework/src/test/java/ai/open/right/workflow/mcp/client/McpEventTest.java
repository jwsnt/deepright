package ai.open.right.workflow.mcp.client;

import ai.open.right.workflow.flow.config.McpConfig;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import org.junit.Assert;
import org.junit.Test;

import java.util.Map;

public class McpEventTest {

    @Test
    public void test1() {
        McpDimension mcpDimension = McpDimension.builder()
                .device("DEVICE")
                .workflow("WORKFLOW")
                .mcpConfig(new McpConfig())
                .biz("BIZ")
                .chat("CHAT")
                .workflow("WORKFLOW")
                .build();
        McpEvent mcpEvent = new McpEvent(mcpDimension, "CLIENT", "NAME", "URI");
        mcpEvent.init();
        Assert.assertEquals("WORKFLOW",mcpEvent.getWorkflow());
        Assert.assertEquals("BIZ", mcpEvent.getBiz());
        Assert.assertEquals("CHAT", mcpEvent.getChat());
        Assert.assertEquals("DEVICE", mcpEvent.getDevice());
        Assert.assertEquals("BIZ-CHAT-DEVICE", mcpEvent.getDimension());
        Assert.assertNotNull(mcpEvent.getNow());
        Assert.assertEquals(McpEvent.TYPE, mcpEvent.getType());
        Map<String, Object> body = Map.class.cast(mcpEvent.getData());
        Assert.assertEquals(Integer.valueOf(4), Integer.valueOf(body.size()));
        Assert.assertEquals(mcpDimension, body.get("mcpDimension"));
        Assert.assertEquals("CLIENT", body.get("result"));
        Assert.assertEquals("NAME", body.get("client"));
        Assert.assertEquals("URI", body.get("param"));
    }

    @Test
    public void test2() {
        McpDimension mcpDimension = McpDimension.builder()
                .device("DEVICE")
                .mcpConfig(new McpConfig())
                .biz("BIZ")
                .chat("CHAT")
                .workflow("WORKFLOW")
                .build();
        McpEvent mcpEvent = new McpEvent(mcpDimension, "CLIENT", "NAME");
        Assert.assertEquals("BIZ", mcpEvent.getBiz());
        Assert.assertEquals("CHAT", mcpEvent.getChat());
        Assert.assertEquals("DEVICE", mcpEvent.getDevice());
        Assert.assertEquals("BIZ-CHAT-DEVICE", mcpEvent.getDimension());
        Assert.assertNotNull(mcpEvent.getNow());
        Assert.assertEquals(McpEvent.TYPE, mcpEvent.getType());
        Map<String, Object> body = Map.class.cast(mcpEvent.getData());
        Assert.assertEquals(Integer.valueOf(4), Integer.valueOf(body.size()));
        Assert.assertEquals(mcpDimension, body.get("mcpDimension"));
        Assert.assertEquals("CLIENT", body.get("result"));
        Assert.assertEquals("NAME", body.get("client"));
        Assert.assertNull(body.get("param"));
    }
}
