package ai.open.right.workflow.mcp.server.cmd;

import ai.open.right.workflow.flow.config.McpExportConfig;
import ai.open.right.workflow.mcp.server.McpRequest;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class McpCmdInitializeTest {

    @Test
    public void test() throws Exception {
        McpCmdInitialize mcpCmdInitialize = new McpCmdInitialize();
        mcpCmdInitialize.setProject("PROJECT");
        Assert.assertNull(mcpCmdInitialize.getInit());
        mcpCmdInitialize.init();
        Assert.assertNotNull(mcpCmdInitialize.getInit());
        McpRequest request = EasyMock.createMock(McpRequest.class);
        request.write(mcpCmdInitialize.getInit());
        EasyMock.replay(request);
        mcpCmdInitialize.cmd(request);
        EasyMock.verify(request);
        mcpCmdInitialize.export(new McpExportConfig());
    }

    @Test
    public void testInit() throws Exception {
        McpCmdInitialize.InitConfig initConfig = new McpCmdInitialize.InitConfig();
        initConfig.setProject("Project");
        McpCmdInitialize mcpCmdInitialize = initConfig.mcpInitialize();
        Assert.assertEquals(initConfig.getProject(), mcpCmdInitialize.getProject());
    }
}
