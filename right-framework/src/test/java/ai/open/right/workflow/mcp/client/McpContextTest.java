package ai.open.right.workflow.mcp.client;

import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class McpContextTest {

    @Test
    public void test() {
        McpContext mcpContext = new McpContext();
        Map<String, Object> arg = new HashMap<>();
        mcpContext.setUri("URI");
        mcpContext.setArguments(arg);
        Assert.assertEquals("URI", mcpContext.getUri());
        Assert.assertEquals(arg, mcpContext.getArguments());
    }
}
