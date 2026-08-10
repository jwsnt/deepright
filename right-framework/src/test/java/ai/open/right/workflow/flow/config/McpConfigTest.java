package ai.open.right.workflow.flow.config;

import org.junit.Assert;
import org.junit.Test;

public class McpConfigTest {

    @Test
    public void test() {
        McpConfig mcpConfig = new McpConfig();
        Assert.assertFalse(mcpConfig.hasExport());
        Assert.assertFalse(mcpConfig.hasRewriter());
        mcpConfig.setDynamic("DYNAMIC");
        mcpConfig.setName("NAME");
        mcpConfig.setTimeout(1000);
        mcpConfig.setClient("CLIENT");
        mcpConfig.setRewriter("Listener");
        McpExportConfig exportConfig = new McpExportConfig();
        mcpConfig.setExportConfig(exportConfig);
        Assert.assertTrue(mcpConfig.hasRewriter());
        Assert.assertTrue(mcpConfig.hasExport());
        Assert.assertEquals(exportConfig, mcpConfig.getExportConfig());
        Assert.assertEquals("Listener", mcpConfig.getRewriter());
        Assert.assertEquals("DYNAMIC", mcpConfig.getDynamic());
        Assert.assertEquals(Integer.valueOf(1000), mcpConfig.getTimeout());
        Assert.assertEquals("NAME", mcpConfig.getName());
        Assert.assertEquals("CLIENT", mcpConfig.getClient());
    }
    @Test
    public void testMergeNull() throws Exception {
        McpConfig config = new McpConfig();
        config.setRewriter("R");
        Assert.assertEquals("R", config.merge(null).getRewriter());
    }

    @Test
    public void testGetTimeoutNull() {
        McpConfig config = new McpConfig();
        config.setTimeout(null);
        Assert.assertNull(config.getTimeout());
    }
}
