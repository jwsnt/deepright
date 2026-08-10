package ai.open.right.workflow.flow.llm.provider.qwen;

import org.junit.Assert;
import org.junit.Test;

public class QwenContentTest {

    @Test
    public void test() throws Exception {
        QwenRouter.OpenAiContent ct = new QwenRouter.OpenAiContent("DiscordConfigTest", "B", 1L);
        Assert.assertEquals(ct.getContent(), "B");
        Assert.assertEquals(ct.getRole(), "DiscordConfigTest");
    }
}
