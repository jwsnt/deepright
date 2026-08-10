package ai.open.right.workflow.flow.llm.provider.kimi;

import org.junit.Assert;
import org.junit.Test;

public class KimiContentTest {

    @Test
    public void test() throws Exception {
        KimiRouter.OpenAiContent ct = new KimiRouter.OpenAiContent("DiscordConfigTest", "B",1L);
        Assert.assertEquals(ct.getContent(), "B");
        Assert.assertEquals(ct.getRole(), "DiscordConfigTest");
    }
}
