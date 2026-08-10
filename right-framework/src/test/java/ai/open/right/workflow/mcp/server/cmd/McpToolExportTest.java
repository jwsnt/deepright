package ai.open.right.workflow.mcp.server.cmd;

import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class McpToolExportTest {

    @Test
    public void test() {
        Map<String, Object> input = new HashMap<>();
        McpCmdToolsList.McpToolExport mcpToolExport = McpCmdToolsList.McpToolExport.builder().build();
        mcpToolExport.setName("NAME");
        mcpToolExport.setDescription("DESC");
        mcpToolExport.setInputSchema(input);
        Assert.assertEquals(input, mcpToolExport.getInputSchema());
        Assert.assertEquals("NAME", mcpToolExport.getName());
        Assert.assertEquals("DESC", mcpToolExport.getDescription());
    }
}
