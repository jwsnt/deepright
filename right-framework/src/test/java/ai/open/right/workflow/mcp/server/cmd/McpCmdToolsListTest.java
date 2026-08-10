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

public class McpCmdToolsListTest {

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
        McpCmdToolsList mcpCmdToolsList = new McpCmdToolsList();
        mcpCmdToolsList.setMcpCmdConfigService(mcpCmdConfigService);
        mcpCmdToolsList.export(exportConfig);
        Assert.assertEquals(1, mcpCmdToolsList.getMcpCmdExports().size());
        McpCmdToolsList.McpToolExport mcpToolExport = mcpCmdToolsList.getMcpCmdExports().getFirst();
        Assert.assertEquals("{\"inputSchema\":{\"type\":\"object\",\"properties\":{\"A\":\"B\"},\"required\":[\"X\"]},\"description\":\"Description\",\"name\":\"BIZ@WORKFLOW\"}", JsonUtils.write(mcpToolExport));
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
        McpCmdToolsList mcpCmdToolsList = new McpCmdToolsList();
        mcpCmdToolsList.setMcpCmdConfigService(mcpCmdConfigService);
        mcpCmdToolsList.export(exportConfig);
        Assert.assertEquals(1, mcpCmdToolsList.getMcpCmdExports().size());
        McpCmdToolsList.McpToolExport mcpToolExport = mcpCmdToolsList.getMcpCmdExports().getFirst();
        Assert.assertEquals("{\"inputSchema\":{\"type\":\"object\"},\"description\":\"Description\",\"name\":\"BIZ@WORKFLOW\"}", JsonUtils.write(mcpToolExport));
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
        McpCmdToolsList mcpCmdToolsList = new McpCmdToolsList();
        mcpCmdToolsList.setMcpCmdConfigService(mcpCmdConfigService);
        mcpCmdToolsList.export(exportConfig);
        McpRequest request = new NettyMcpRequest() {
            @Override
            public void write(McpResponse response) throws Exception {
                Assert.assertEquals("{\"result\":{\"tools\":[{\"inputSchema\":{\"type\":\"object\",\"properties\":{\"A\":\"B\"},\"required\":[\"X\"]},\"description\":\"Description\",\"name\":\"BIZ@WORKFLOW\"}]},\"notifier\":false,\"wrap\":true}", JsonUtils.write(response));
            }
        };
        mcpCmdToolsList.cmd(request);
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
        McpCmdToolsList mcpCmdToolsList = new McpCmdToolsList();
        Assert.assertEquals("NAME", mcpCmdToolsList.buildName(exportConfig));
    }

    @Test
    public void testBuildWithOutName() throws Exception {
        McpExportConfig exportConfig = new McpExportConfig();
        exportConfig.setProperties(ImmutableMap.of("A", "B"));
        exportConfig.setRequired(Arrays.asList("X"));
        exportConfig.setDescription("Description");
        exportConfig.setWorkflow("WORKFLOW");
        exportConfig.setBiz("BIZ");
        McpCmdToolsList mcpCmdToolsList = new McpCmdToolsList();
        Assert.assertEquals("BIZ@WORKFLOW", mcpCmdToolsList.buildName(exportConfig));
    }

    @Test
    public void testInitConfig() throws Exception {
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        EasyMock.replay(mcpCmdConfigService);
        McpCmdToolsList.InitConfig initConfig = new McpCmdToolsList.InitConfig();
        initConfig.setMcpCmdConfigService(mcpCmdConfigService);
        McpCmdToolsList mcpCmdToolsList = initConfig.mcpToolsList();
        Assert.assertEquals(mcpCmdConfigService, mcpCmdToolsList.getMcpCmdConfigService());
        EasyMock.verify(mcpCmdConfigService);
    }
}
