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

public class McpCmdResourcesTemplatesListTest {


    @Test
    public void testInit() throws Exception {
        McpExportConfig exportConfig = new McpExportConfig();
        exportConfig.setProperties(ImmutableMap.of("A", "B"));
        exportConfig.setRequired(Arrays.asList("X"));
        exportConfig.setDescription("Description");
        exportConfig.setWorkflow("WORKFLOW");
        exportConfig.setBiz("BIZ");
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        mcpCmdConfigService.export("BIZ/WORKFLOW@", exportConfig);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(mcpCmdConfigService);
        McpCmdResourcesTemplatesList mcpCmdResourcesList = new McpCmdResourcesTemplatesList();
        mcpCmdResourcesList.setMcpCmdConfigService(mcpCmdConfigService);
        mcpCmdResourcesList.export(exportConfig);
        Assert.assertEquals(1, mcpCmdResourcesList.getMcpCmdExports().size());
        McpCmdResourcesTemplatesList.McpResourceTemplateExport mcpResourceExport = mcpCmdResourcesList.getMcpCmdExports().getFirst();
        Assert.assertEquals("{\"description\":\"Description\",\"uriTemplate\":\"BIZ/WORKFLOW/{query}\",\"name\":\"BIZ@WORKFLOW\"}", JsonUtils.write(mcpResourceExport));
        EasyMock.verify(mcpCmdConfigService);
    }

    @Test
    public void testInitWithOutPropertiesAndRequired() throws Exception {
        McpExportConfig exportConfig = new McpExportConfig();
        exportConfig.setDescription("Description");
        exportConfig.setWorkflow("WORKFLOW");
        exportConfig.setBiz("BIZ");
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        mcpCmdConfigService.export("BIZ/WORKFLOW@", exportConfig);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(mcpCmdConfigService);
        McpCmdResourcesTemplatesList mcpCmdResourcesList = new McpCmdResourcesTemplatesList();
        mcpCmdResourcesList.setMcpCmdConfigService(mcpCmdConfigService);
        mcpCmdResourcesList.export(exportConfig);
        Assert.assertEquals(1, mcpCmdResourcesList.getMcpCmdExports().size());
        McpCmdResourcesTemplatesList.McpResourceTemplateExport mcpResourceExport = mcpCmdResourcesList.getMcpCmdExports().getFirst();
        Assert.assertEquals("{\"description\":\"Description\",\"uriTemplate\":\"BIZ/WORKFLOW/{query}\",\"name\":\"BIZ@WORKFLOW\"}", JsonUtils.write(mcpResourceExport));
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
        mcpCmdConfigService.export("BIZ/WORKFLOW@", exportConfig);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(mcpCmdConfigService);
        McpCmdResourcesTemplatesList mcpCmdResourcesList = new McpCmdResourcesTemplatesList();
        mcpCmdResourcesList.setMcpCmdConfigService(mcpCmdConfigService);
        mcpCmdResourcesList.export(exportConfig);
        McpRequest request = new NettyMcpRequest() {
            @Override
            public void write(McpResponse response) throws Exception {
                Assert.assertEquals("{\"result\":{\"resourceTemplates\":[{\"description\":\"Description\",\"uriTemplate\":\"BIZ/WORKFLOW/{query}\",\"name\":\"BIZ@WORKFLOW\"}]},\"notifier\":false,\"wrap\":true}", JsonUtils.write(response));
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
        McpCmdResourcesTemplatesList mcpCmdResourcesList = new McpCmdResourcesTemplatesList();
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
        McpCmdResourcesTemplatesList mcpCmdResourcesList = new McpCmdResourcesTemplatesList();
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
        McpCmdResourcesTemplatesList mcpCmdResourcesList = new McpCmdResourcesTemplatesList();
        Assert.assertEquals("BIZ/WORKFLOW/{query}", mcpCmdResourcesList.buildUriTemplate(exportConfig));
    }

    @Test
    public void testBuildWithOutUri() throws Exception {
        McpExportConfig exportConfig = new McpExportConfig();
        exportConfig.setProperties(ImmutableMap.of("A", "B"));
        exportConfig.setRequired(Arrays.asList("X"));
        exportConfig.setDescription("Description");
        exportConfig.setWorkflow("WORKFLOW");
        exportConfig.setBiz("BIZ");
        McpCmdResourcesTemplatesList mcpCmdResourcesList = new McpCmdResourcesTemplatesList();
        Assert.assertEquals("BIZ/WORKFLOW/{query}", mcpCmdResourcesList.buildUriTemplate(exportConfig));
    }

    @Test
    public void testInitConfig() throws Exception {
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        EasyMock.replay(mcpCmdConfigService);
        McpCmdResourcesTemplatesList.InitConfig initConfig = new McpCmdResourcesTemplatesList.InitConfig();
        initConfig.setMcpCmdConfigService(mcpCmdConfigService);
        McpCmdResourcesTemplatesList mcpCmdResourcesList = initConfig.mcpCmdResourcesTemplatesList();
        Assert.assertEquals(mcpCmdConfigService, mcpCmdResourcesList.getMcpCmdConfigService());
        EasyMock.verify(mcpCmdConfigService);
    }
    @Test
    public void testCmdTemplatesNull() throws Exception {
        McpCmdResourcesTemplatesList cmd = new McpCmdResourcesTemplatesList();
        McpRequest request = EasyMock.createMock(McpRequest.class);
        EasyMock.expect(request.getId()).andReturn("1").anyTimes();
        request.write(EasyMock.anyObject());
        EasyMock.expectLastCall().once();
        EasyMock.replay(request);
        cmd.cmd(request);
        EasyMock.verify(request);
    }
}
