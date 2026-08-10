package ai.open.right.workflow.mcp.server.cmd;

import ai.open.right.integration.RightConfig;
import ai.open.right.integration.RightService;
import ai.open.right.netty.mcp.server.NettyMcpRequest;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.config.McpExportConfig;
import ai.open.right.workflow.mcp.server.McpCmdConfigService;
import ai.open.right.workflow.mcp.server.McpRequest;
import ai.open.right.workflow.mcp.server.McpResponse;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.nio.charset.StandardCharsets;
import java.util.Map;
import java.util.concurrent.Future;

public class McpCmdToolsCallTest {

    @Test
    public void testExecute() throws Exception {
        String request = IOUtils.toString(ResourceUtils.getURL("classpath:MCP_Tools_call_request.json").openStream(), StandardCharsets.UTF_8);
        Map map = JsonUtils.read(request, Map.class);
        McpRequest mcpRequest = EasyMock.createMock(McpRequest.class);
        EasyMock.expect(mcpRequest.getContent()).andReturn((Map<String, Object>) map.get("content"));
        //EasyMock.expect(mcpRequest.getHeaders()).andReturn((Map<String, String>) map.get("headers"));
        mcpRequest.write(EasyMock.anyObject(McpResponse.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(mcpRequest);
        RightConfig rightConfig = RightConfig.builder().build();
        RightService rightService = EasyMock.createMock(RightService.class);
        Future<String> future = EasyMock.createMock(Future.class);
        EasyMock.expect(future.get()).andReturn("HELLO WORLD").anyTimes();
        EasyMock.expect(rightService.get(rightConfig)).andReturn(future).anyTimes();
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        McpExportConfig mcpExportConfig = new McpExportConfig();
        mcpExportConfig.setQuery("Query");
        EasyMock.expect(mcpCmdConfigService.fetch("mcp_server_tools_call:cr")).andReturn(mcpExportConfig).anyTimes();
        EasyMock.replay(mcpCmdConfigService, rightService, future);
        McpCmdToolsCall mcpCmdToolsCall = new McpCmdToolsCall() {
            protected RightConfig buildRightConfig(McpRequest mcpRequest) throws Exception {
                return rightConfig;
            }
        };
        mcpCmdToolsCall.setMcpCmdConfigService(mcpCmdConfigService);
        mcpCmdToolsCall.setRightService(rightService);
        mcpCmdToolsCall.cmd(mcpRequest);
        EasyMock.verify(mcpCmdConfigService, rightService, future, mcpRequest);
    }

    @Test
    public void testExecuteWithException() throws Exception {
        String request = IOUtils.toString(ResourceUtils.getURL("classpath:MCP_Tools_call_request.json").openStream(), StandardCharsets.UTF_8);
        Map map = JsonUtils.read(request, Map.class);
        McpRequest mcpRequest = EasyMock.createMock(McpRequest.class);
        EasyMock.expect(mcpRequest.getContent()).andReturn((Map<String, Object>) map.get("content"));
        //EasyMock.expect(mcpRequest.getHeaders()).andReturn((Map<String, String>) map.get("headers"));
        mcpRequest.write(EasyMock.anyObject(McpResponse.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(mcpRequest);
        RightConfig rightConfig = RightConfig.builder().build();
        RightService rightService = EasyMock.createMock(RightService.class);
        Future<String> future = EasyMock.createMock(Future.class);
        EasyMock.expect(future.get()).andThrow(new RuntimeException("EXP")).anyTimes();
        EasyMock.expect(rightService.get(rightConfig)).andReturn(future).anyTimes();
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        McpExportConfig mcpExportConfig = new McpExportConfig();
        mcpExportConfig.setQuery("Query");
        EasyMock.expect(mcpCmdConfigService.fetch("mcp_server_tools_call:cr")).andReturn(mcpExportConfig).anyTimes();
        EasyMock.replay(mcpCmdConfigService, rightService, future);
        McpCmdToolsCall mcpCmdToolsCall = new McpCmdToolsCall() {
            protected RightConfig buildRightConfig(McpRequest mcpRequest) throws Exception {
                return rightConfig;
            }
        };
        mcpCmdToolsCall.setMcpCmdConfigService(mcpCmdConfigService);
        mcpCmdToolsCall.setRightService(rightService);
        mcpCmdToolsCall.cmd(mcpRequest);
        EasyMock.verify(mcpCmdConfigService, rightService, mcpRequest, future);
    }

    @Test
    public void testBuildRightConfig() throws Exception {
        String request = IOUtils.toString(ResourceUtils.getURL("classpath:MCP_Tools_call_request.json").openStream(), StandardCharsets.UTF_8);
        Map map = JsonUtils.read(request, Map.class);
        NettyMcpRequest mcpRequest = NettyMcpRequest.builder()
                .content((Map<String, Object>) map.get("content"))
                .headers((Map<String, String>) map.get("headers"))
                .build().init();
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        McpExportConfig mcpExportConfig = new McpExportConfig();
        mcpExportConfig.setQuery("Query");
        mcpExportConfig.setWorkflow("WORKFLOW");
        mcpExportConfig.setBiz("BIZ");
        EasyMock.expect(mcpCmdConfigService.fetch("mcp_server_tools_call:cr")).andReturn(mcpExportConfig).anyTimes();
        EasyMock.replay(mcpCmdConfigService);
        McpCmdToolsCall mcpCmdToolsCall = new McpCmdToolsCall();
        mcpCmdToolsCall.setMcpCmdConfigService(mcpCmdConfigService);
        mcpCmdToolsCall.setTimeout4Llm(100086);
        RightConfig rightConfig = mcpCmdToolsCall.buildRightConfig(mcpRequest);
        Assert.assertEquals("{\"conversation\":\"conversation_10086\",\"chat\":\"chat_10086\",\"device\":\"device_10086\",\"trace\":\"trace_10086\",\"sec-fetch-mode\":\"cors\",\"content-length\":\"144\",\"accept-language\":\"*\",\"host\":\"127.0.0.1:9997\",\"connection\":\"keep-alive\",\"content-type\":\"application/json\",\"HOME\":\"/Users/shenjiawei\",\"accept-encoding\":\"gzip, deflate\",\"accept\":\"application/json, text/event-stream\",\"user-agent\":\"node\"}", JsonUtils.write(rightConfig.getMetadata()));
        Assert.assertEquals("device_10086", rightConfig.getUserContext().getDevice());
        Assert.assertEquals("conversation_10086", rightConfig.getConversation());
        Assert.assertEquals("{\"week\":220}", rightConfig.getQuery());
        Assert.assertEquals("chat_10086", rightConfig.getChat());
        Assert.assertEquals("trace_10086", rightConfig.getTrace());
        Assert.assertEquals(Integer.valueOf(100086), rightConfig.getTimeout());
        Assert.assertEquals("WORKFLOW", rightConfig.getWorkflow());
        Assert.assertEquals("BIZ", rightConfig.getBiz());
        EasyMock.verify(mcpCmdConfigService);
    }

    @Test
    public void testInit() throws Exception {
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        RightService rightService = EasyMock.createMock(RightService.class);
        EasyMock.replay(mcpCmdConfigService, rightService);
        McpCmdToolsCall.InitConfig initConfig = new McpCmdToolsCall.InitConfig();
        initConfig.setMcpCmdConfigService(mcpCmdConfigService);
        initConfig.setRightService(rightService);
        initConfig.setTimeout4Llm(10086);
        McpCmdToolsCall mcpCmdToolsCall = initConfig.mcpCmdToolsCall();
        Assert.assertEquals(mcpCmdToolsCall.getMcpCmdConfigService(), mcpCmdConfigService);
        Assert.assertEquals(mcpCmdToolsCall.getRightService(), rightService);
        Assert.assertEquals(mcpCmdToolsCall.getTimeout4Llm(), Integer.valueOf(10086));
        EasyMock.verify(mcpCmdConfigService, rightService);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = McpCmdToolsCall.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = McpCmdToolsCall.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
