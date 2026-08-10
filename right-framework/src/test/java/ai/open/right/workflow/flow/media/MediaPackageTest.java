package ai.open.right.workflow.flow.media;

import org.junit.Assert;
import org.junit.Test;

public class MediaPackageTest {

    @Test
    public void testGetSet() {
        MediaPackage mediaPackage = MediaPackage.builder()
                .content("CONTENT")
                .source("SOURCE")
                .build();
        Assert.assertEquals("CONTENT", mediaPackage.getContent());
        Assert.assertEquals("SOURCE", mediaPackage.getSource());
    }
}
