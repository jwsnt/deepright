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

public class McpCmdResourcesListTest {

    @Test
    public void testInit() throws Exception {
        McpExportConfig exportConfig = new McpExportConfig();
        exportConfig.setProperties(ImmutableMap.of("A", "B"));
        exportConfig.setRequired(Arrays.asList("X"));
        exportConfig.setDescription("Description");
        exportConfig.setWorkflow("WORKFLOW");
        exportConfig.setBiz("BIZ");
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        mcpCmdConfigService.export("BIZ/WORKFLOW", exportConfig);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(mcpCmdConfigService);
        McpCmdResourcesList mcpCmdResourcesList = new McpCmdResourcesList();
        mcpCmdResourcesList.setMcpCmdConfigService(mcpCmdConfigService);
        mcpCmdResourcesList.export(exportConfig);
        Assert.assertEquals(1, mcpCmdResourcesList.getMcpCmdExports().size());
        McpCmdResourcesList.McpResourceExport mcpResourceExport = mcpCmdResourcesList.getMcpCmdExports().getFirst();
        Assert.assertEquals("{\"description\":\"Description\",\"name\":\"BIZ@WORKFLOW\",\"uri\":\"BIZ/WORKFLOW\"}", JsonUtils.write(mcpResourceExport));
        EasyMock.verify(mcpCmdConfigService);
    }

    @Test
    public void testInitWithOutPropertiesAndRequired() throws Exception {
        McpExportConfig exportConfig = new McpExportConfig();
        exportConfig.setDescription("Description");
        exportConfig.setWorkflow("WORKFLOW");
        exportConfig.setBiz("BIZ");
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        mcpCmdConfigService.export("BIZ/WORKFLOW", exportConfig);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(mcpCmdConfigService);
        McpCmdResourcesList mcpCmdResourcesList = new McpCmdResourcesList();
        mcpCmdResourcesList.setMcpCmdConfigService(mcpCmdConfigService);
        mcpCmdResourcesList.export(exportConfig);
        Assert.assertEquals(1, mcpCmdResourcesList.getMcpCmdExports().size());
        McpCmdResourcesList.McpResourceExport mcpResourceExport = mcpCmdResourcesList.getMcpCmdExports().getFirst();
        Assert.assertEquals("{\"description\":\"Description\",\"name\":\"BIZ@WORKFLOW\",\"uri\":\"BIZ/WORKFLOW\"}", JsonUtils.write(mcpResourceExport));
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
        mcpCmdConfigService.export("BIZ/WORKFLOW", exportConfig);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(mcpCmdConfigService);
        McpCmdResourcesList mcpCmdResourcesList = new McpCmdResourcesList();
        mcpCmdResourcesList.setMcpCmdConfigService(mcpCmdConfigService);
        mcpCmdResourcesList.export(exportConfig);
        McpRequest request = new NettyMcpRequest() {
            @Override
            public void write(McpResponse response) throws Exception {
                Assert.assertEquals("{\"result\":{\"resources\":[{\"description\":\"Description\",\"name\":\"BIZ@WORKFLOW\",\"uri\":\"BIZ/WORKFLOW\"}]},\"notifier\":false,\"wrap\":true}", JsonUtils.write(response));
            }
        };
        mcpCmdResourcesList.cmd(request);
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
        McpCmdResourcesList mcpCmdResourcesList = new McpCmdResourcesList();
        Assert.assertEquals("NAME", mcpCmdResourcesList.buildName(exportConfig));
    }

    @Test
    public void testBuildWithOutName() throws Exception {
        McpExportConfig exportConfig = new McpExportConfig();
        exportConfig.setProperties(ImmutableMap.of("A", "B"));
        exportConfig.setRequired(Arrays.asList("X"));
        exportConfig.setDescription("Description");
        exportConfig.setWorkflow("WORKFLOW");
        exportConfig.setBiz("BIZ");
        McpCmdResourcesList mcpCmdResourcesList = new McpCmdResourcesList();
        Assert.assertEquals("BIZ@WORKFLOW", mcpCmdResourcesList.buildName(exportConfig));
    }

    @Test
    public void testBuildUri() throws Exception {
        McpExportConfig exportConfig = new McpExportConfig();
        exportConfig.setProperties(ImmutableMap.of("A", "B"));
        exportConfig.setRequired(Arrays.asList("X"));
        exportConfig.setDescription("Description");
        exportConfig.setWorkflow("WORKFLOW");
        exportConfig.setBiz("BIZ");
        exportConfig.setUri("NAME");
        McpCmdResourcesList mcpCmdResourcesList = new McpCmdResourcesList();
        Assert.assertEquals("NAME", mcpCmdResourcesList.buildUri(exportConfig));
    }

    @Test
    public void testBuildWithOutUri() throws Exception {
        McpExportConfig exportConfig = new McpExportConfig();
        exportConfig.setProperties(ImmutableMap.of("A", "B"));
        exportConfig.setRequired(Arrays.asList("X"));
        exportConfig.setDescription("Description");
        exportConfig.setWorkflow("WORKFLOW");
        exportConfig.setBiz("BIZ");
        McpCmdResourcesList mcpCmdResourcesList = new McpCmdResourcesList();
        Assert.assertEquals("BIZ/WORKFLOW", mcpCmdResourcesList.buildUri(exportConfig));
    }

    @Test
    public void testInitConfig() throws Exception {
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        EasyMock.replay(mcpCmdConfigService);
        McpCmdResourcesList.InitConfig initConfig = new McpCmdResourcesList.InitConfig();
        initConfig.setMcpCmdConfigService(mcpCmdConfigService);
        McpCmdResourcesList mcpCmdResourcesList = initConfig.mcpCmdResourcesList();
        Assert.assertEquals(mcpCmdConfigService, mcpCmdResourcesList.getMcpCmdConfigService());
        EasyMock.verify(mcpCmdConfigService);
    }
}
