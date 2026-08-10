package ai.open.right.workflow.mcp.server.cmd;

import org.junit.Assert;
import org.junit.Test;

public class McpResourceTemplateExportTest {

    @Test
    public void test() {
        McpCmdResourcesTemplatesList.McpResourceTemplateExport export = McpCmdResourcesTemplatesList.McpResourceTemplateExport.builder().build();
        export.setUriTemplate("URI");
        export.setName("NAME");
        export.setMimeType("MIME");
        export.setDescription("DESC");
        Assert.assertEquals("URI", export.getUriTemplate());
        Assert.assertEquals("NAME", export.getName());
        Assert.assertEquals("MIME", export.getMimeType());
        Assert.assertEquals("DESC", export.getDescription());
    }
}
