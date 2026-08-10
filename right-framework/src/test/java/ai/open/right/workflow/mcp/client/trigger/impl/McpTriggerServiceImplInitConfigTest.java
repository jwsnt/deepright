package ai.open.right.workflow.mcp.client.trigger.impl;

import ai.open.right.workflow.mcp.client.trigger.McpTrigger;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class McpTriggerServiceImplInitConfigTest {

    @Test
    public void testInit() throws Exception {
        Map<String, McpTrigger> triggers = new HashMap<>();
        McpTriggerServiceImpl.InitConfig initConfig = new McpTriggerServiceImpl.InitConfig();
        initConfig.setTriggers(triggers);
        McpTriggerServiceImpl empty = (McpTriggerServiceImpl) initConfig.mcpTriggerService();
        Assert.assertEquals(triggers, empty.getTriggers());
    }
}
