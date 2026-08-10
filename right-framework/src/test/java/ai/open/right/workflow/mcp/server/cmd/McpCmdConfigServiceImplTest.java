package ai.open.right.workflow.mcp.server.cmd;

import ai.open.right.workflow.flow.config.McpExportConfig;
import org.junit.Assert;
import org.junit.Test;

public class McpCmdConfigServiceImplTest {

    @Test
    public void testFetch() throws Exception {
        McpExportConfig config = new McpExportConfig();
        McpCmdConfigServiceImpl mcpCmdConfigService = new McpCmdConfigServiceImpl();
        mcpCmdConfigService.export("A", config);
        Assert.assertEquals(config, mcpCmdConfigService.fetch("A"));
    }

    @Test(expected = IllegalArgumentException.class)
    public void testEmpty() throws Exception {
        McpCmdConfigServiceImpl mcpCmdConfigService = new McpCmdConfigServiceImpl();
        mcpCmdConfigService.fetch("A");
    }

    @Test
    public void testInit() throws Exception {
        McpCmdConfigServiceImpl.InitConfig initConfig = new McpCmdConfigServiceImpl.InitConfig();
        Assert.assertNotNull(initConfig.mcpCmdConfigService4Prompt());
        Assert.assertNotNull(initConfig.mcpCmdConfigService4Resources());
        Assert.assertNotNull(initConfig.mcpCmdConfigService4Tools());
        Assert.assertNotNull(initConfig.mcpCmdConfigService4ResourcesTemplates());
    }
}
