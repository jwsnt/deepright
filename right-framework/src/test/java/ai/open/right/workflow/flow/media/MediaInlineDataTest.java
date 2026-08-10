package ai.open.right.workflow.flow.media;

import org.junit.Assert;
import org.junit.Test;

public class MediaInlineDataTest {

    @Test
    public void test() {
        MediaInlineData mediaInlineData = MediaInlineData.builder()
                .data("D")
                .mediaType("M")
                .build();
        Assert.assertEquals("M", mediaInlineData.getMediaType());
        Assert.assertEquals("D", mediaInlineData.getData());
    }
}
