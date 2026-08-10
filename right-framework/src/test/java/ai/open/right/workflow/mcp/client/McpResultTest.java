package ai.open.right.workflow.mcp.client;

import org.junit.Assert;
import org.junit.Test;

import java.util.Date;

public class McpResultTest {

    @Test
    public void testString() {
        McpResult result = new McpResult();
        result.setResult("OK YES");
        result.setClient("CLIENT");
        result.setName("RESULT");
        Assert.assertEquals("McpResult(client=CLIENT, name=RESULT, result=OK YES)",result.toString());
    }
}
