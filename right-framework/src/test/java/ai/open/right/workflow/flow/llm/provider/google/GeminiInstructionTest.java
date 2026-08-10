package ai.open.right.workflow.flow.llm.provider.google;

import org.junit.Assert;
import org.junit.Test;

public class GeminiInstructionTest {

    @Test
    public void test() throws Exception {
        GoogleRouter.GoogleMessage.GoogleInstruction inst = new GoogleRouter.GoogleMessage.GoogleInstruction("DiscordConfigTest");
        Assert.assertEquals("DiscordConfigTest", inst.getPart().getText());
    }
}
