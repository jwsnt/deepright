package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.workflow.flow.llm.config.LLMFunCall;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;

public class GoogleToolsTest {

    @Test
    public void testTools() throws Exception {
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setName("NAME");
        llmFunCall.setDescription("DESC");
        GoogleRouter.GoogleMessage.GoogleTools tools = new GoogleRouter.GoogleMessage.GoogleTools(Arrays.asList(llmFunCall));
        Assert.assertFalse(tools.getFunctions().isEmpty());
    }
}
