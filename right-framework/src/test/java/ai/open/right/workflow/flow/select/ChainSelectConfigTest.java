package ai.open.right.workflow.flow.select;

import org.junit.Assert;
import org.junit.Test;

public class ChainSelectConfigTest {

    @Test
    public void test() {
        ChainSelectConfig chainSelectConfig = new ChainSelectConfig();
        chainSelectConfig.setChain("Chain");
        chainSelectConfig.setDynamic("Dynamic");
        Assert.assertEquals("Chain", chainSelectConfig.getChain());
        Assert.assertEquals("Dynamic", chainSelectConfig.getDynamic());
    }

    @Test
    public void testHashCode() throws Exception {
        Object object = ChainSelectConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testMerge() throws Exception {
        ChainSelectConfig config1 = new ChainSelectConfig();
        config1.setChain("chain1");
        config1.setDynamic("dynamic1");
        config1.setName("name1");
        ChainSelectConfig config2 = new ChainSelectConfig();
        config2.setChain("chain2");
        config2.setDynamic("dynamic2");
        config2.setName("name2");
        ChainSelectConfig merged = config1.merge(config2);
        Assert.assertEquals("chain1", merged.getChain());
        Assert.assertEquals("dynamic1", merged.getDynamic());
        Assert.assertEquals("name1", merged.getName());
        ChainSelectConfig config3 = new ChainSelectConfig();
        config3.setChain(null);
        config3.setDynamic(null);
        config3.setName(null);
        ChainSelectConfig merged2 = config3.merge(config2);
        Assert.assertEquals("chain2", merged2.getChain());
        Assert.assertEquals("dynamic2", merged2.getDynamic());
        Assert.assertEquals("name2", merged2.getName());
        ChainSelectConfig merged3 = config1.merge(null);
        Assert.assertEquals("chain1", merged3.getChain());
        Assert.assertEquals("dynamic1", merged3.getDynamic());
        Assert.assertEquals("name1", merged3.getName());
    }
}
