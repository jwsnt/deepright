package ai.open.right.workflow.mcp.server.cmd;

import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class McpPromptExportTest {

    @Test
    public void test() {
        McpCmdPromptList.McpPromptExport export = McpCmdPromptList.McpPromptExport.builder().build();
        Map<String, Object> input = new HashMap<>();
        export.setDescription("DES");
        export.setName("NAME");
        export.setInputSchema(input);
        Assert.assertEquals(input, export.getInputSchema());
        Assert.assertEquals("DES", export.getDescription());
        Assert.assertEquals("NAME", export.getName());
    }
}
