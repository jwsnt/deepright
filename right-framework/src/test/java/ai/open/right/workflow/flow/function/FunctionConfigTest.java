package ai.open.right.workflow.flow.function;

import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class FunctionConfigTest {

    @Test
    public void testMergeWithNullOther() throws Exception {
        FunctionConfig base = new FunctionConfig();
        base.setName("N1");
        base.setResource("R1");
        base.setTimeout(100);
        base.setOriginal(true);
        Map<String, String> env = new HashMap<>();
        env.put("k", "v");
        base.setEnvironment(env);
        Assert.assertEquals(base, base.merge(null));
        Assert.assertEquals("N1", base.getName());
        Assert.assertEquals("R1", base.getResource());
        Assert.assertEquals(Integer.valueOf(100), base.getTimeout(200));
        Assert.assertTrue(base.getOriginal());
        Assert.assertTrue(base.hasEnvironment());
    }

    @Test
    public void testMergeTakesCurrentWhenSet() throws Exception {
        FunctionConfig cur = new FunctionConfig();
        Map<String, String> env = new HashMap<>();
        env.put("a", "1");
        cur.setEnvironment(env);
        cur.setResource("CURRENT_RES");
        cur.setOriginal(true);
        cur.setTimeout(10);
        cur.setName("CUR");
        FunctionConfig other = new FunctionConfig();
        Map<String, String> env2 = new HashMap<>();
        env2.put("b", "2");
        other.setEnvironment(env2);
        other.setResource("OTHER_RES");
        other.setOriginal(false);
        other.setTimeout(999);
        other.setName("OTHER");
        cur.merge(other);
        Assert.assertEquals("1", cur.getEnvironment().get("a"));
        Assert.assertEquals("2", cur.getEnvironment().get("b"));
        Assert.assertEquals("CURRENT_RES", cur.getResource());
        Assert.assertTrue(cur.getOriginal());
        Assert.assertEquals(Integer.valueOf(10), cur.getTimeout(1000));
        Assert.assertEquals("CUR", cur.getName());
    }

    @Test
    public void testMergeFillsMissingFromOther() throws Exception {
        FunctionConfig cur = new FunctionConfig();
        FunctionConfig other = new FunctionConfig();
        Map<String, String> env2 = new HashMap<>();
        env2.put("b", "2");
        other.setEnvironment(env2);
        other.setResource("OTHER_RES");
        other.setOriginal(true);
        other.setTimeout(999);
        other.setName("OTHER");
        cur.merge(other);
        Assert.assertEquals(env2, cur.getEnvironment());
        Assert.assertEquals("OTHER_RES", cur.getResource());
        Assert.assertTrue(cur.getOriginal());
        Assert.assertEquals(Integer.valueOf(999), cur.getTimeout(1000));
        Assert.assertEquals("OTHER", cur.getName());
    }

    @Test
    public void testGettersHasMethodsAndDefaults() throws Exception {
        FunctionConfig cfg = new FunctionConfig();
        Assert.assertFalse(cfg.getOriginal());
        Assert.assertFalse(cfg.hasEnvironment());
        Assert.assertFalse(cfg.hasResource());
        Assert.assertEquals(Integer.valueOf(5000), cfg.getTimeout(5000));
        cfg.setResource("");
        Assert.assertFalse(cfg.hasResource());
        cfg.setResource(" ");
        Assert.assertTrue(cfg.hasResource());
        cfg.setResource("RES");
        Assert.assertTrue(cfg.hasResource());
    }

    @Test
    public void testName() {
        FunctionConfig functionConfig = new FunctionConfig();
        Assert.assertEquals("OK", functionConfig.getName("OK"));
        functionConfig.setName("YES");
        Assert.assertEquals("YES", functionConfig.getName("OK"));
    }
}
