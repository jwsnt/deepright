package ai.open.right.workflow.mcp.client.utils;

import ai.open.right.workflow.mcp.client.utils.McpContentUtils;
import com.google.common.collect.ImmutableMap;
import org.junit.Assert;
import org.junit.Test;

import java.util.Collections;
import java.util.Date;

public class McpContentUtilsTest {

    @Test
    public void testError1() throws Exception {
        Assert.assertNull(McpContentUtils.error(Collections.singletonMap("KEY", new Date())));
    }

    @Test
    public void testError2() throws Exception {
        Assert.assertEquals("OK", McpContentUtils.error(ImmutableMap.of("type", "text/plain", "text", "OK")));
    }
}
