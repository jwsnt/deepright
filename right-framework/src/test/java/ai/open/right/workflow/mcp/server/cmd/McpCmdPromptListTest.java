package ai.open.right.workflow.mcp.server.cmd;

import ai.open.right.netty.mcp.server.NettyMcpRequest;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.config.McpExportConfig;
import ai.open.right.workflow.mcp.server.McpCmdConfigService;
import ai.open.right.workflow.mcp.server.McpRequest;
import ai.open.right.workflow.mcp.server.McpResponse;
import com.google.common.collect.ImmutableMap;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;

public class McpCmdPromptListTest {

    @Test
    public void testInit() throws Exception {
        McpExportConfig exportConfig = new McpExportConfig();
        exportConfig.setProperties(ImmutableMap.of("A", "B"));
        exportConfig.setRequired(Arrays.asList("X"));
        exportConfig.setDescription("Description");
        exportConfig.setWorkflow("WORKFLOW");
        exportConfig.setBiz("BIZ");
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        mcpCmdConfigService.export("BIZ@WORKFLOW", exportConfig);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(mcpCmdConfigService);
        McpCmdPromptList mcpCmdPromptList = new McpCmdPromptList();
        mcpCmdPromptList.setMcpCmdConfigService(mcpCmdConfigService);
        mcpCmdPromptList.export(exportConfig);
        Assert.assertEquals(1, mcpCmdPromptList.getMcpCmdExports().size());
        McpCmdPromptList.McpPromptExport mcpPromptExport = mcpCmdPromptList.getMcpCmdExports().getFirst();
        Assert.assertEquals("{\"inputSchema\":{\"type\":\"object\",\"properties\":{\"A\":\"B\"},\"required\":[\"X\"]},\"description\":\"Description\",\"name\":\"BIZ@WORKFLOW\"}", JsonUtils.write(mcpPromptExport));
        EasyMock.verify(mcpCmdConfigService);
    }

    @Test
    public void testInitWithOutPropertiesAndRequired() throws Exception {
        McpExportConfig exportConfig = new McpExportConfig();
        exportConfig.setDescription("Description");
        exportConfig.setWorkflow("WORKFLOW");
        exportConfig.setBiz("BIZ");
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        mcpCmdConfigService.export("BIZ@WORKFLOW", exportConfig);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(mcpCmdConfigService);
        McpCmdPromptList mcpCmdPromptList = new McpCmdPromptList();
        mcpCmdPromptList.setMcpCmdConfigService(mcpCmdConfigService);
        mcpCmdPromptList.export(exportConfig);
        Assert.assertEquals(1, mcpCmdPromptList.getMcpCmdExports().size());
        McpCmdPromptList.McpPromptExport mcpPromptExport = mcpCmdPromptList.getMcpCmdExports().getFirst();
        Assert.assertEquals("{\"inputSchema\":{\"type\":\"object\"},\"description\":\"Description\",\"name\":\"BIZ@WORKFLOW\"}", JsonUtils.write(mcpPromptExport));
        EasyMock.verify(mcpCmdConfigService);
    }

    @Test
    public void testExecute() throws Exception {
        McpExportConfig exportConfig = new McpExportConfig();
        exportConfig.setProperties(ImmutableMap.of("A", "B"));
        exportConfig.setRequired(Arrays.asList("X"));
        exportConfig.setDescription("Description");
        exportConfig.setWorkflow("WORKFLOW");
        exportConfig.setBiz("BIZ");
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        mcpCmdConfigService.export("BIZ@WORKFLOW", exportConfig);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(mcpCmdConfigService);
        McpCmdPromptList mcpCmdPromptList = new McpCmdPromptList();
        mcpCmdPromptList.setMcpCmdConfigService(mcpCmdConfigService);
        mcpCmdPromptList.export(exportConfig);
        McpRequest request = new NettyMcpRequest() {
            @Override
            public void write(McpResponse response) throws Exception {
                Assert.assertEquals("{\"result\":{\"prompts\":[{\"inputSchema\":{\"type\":\"object\",\"properties\":{\"A\":\"B\"},\"required\":[\"X\"]},\"description\":\"Description\",\"name\":\"BIZ@WORKFLOW\"}]},\"notifier\":false,\"wrap\":true}", JsonUtils.write(response));
            }
        };
        mcpCmdPromptList.cmd(request);
        EasyMock.verify(mcpCmdConfigService);
    }

    @Test
    public void testBuildName() throws Exception {
        McpExportConfig exportConfig = new McpExportConfig();
        exportConfig.setProperties(ImmutableMap.of("A", "B"));
        exportConfig.setRequired(Arrays.asList("X"));
        exportConfig.setDescription("Description");
        exportConfig.setWorkflow("WORKFLOW");
        exportConfig.setBiz("BIZ");
        exportConfig.setName("NAME");
        McpCmdPromptList mcpCmdPromptList = new McpCmdPromptList();
        Assert.assertEquals("NAME", mcpCmdPromptList.buildName(exportConfig));
    }

    @Test
    public void testBuildWithOutName() throws Exception {
        McpExportConfig exportConfig = new McpExportConfig();
        exportConfig.setProperties(ImmutableMap.of("A", "B"));
        exportConfig.setRequired(Arrays.asList("X"));
        exportConfig.setDescription("Description");
        exportConfig.setWorkflow("WORKFLOW");
        exportConfig.setBiz("BIZ");
        McpCmdPromptList mcpCmdPromptList = new McpCmdPromptList();
        Assert.assertEquals("BIZ@WORKFLOW", mcpCmdPromptList.buildName(exportConfig));
    }

    @Test
    public void testInitConfig() throws Exception {
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        EasyMock.replay(mcpCmdConfigService);
        McpCmdPromptList.InitConfig initConfig = new McpCmdPromptList.InitConfig();
        initConfig.setMcpCmdConfigService(mcpCmdConfigService);
        McpCmdPromptList promptList = initConfig.mcpCmdPromptList();
        Assert.assertEquals(mcpCmdConfigService, promptList.getMcpCmdConfigService());
        EasyMock.verify(mcpCmdConfigService);
    }
}
