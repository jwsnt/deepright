package ai.open.right.workflow.mcp.server.cmd;

import ai.open.right.integration.RightService;
import ai.open.right.netty.mcp.server.NettyMcpRequest;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.config.McpExportConfig;
import ai.open.right.workflow.mcp.server.McpCmdConfigService;
import ai.open.right.workflow.mcp.server.McpRequest;
import com.google.common.collect.ImmutableMap;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.Map;

public class McpCmdResourcesReadTest {

    @Test
    public void testBuildResponse() throws Exception {
        String request = IOUtils.toString(ResourceUtils.getURL("classpath:MCP_Resource_read_request.json").openStream(), StandardCharsets.UTF_8);
        Map map = JsonUtils.read(request, Map.class);
        NettyMcpRequest mcpRequest = NettyMcpRequest.builder()
                .content((Map<String, Object>) map.get("content"))
                .headers((Map<String, String>) map.get("headers"))
                .build();
        McpCmdResourcesRead mcpCmdResourcesRead = new McpCmdResourcesRead();
        Assert.assertEquals("{\"result\":{\"contents\":[{\"uri\":\"\",\"mimeType\":\"text/plain\",\"text\":\"HELLO WORLD\"}]},\"notifier\":false,\"wrap\":true}", JsonUtils.write(mcpCmdResourcesRead.buildResponse(mcpRequest, "HELLO WORLD")));
    }

    @Test
    public void testBuildDimension() throws Exception {
        String request = IOUtils.toString(ResourceUtils.getURL("classpath:MCP_Resource_read_request.json").openStream(), StandardCharsets.UTF_8);
        Map map = JsonUtils.read(request, Map.class);
        NettyMcpRequest mcpRequest = NettyMcpRequest.builder()
                .content((Map<String, Object>) map.get("content"))
                .headers((Map<String, String>) map.get("headers"))
                .build();
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        McpExportConfig mcpExportConfig = new McpExportConfig();
        mcpExportConfig.setBiz("mcp_server_tools_call");
        mcpExportConfig.setWorkflow("cr");
        EasyMock.expect(mcpCmdConfigService.fetch("mcp_server_tools_call/cr")).andReturn(mcpExportConfig).anyTimes();
        EasyMock.replay(mcpCmdConfigService);
        McpCmdResourcesRead mcpCmdResourcesRead = new McpCmdResourcesRead();
        mcpCmdResourcesRead.setMcpCmdConfigService(mcpCmdConfigService);
        Assert.assertEquals("[mcp_server_tools_call, cr]", Arrays.toString(mcpCmdResourcesRead.buildDimension(mcpRequest)));
        EasyMock.verify(mcpCmdConfigService);
    }

    @Test
    public void testWithBuildQuery() throws Exception {
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        McpExportConfig mcpExportConfig = new McpExportConfig();
        mcpExportConfig.setQuery("Query");
        EasyMock.expect(mcpCmdConfigService.fetch("mcp_server_tools_call/cr")).andReturn(mcpExportConfig).anyTimes();
        EasyMock.replay(mcpCmdConfigService);
        McpCmdResourcesRead mcpCmdResourcesRead = new McpCmdResourcesRead();
        mcpCmdResourcesRead.setMcpCmdConfigService(mcpCmdConfigService);
        String request = IOUtils.toString(ResourceUtils.getURL("classpath:MCP_Resource_read_request.json").openStream(), StandardCharsets.UTF_8);
        Map map = JsonUtils.read(request, Map.class);
        NettyMcpRequest mcpRequest = NettyMcpRequest.builder()
                .content((Map<String, Object>) map.get("content"))
                .headers((Map<String, String>) map.get("headers"))
                .build();
        Assert.assertEquals("Query", mcpCmdResourcesRead.buildQuery(mcpRequest));
        EasyMock.verify(mcpCmdConfigService);
    }

    @Test
    public void testWithBuildQueryThreePart() throws Exception {
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        McpExportConfig mcpExportConfig = new McpExportConfig();
        mcpExportConfig.setQuery("Query");
        EasyMock.expect(mcpCmdConfigService.fetch("mcp_server_tools_call/cr@")).andReturn(mcpExportConfig).anyTimes();
        EasyMock.replay(mcpCmdConfigService);
        McpCmdResourcesRead mcpCmdResourcesRead = new McpCmdResourcesRead();
        mcpCmdResourcesRead.setMcpCmdConfigService(mcpCmdConfigService);
        String request = IOUtils.toString(ResourceUtils.getURL("classpath:MCP_Resource_template_read_request.json").openStream(), StandardCharsets.UTF_8);
        Map map = JsonUtils.read(request, Map.class);
        NettyMcpRequest mcpRequest = NettyMcpRequest.builder()
                .content((Map<String, Object>) map.get("content"))
                .headers((Map<String, String>) map.get("headers"))
                .build();
        Assert.assertEquals("abc", mcpCmdResourcesRead.buildQuery(mcpRequest));
        EasyMock.verify(mcpCmdConfigService);
    }

    @Test
    public void testCheck() throws Exception {
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        McpExportConfig mcpExportConfig = new McpExportConfig();
        mcpExportConfig.setQuery("Query");
        EasyMock.expect(mcpCmdConfigService.fetch("mcp_server_tools_call/cr")).andReturn(mcpExportConfig).anyTimes();
        EasyMock.replay(mcpCmdConfigService);
        McpCmdResourcesRead mcpCmdResourcesRead = new McpCmdResourcesRead();
        mcpCmdResourcesRead.setMcpCmdConfigService(mcpCmdConfigService);
        String request = IOUtils.toString(ResourceUtils.getURL("classpath:MCP_Resource_template_read_request.json").openStream(), StandardCharsets.UTF_8);
        Map map = JsonUtils.read(request, Map.class);
        NettyMcpRequest mcpRequest = NettyMcpRequest.builder()
                .content((Map<String, Object>) map.get("content"))
                .headers((Map<String, String>) map.get("headers"))
                .build();
        mcpCmdResourcesRead.checkRequest(mcpRequest);
        EasyMock.verify(mcpCmdConfigService);
    }

    @Test
    public void testInit() throws Exception {
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        RightService rightService = EasyMock.createMock(RightService.class);
        EasyMock.replay(mcpCmdConfigService, rightService);
        McpCmdResourcesRead.InitConfig initConfig = new McpCmdResourcesRead.InitConfig();
        initConfig.setMcpCmdConfigService(mcpCmdConfigService);
        initConfig.setRightService(rightService);
        initConfig.setTimeout4Llm(10086);
        McpCmdResourcesRead mcpCmdResourcesRead = initConfig.mcpCmdResourcesRead();
        Assert.assertEquals(mcpCmdResourcesRead.getMcpCmdConfigService(), mcpCmdConfigService);
        Assert.assertEquals(mcpCmdResourcesRead.getRightService(), rightService);
        Assert.assertEquals(mcpCmdResourcesRead.getTimeout4Llm(), Integer.valueOf(10086));
        EasyMock.verify(mcpCmdConfigService, rightService);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = McpCmdResourcesRead.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = McpCmdResourcesRead.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCmdParamsNull() throws Exception {
        McpCmdResourcesRead cmd = new McpCmdResourcesRead();
        McpRequest request = EasyMock.createMock(McpRequest.class);
        EasyMock.expect(request.getContent()).andReturn(ImmutableMap.of("A", "B")).anyTimes();
        EasyMock.expect(request.getId()).andReturn("1").anyTimes();
        EasyMock.replay(request);
        try {
            cmd.cmd(request);
        } finally {
            EasyMock.verify(request);
        }
    }
}
