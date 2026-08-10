package ai.open.right.workflow.flow.llm.rag.meta;

import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;

/**
 * RagMetaConfig 单测：继承 AllowedConfig 的白名单/黑名单与 allowed 行为。
 */
public class RagMetaConfigTest {

    @Test
    public void testInstantiation() {
        RagMetaConfig config = new RagMetaConfig();
        Assert.assertNotNull(config);
    }

    @Test
    public void testAllowedWhenNoList() {
        RagMetaConfig config = new RagMetaConfig();
        Assert.assertTrue(config.allowed("any-name"));
    }

    @Test
    public void testAllowedWithWhiteListMatch() {
        RagMetaConfig config = new RagMetaConfig();
        config.setWhiteList(Arrays.asList("foo", "bar.*"));
        Assert.assertTrue(config.allowed("foo"));
        Assert.assertTrue(config.allowed("bar.xxx"));
    }

    @Test
    public void testAllowedWithWhiteListNoMatch() {
        RagMetaConfig config = new RagMetaConfig();
        config.setWhiteList(Arrays.asList("foo", "bar"));
        Assert.assertFalse(config.allowed("other"));
    }

    @Test
    public void testAllowedWithBlackListMatch() {
        RagMetaConfig config = new RagMetaConfig();
        config.setBlackList(Arrays.asList("exclude", "no.*"));
        Assert.assertFalse(config.allowed("exclude"));
        Assert.assertFalse(config.allowed("no.xxx"));
    }

    @Test
    public void testAllowedWithBlackListNoMatch() {
        RagMetaConfig config = new RagMetaConfig();
        config.setBlackList(Arrays.asList("exclude"));
        Assert.assertTrue(config.allowed("allowed-name"));
    }

    @Test
    public void testAddWhiteAndAddBlack() {
        RagMetaConfig config = new RagMetaConfig();
        config.addWhite("w1").addWhite("w2");
        config.addBlack("b1").addBlack("b2");
        Assert.assertEquals(Arrays.asList("w1", "w2"), config.getWhiteList());
        Assert.assertEquals(Arrays.asList("b1", "b2"), config.getBlackList());
    }

    @Test
    public void testWhiteListTakesPrecedenceOverBlackList() {
        RagMetaConfig config = new RagMetaConfig();
        config.setWhiteList(Arrays.asList("only"));
        config.setBlackList(Arrays.asList("only"));
        Assert.assertTrue(config.allowed("only"));
    }
}
