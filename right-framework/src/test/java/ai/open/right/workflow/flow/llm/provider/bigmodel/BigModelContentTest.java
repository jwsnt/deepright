package ai.open.right.workflow.flow.llm.provider.bigmodel;

import ai.open.right.workflow.flow.llm.provider.deepseek.DeepSeekRouter;
import org.junit.Assert;
import org.junit.Test;

public class BigModelContentTest {

    @Test
    public void test() throws Exception {
        DeepSeekRouter.OpenAiContent ct = new DeepSeekRouter.OpenAiContent("DiscordConfigTest", "B", 1L);
        Assert.assertEquals(ct.getContent(), "B");
        Assert.assertEquals(ct.getRole(), "DiscordConfigTest");
    }
}
