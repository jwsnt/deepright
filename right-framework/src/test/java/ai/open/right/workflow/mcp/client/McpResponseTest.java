package ai.open.right.workflow.mcp.client;

import ai.open.right.WorkflowException;
import ai.open.right.workflow.mcp.client.McpResponse;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class McpResponseTest {

    @Test(expected = WorkflowException.class)
    public void test() throws Exception {
        McpResponse mcpResponse = new McpResponse();
        mcpResponse.setId("HELLO");
        Map<String, Object> error = new HashMap<String, Object>();
        error.put("message", "ERROR");
        error.put("code", 10086);
        error.put("isError", true);
        mcpResponse.setResult(error);
        mcpResponse.check(true);
        Assert.assertEquals("HELLO", mcpResponse.getId());
        mcpResponse.setId(null);
        Assert.assertNull(mcpResponse.getId());
    }

    @Test
    public void testCheckErrorAndSuccess() throws Exception {
        McpResponse mcpResponse = new McpResponse();
        mcpResponse.setId("HELLO");
        Map<String, Object> error = new HashMap<String, Object>();
        error.put("message", "ERROR");
        error.put("code", 10086);
        error.put("isError", true);
        mcpResponse.setResult(error);
        mcpResponse.check();
        Assert.assertEquals(error, mcpResponse.getResult());
    }
}
