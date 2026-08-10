package ai.open.right.workflow.flow.llm.provider.google;

import org.junit.Assert;
import org.junit.Test;

public class GeminiPartTest {

    @Test
    public void test() throws Exception {
        GoogleRouter.GoogleMessage.GooglePart part = new GoogleRouter.GoogleMessage.GooglePart("DiscordConfigTest");
        Assert.assertEquals("DiscordConfigTest", part.getText());
    }
}
