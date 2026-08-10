package ai.open.right.workflow.flow.llm.provider.google;

import org.junit.Assert;
import org.junit.Test;

public class GoogleMimeTest {

    @Test
    public void testGetSet() {
        GoogleRouter.GoogleMessage.GoogleMime googleMime = GoogleRouter.GoogleMessage.GoogleMime.builder()
                .mimeType("MIME")
                .data("DATA")
                .uri("URI")
                .build();
        Assert.assertEquals("MIME", googleMime.getMimeType());
        Assert.assertEquals("DATA", googleMime.getData());
        Assert.assertEquals("URI", googleMime.getUri());
    }
}
