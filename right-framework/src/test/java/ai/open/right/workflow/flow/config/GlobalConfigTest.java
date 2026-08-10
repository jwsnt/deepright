package ai.open.right.workflow.flow.config;

import com.google.common.collect.ImmutableMap;
import org.junit.Assert;
import org.junit.Test;

public class GlobalConfigTest {

    @Test
    public void testMerge1() throws Exception {
        GlobalConfig g1 = new GlobalConfig();
        Assert.assertFalse(g1.hasGlobalConfig());
        g1.setGlobalConfig(ImmutableMap.of("A", "B"));
        Assert.assertTrue(g1.hasGlobalConfig());
        GlobalConfig g2 = new GlobalConfig();
        Assert.assertEquals("B", g2.merge(g1).getGlobalConfig().get("A"));
    }

    @Test
    public void testMerge2() throws Exception {
        GlobalConfig g1 = new GlobalConfig();
        g1.setGlobalConfig(ImmutableMap.of("A", "B1", "C", "D"));
        GlobalConfig g2 = new GlobalConfig();
        g2.setGlobalConfig(ImmutableMap.of("A", "B2"));
        Assert.assertEquals("B2", g2.merge(g1).getGlobalConfig().get("A"));
        Assert.assertEquals("D", g2.merge(g1).getGlobalConfig().get("C"));
    }
}
