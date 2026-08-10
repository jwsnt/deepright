package ai.open.right.workflow.mcp.server.cmd;

import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class McpCmdResponseTest {

    @Test
    public void test() {
        Map<String, Object> map = new HashMap<>();
        McpCmdResponse mcpCmdResponse = McpCmdResponse.builder()
                .result(map)
                .notifier(true)
                .wrap(false)
                .build();
        Assert.assertEquals(map, mcpCmdResponse.getResult());
        Assert.assertTrue(mcpCmdResponse.getNotifier());
        Assert.assertFalse(mcpCmdResponse.getWrap());
        Map<String, Object> map2 = new HashMap<>();
        mcpCmdResponse.setResult(map2);
        mcpCmdResponse.setNotifier(false);
        mcpCmdResponse.setWrap(true);
        Assert.assertFalse(mcpCmdResponse.getNotifier());
        Assert.assertTrue(mcpCmdResponse.getWrap());
    }
}
