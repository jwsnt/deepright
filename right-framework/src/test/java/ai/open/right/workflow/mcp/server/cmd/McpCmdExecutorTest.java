package ai.open.right.workflow.mcp.server.cmd;

import ai.open.right.ObjectBuilder;
import ai.open.right.integration.RightConfig;
import ai.open.right.integration.RightService;
import ai.open.right.workflow.flow.config.McpExportConfig;
import ai.open.right.workflow.mcp.server.McpCmdConfigService;
import ai.open.right.workflow.mcp.server.McpRequest;
import com.google.common.collect.ImmutableMap;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.concurrent.Future;

public class McpCmdExecutorTest {

    @Test
    public void testExport() throws Exception {
        McpCmdExportExecutor mcpCmdExecutor = new McpCmdExportExecutor() {
            @Override
            protected McpCmdResponse buildResponse(McpRequest mcpRequest, String content) throws Exception {
                return null;
            }

            @Override
            protected String buildQuery(McpRequest mcpRequest) throws Exception {
                return null;
            }
        };
        mcpCmdExecutor.export(new McpExportConfig());
    }

    @Test
    public void testExecute() throws Exception {
        RightConfig rightConfig = RightConfig.builder().build();
        McpCmdExportExecutor mcpCmdExecutor = new McpCmdExportExecutor() {

            @Override
            protected McpCmdResponse buildResponse(McpRequest mcpRequest, String content) throws Exception {
                Assert.assertEquals("HELLO WORLD", content);
                return null;
            }

            @Override
            protected String buildQuery(McpRequest mcpRequest) throws Exception {
                return null;
            }

            protected RightConfig buildRightConfig(McpRequest mcpRequest) throws Exception {
                return rightConfig;
            }
        };
        RightService rightService = EasyMock.createMock(RightService.class);
        Future<String> future = EasyMock.createMock(Future.class);
        EasyMock.expect(future.get()).andReturn("HELLO WORLD").anyTimes();
        EasyMock.expect(rightService.get(rightConfig)).andReturn(future).anyTimes();
        EasyMock.replay(rightService, future);
        mcpCmdExecutor.setRightService(rightService);
        McpRequest request = EasyMock.createMock(McpRequest.class);
        request.write(null);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getContent()).andReturn(ImmutableMap.of("params", ImmutableMap.of("name", "HELLO")));
        EasyMock.replay(request);
        mcpCmdExecutor.cmd(request);
        EasyMock.verify(request);
        EasyMock.verify(rightService, future);
    }

    @Test
    public void testSetGet() throws Exception {
        McpCmdExportExecutor mcpCmdExecutor = new McpCmdExportExecutor() {
            @Override
            protected McpCmdResponse buildResponse(McpRequest mcpRequest, String content) throws Exception {
                return null;
            }

            @Override
            protected String buildQuery(McpRequest mcpRequest) throws Exception {
                return null;
            }
        };
        McpCmdConfigService mcpCmdConfigService = EasyMock.createMock(McpCmdConfigService.class);
        RightService rightService = EasyMock.createMock(RightService.class);
        EasyMock.replay(mcpCmdConfigService, rightService);
        mcpCmdExecutor.setMcpCmdConfigService(mcpCmdConfigService);
        mcpCmdExecutor.setRightService(rightService);
        mcpCmdExecutor.setTimeout4Llm(Integer.valueOf(10086));
        Assert.assertEquals(mcpCmdConfigService, mcpCmdExecutor.getMcpCmdConfigService());
        Assert.assertEquals(rightService, mcpCmdExecutor.getRightService());
        Assert.assertEquals(Integer.valueOf(10086), mcpCmdExecutor.getTimeout4Llm());
        EasyMock.verify(mcpCmdConfigService, rightService);
    }
}
