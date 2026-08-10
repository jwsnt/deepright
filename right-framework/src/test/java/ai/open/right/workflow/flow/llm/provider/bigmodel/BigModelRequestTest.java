package ai.open.right.workflow.flow.llm.provider.bigmodel;

import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import org.junit.Assert;
import org.junit.Test;

import java.util.Collections;

public class BigModelRequestTest {

    @Test
    public void test() {
        OpenAiRequest req = new OpenAiRequest();
        req.setResponseFormat(Collections.singletonMap("Hello", "World"));
        req.setTemperature(0.3D);
        req.setMaxTokens(1024);
        req.setTopP(0.8);
        req.setFrequencyPenalty(1.0);
        req.setPrompt("Prompt");
        Assert.assertEquals("Prompt", req.getPrompt());
        Assert.assertEquals("World", req.getResponseFormat().get("Hello"));
        Assert.assertEquals(req.getTemperature(), Double.valueOf(0.3D));
        Assert.assertEquals(req.getMaxTokens(), Integer.valueOf(1024));
        Assert.assertEquals(req.getTopP(), Double.valueOf(0.8D));
        Assert.assertEquals(req.getFrequencyPenalty(), Double.valueOf(1.0D));
    }
}
