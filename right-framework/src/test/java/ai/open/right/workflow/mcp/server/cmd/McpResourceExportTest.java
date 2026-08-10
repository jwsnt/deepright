package ai.open.right.workflow.mcp.server.cmd;

import org.junit.Assert;
import org.junit.Test;

public class McpResourceExportTest {

    @Test
    public void test() {
        McpCmdResourcesList.McpResourceExport export = McpCmdResourcesList.McpResourceExport.builder().build();
        export.setUri("URI");
        export.setDescription("DESC");
        export.setName("NAME");
        export.setMimeType("MIMETYPE");
        Assert.assertEquals("URI", export.getUri());
        Assert.assertEquals("DESC", export.getDescription());
        Assert.assertEquals("NAME", export.getName());
        Assert.assertEquals("MIMETYPE", export.getMimeType());
    }
}
