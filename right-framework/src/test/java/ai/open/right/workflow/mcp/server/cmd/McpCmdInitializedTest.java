package ai.open.right.workflow.mcp.server.cmd;

import ai.open.right.workflow.flow.config.McpExportConfig;
import ai.open.right.workflow.mcp.server.McpRequest;
import org.easymock.EasyMock;
import org.junit.Test;

public class McpCmdInitializedTest {

    @Test
    public void test() throws Exception {
        McpCmdInitialized mcpCmdInitialized = new McpCmdInitialized();
        mcpCmdInitialized.export(new McpExportConfig());
        McpRequest request = EasyMock.createMock(McpRequest.class);
        request.write(McpCmdInitialized.NOTIFIER);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(request);
        mcpCmdInitialized.cmd(request);
        EasyMock.verify(request);
    }

    @Test
    public void testInit() throws Exception {
        McpCmdInitialized.InitConfig initConfig = new McpCmdInitialized.InitConfig();
        McpCmdInitialized mcpCmdInitialize = initConfig.mcpInitialized();
        McpRequest request = EasyMock.createMock(McpRequest.class);
        request.write(McpCmdInitialized.NOTIFIER);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(request);
        mcpCmdInitialize.cmd(request);
        EasyMock.verify(request);
    }
}
