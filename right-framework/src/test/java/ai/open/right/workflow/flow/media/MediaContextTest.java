package ai.open.right.workflow.flow.media;

import org.junit.Assert;
import org.junit.Test;

public class MediaContextTest {

    @Test
    public void testCanEncodeBase64() {
        MediaContext mediaContext = new MediaContext();
        mediaContext.setType(MediaContext.TEXT);
        Assert.assertFalse(mediaContext.canEncodeBase64());
        mediaContext.setType(MediaContext.PREFIX_INLINE + "image");
        Assert.assertFalse(mediaContext.canEncodeBase64());
        mediaContext.setType("image");
        Assert.assertTrue(mediaContext.canEncodeBase64());
    }

    @Test
    public void testCanEncodeBase64WithOutType() {
        MediaContext mediaContext = new MediaContext();
        Assert.assertFalse(mediaContext.canEncodeBase64());
    }

    @Test
    public void testGetType() {
        MediaContext mediaContext = new MediaContext();
        mediaContext.setType("image");
        Assert.assertEquals("image", mediaContext.getType());
        Assert.assertEquals("image", mediaContext.getType("png"));
        mediaContext.setType(null);
        Assert.assertEquals("png", mediaContext.getType("png"));
    }

    @Test
    public void testInline() {
        Assert.assertFalse(MediaContext.isInline("text"));
        Assert.assertTrue(MediaContext.isInline("inline:text"));
    }

    @Test
    public void testPureType() {
        Assert.assertEquals("text", MediaContext.pureType("inline:text"));
        Assert.assertEquals("text", MediaContext.pureType("text"));
        Assert.assertNull(MediaContext.pureType(null));
    }

    @Test
    public void testMime() {
        Assert.assertEquals("image/png", MediaContext.mimeType("test.png"));
        Assert.assertEquals("image/png", MediaContext.mimeType("/test.png"));
        Assert.assertEquals("image/png", MediaContext.mimeType("http://test.png"));
        Assert.assertEquals("image/png", MediaContext.mimeType("http://1.2.3/test.png"));
        Assert.assertEquals("image/png", MediaContext.mimeType("http://1.2.3/4.5.6/test.png"));
        Assert.assertNull(MediaContext.mimeType("http://1.2.3/4.5.6/"));
    }

    @Test
    public void testMimeException() {
        Assert.assertEquals("", MediaContext.mimeType(null));
    }
}
