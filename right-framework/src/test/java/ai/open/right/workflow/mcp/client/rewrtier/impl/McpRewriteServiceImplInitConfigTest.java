package ai.open.right.workflow.mcp.client.rewrtier.impl;

import ai.open.right.workflow.mcp.client.rewrtier.McpRewriter;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class McpRewriteServiceImplInitConfigTest {

    @Test
    public void testInit() throws Exception {
        Map<String, McpRewriter> rewriter = new HashMap<>();
        McpRewriteServiceImpl.InitConfig initConfig = new McpRewriteServiceImpl.InitConfig();
        initConfig.setRewriter(rewriter);
        McpRewriteServiceImpl empty = (McpRewriteServiceImpl) initConfig.mcpRewriteService();
        Assert.assertEquals(rewriter, empty.getRewriter());
    }
}
