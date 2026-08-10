package ai.open.right.workflow.mcp.server.cmd;

import ai.open.right.integration.RightService;
import ai.open.right.netty.mcp.server.NettyMcpRequest;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.config.McpExportConfig;
import ai.open.right.workflow.mcp.server.McpCmdConfigService;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.Map;

public class McpCmdPromptGetTest {
    @Test
    public void testWithOneHistory() throws Exception {
        McpCmdPrompt history = new McpCmdPrompt();
        history.setRole(McpCmdPrompt.ROLE_USER);
        history.setContent("HELLO");
        McpCmdPromptGet mcpCmdPromptGet = new McpCmdPromptGet();
        McpCmdResponse mcpCmdResponse = mcpCmdPromptGet.buildResponse(null, JsonUtils.write(Arrays.asList(history)));
        Assert.assertEquals("[{\"result\":{\"messages\":[{\"role\":\"user\",\"content\":{\"type\":\"text\",\"text\":\"HELLO\"}}]},\"notifier\":false,\"wrap\":true}]", JsonUtils.write(Arrays.asList(mcpCmdResponse)));
    }

    @Test
    public void testWithTwoHistory() throws Exception {
        McpCmdPrompt history1 = new McpCmdPrompt();
        history1.setRole(McpCmdPrompt.ROLE_USER);
        history1.setContent("HELLO");
        McpCmdPrompt history2 = new McpCmdPrompt();
        history2.setRole(McpCmdPrompt.ROLE_ASSISTANT);
        history2.setContent("WORLD");
        McpCmdPromptGet mcpCmdPromptGet = new McpCmdPromptGet();
        McpCmdResponse mcpCmdResponse = mcpCmdPromptGet.buildResponse(null, JsonUtils.write(Arrays.asList(history1, history2)));
        Assert.assertEquals("{\"result\":{\"messages\":[{\"role\":\"user\",\"content\":{\"type\":\"text\",\"text\":\"HELLO\"}},{\"role\":\"assistant\",\"content\":{\"type\":\"text\",\"text\":\"WORLD\"}}]},\"notifier\":false,\"wrap\":true}", JsonUtils.write(mcpCmdResponse));
    }

    @Test
    public void testWithInValidHistory() throws Exception {
        McpCmdPromptGet mcpCmdPromptGet = new McpCmdPromptGet();
        McpCmdResponse mcpCmdResponse = mcpCmdPromptGet.buildResponse(null, "HELLO");
        Assert.assertEquals("{\"result\":{\"messages\":[{\"role\":\"user\",\"content\":{\"type\":\"text\",\"text\":\"HELLO\"}}]},\"notifier\":false,\"wrap\":true}", JsonUtils.write(mcpCmdResponse));
    }

    @Test
    public void testWithBuildQueryAndQuery() throws Exception {
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        McpExportConfig mcpExportConfig = new McpExportConfig();
        mcpExportConfig.setQuery("Query");
        EasyMock.expect(mcpCmdConfigService.fetch("mcp_server_tools_call:cr")).andReturn(mcpExportConfig).anyTimes();
        EasyMock.replay(mcpCmdConfigService);
        McpCmdPromptGet mcpCmdPromptGet = new McpCmdPromptGet();
        mcpCmdPromptGet.setMcpCmdConfigService(mcpCmdConfigService);
        String request = IOUtils.toString(ResourceUtils.getURL("classpath:MCP_Prompt_get_request.json").openStream(), StandardCharsets.UTF_8);
        Map map = JsonUtils.read(request, Map.class);
        NettyMcpRequest mcpRequest = NettyMcpRequest.builder()
                .content((Map<String, Object>) map.get("content"))
                .headers((Map<String, String>) map.get("headers"))
                .build();
        Assert.assertEquals("Query", mcpCmdPromptGet.buildQuery(mcpRequest));
        EasyMock.verify(mcpCmdConfigService);
    }

    @Test
    public void testWithBuildQueryAndName() throws Exception {
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        McpExportConfig mcpExportConfig = new McpExportConfig();
        mcpExportConfig.setName("NAME");
        EasyMock.expect(mcpCmdConfigService.fetch("mcp_server_tools_call:cr")).andReturn(mcpExportConfig).anyTimes();
        EasyMock.replay(mcpCmdConfigService);
        McpCmdPromptGet mcpCmdPromptGet = new McpCmdPromptGet();
        mcpCmdPromptGet.setMcpCmdConfigService(mcpCmdConfigService);
        String request = IOUtils.toString(ResourceUtils.getURL("classpath:MCP_Prompt_get_request.json").openStream(), StandardCharsets.UTF_8);
        Map map = JsonUtils.read(request, Map.class);
        NettyMcpRequest mcpRequest = NettyMcpRequest.builder()
                .content((Map<String, Object>) map.get("content"))
                .headers((Map<String, String>) map.get("headers"))
                .build();
        Assert.assertEquals("NAME", mcpCmdPromptGet.buildQuery(mcpRequest));
        EasyMock.verify(mcpCmdConfigService);
    }

    @Test
    public void testInit() throws Exception {
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        RightService rightService = EasyMock.createMock(RightService.class);
        EasyMock.replay(mcpCmdConfigService, rightService);
        McpCmdPromptGet.InitConfig initConfig = new McpCmdPromptGet.InitConfig();
        initConfig.setMcpCmdConfigService(mcpCmdConfigService);
        initConfig.setRightService(rightService);
        initConfig.setTimeout4Llm(10086);
        McpCmdPromptGet mcpCmdPromptGet = initConfig.mcpCmdPromptGet();
        Assert.assertEquals(mcpCmdPromptGet.getMcpCmdConfigService(), mcpCmdConfigService);
        Assert.assertEquals(mcpCmdPromptGet.getRightService(), rightService);
        Assert.assertEquals(mcpCmdPromptGet.getTimeout4Llm(), Integer.valueOf(10086));
        EasyMock.verify(mcpCmdConfigService, rightService);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = McpCmdPromptGet.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = McpCmdPromptGet.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

}
