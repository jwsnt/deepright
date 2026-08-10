package ai.open.right.workflow.mcp.server.cmd;

import org.junit.Assert;
import org.junit.Test;

public class McpCmdPromptTest {

    @Test
    public void test() {
        McpCmdPrompt prompt = new McpCmdPrompt();
        prompt.setRole("user");
        prompt.setContent("content");
        Assert.assertEquals("user", prompt.getRole());
        Assert.assertEquals("content", prompt.getContent());
    }

    @Test
    public void testBuild() {
        McpCmdPrompt prompt = McpCmdPrompt.builder()
                .content("content")
                .role("user")
                .build();
        Assert.assertEquals("content", prompt.getContent());
        Assert.assertEquals("user", prompt.getRole());
    }

    @Test
    public void testConst() {
        McpCmdPrompt prompt = new McpCmdPrompt("content", "user");
        Assert.assertEquals("content", prompt.getContent());
        Assert.assertEquals("user", prompt.getRole());
    }
}
