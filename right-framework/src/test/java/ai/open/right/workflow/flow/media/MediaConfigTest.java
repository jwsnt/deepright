package ai.open.right.workflow.flow.media;

import org.junit.Assert;
import org.junit.Test;

public class MediaConfigTest {

    @Test
    public void testBase64() {
        MediaConfig mediaConfig = new MediaConfig();
        Assert.assertFalse(mediaConfig.getBase64());
        mediaConfig.setBase64(true);
        Assert.assertTrue(mediaConfig.getBase64());
    }

    @Test
    public void testGetSet() {
        MediaConfig mediaConfig = new MediaConfig();
        Assert.assertEquals(MediaConfig.SPLIT, mediaConfig.getSplit());
        Assert.assertEquals("HELLO", mediaConfig.getSplit("HELLO"));
        Assert.assertEquals("WORLD", mediaConfig.getDynamic("WORLD"));
        Assert.assertEquals(Integer.valueOf(10000), mediaConfig.getTimeout4Llm(10000));
        mediaConfig.setSplit("_");
        mediaConfig.setDynamic("DYNAMIC");
        mediaConfig.setTimeout4Llm(1000);
        Assert.assertEquals("_", mediaConfig.getSplit());
        Assert.assertEquals("DYNAMIC", mediaConfig.getDynamic());
        Assert.assertEquals(Integer.valueOf(1000), mediaConfig.getTimeout4Llm());
        Assert.assertEquals("_", mediaConfig.getSplit("HELLO"));
        Assert.assertEquals("DYNAMIC", mediaConfig.getDynamic("WORLD"));
        Assert.assertEquals(Integer.valueOf(1000), mediaConfig.getTimeout4Llm(10000));
    }

    @Test
    public void testMerge() throws Exception {
        MediaConfig config1 = new MediaConfig();
        MediaConfig config2 = new MediaConfig();
        MediaConfig merged = config1.merge(null);
        Assert.assertSame(config1, merged);
        merged = config1.merge(config2);
        Assert.assertEquals(";", merged.getSplit());
        Assert.assertNull(merged.getTimeout4Llm());
        Assert.assertNull(merged.getDynamic());
        Assert.assertEquals(MediaConfig.SPLIT, merged.getSplit());
        Assert.assertFalse(merged.getBase64());
        config1.setTimeout4Llm(100);
        config1.setBase64(true);
        config2.setTimeout4Llm(200);
        config2.setDynamic("test");
        config2.setSplit(",");
        config2.setBase64(false);
        merged = config1.merge(config2);
        Assert.assertEquals(Integer.valueOf(100), merged.getTimeout4Llm());
        Assert.assertEquals("test", merged.getDynamic());
        Assert.assertEquals(",", merged.getSplit());
        Assert.assertTrue(merged.getBase64());
        MediaConfig config3 = new MediaConfig();
        config3.setTimeout4Llm(300);
        merged = new MediaConfig().merge(config3);
        Assert.assertEquals(Integer.valueOf(300), merged.getTimeout4Llm());
    }
}
