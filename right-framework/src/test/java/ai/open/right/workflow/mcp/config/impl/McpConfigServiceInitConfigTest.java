package ai.open.right.workflow.mcp.config.impl;

import ai.open.right.resouce.PlaceholderResolver;
import ai.open.right.workflow.mcp.config.McpConfigInit;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;
import java.util.List;

public class McpConfigServiceInitConfigTest {

    @Test
    public void testInit() throws Exception {
        PlaceholderResolver placeholderResolver = EasyMock.createMock(PlaceholderResolver.class);
        McpConfigInit init = EasyMock.createMock(McpConfigInit.class);
        EasyMock.replay(placeholderResolver, init);
        List<McpConfigInit> initList = Arrays.asList(init);
        McpConfigServiceImpl.InitConfig initConfig = new McpConfigServiceImpl.InitConfig();
        initConfig.setMcpConfigInit(initList);
        initConfig.setPlaceholderResolver(placeholderResolver);
        initConfig.setUri("URI");
        McpConfigServiceImpl mcpConfigService = (McpConfigServiceImpl) initConfig.mcpConfigService();
        Assert.assertEquals(mcpConfigService.getMcpConfigInit(), initList);
        Assert.assertEquals(mcpConfigService.getUri(), "URI");
        Assert.assertEquals(mcpConfigService.getPlaceholderResolver(), placeholderResolver);
        EasyMock.verify(placeholderResolver, init);
    }
}
