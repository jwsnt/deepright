package ai.open.right.workflow.mcp.server.cmd;

import ai.open.right.integration.RightService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class McpCmdInitConfigTest {

    @Test
    public void testInit() throws Exception {
        RightService rightService = EasyMock.createMock(RightService.class);
        EasyMock.replay(rightService);
        McpCmdExportExecutor.McpCmdInitConfig initConfig = new McpCmdExportExecutor.McpCmdInitConfig();
        initConfig.setRightService(rightService);
        initConfig.setTimeout4Llm(10086);
        Assert.assertEquals(rightService, initConfig.getRightService());
        Assert.assertEquals(Integer.valueOf(10086), initConfig.getTimeout4Llm());
        EasyMock.verify(rightService);
    }
}
