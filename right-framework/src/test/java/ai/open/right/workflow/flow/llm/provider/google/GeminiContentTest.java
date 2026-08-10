package ai.open.right.workflow.flow.llm.provider.google;

import org.junit.Assert;
import org.junit.Test;

public class GeminiContentTest {

    @Test
    public void test() throws Exception {
        GoogleRouter.GoogleMessage.GoogleContent content = new GoogleRouter.GoogleMessage.GoogleContent("DiscordConfigTest", "B", 1L);
        Assert.assertEquals("DiscordConfigTest", content.getParts().getFirst().getText());
        Assert.assertEquals("B", content.getRole());
    }
}
