package ai.open.right.workflow.mcp.client;

import ai.open.right.workflow.mcp.client.McpResult;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

public class McpToolsResultTest {

    @Test
    public void test() {
        List<Map<String, Object>> result = new ArrayList<>();
        McpResult mcpToolsResult = new McpResult();
        mcpToolsResult.setResult(result);
        mcpToolsResult.setClient("CLIENT");
        mcpToolsResult.setName("NAME");
        Assert.assertEquals(result, mcpToolsResult.getResult());
        Assert.assertEquals("CLIENT", mcpToolsResult.getClient());
        Assert.assertEquals("NAME", mcpToolsResult.getName());
    }
}
