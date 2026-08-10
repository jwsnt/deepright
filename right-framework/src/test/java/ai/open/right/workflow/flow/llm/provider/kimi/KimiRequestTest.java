package ai.open.right.workflow.flow.llm.provider.kimi;

import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import org.junit.Assert;
import org.junit.Test;

import java.util.Collections;

public class KimiRequestTest {

    @Test
    public void test() {
        OpenAiRequest req = new OpenAiRequest();
        req.setResponseFormat(Collections.singletonMap("Hello", "World"));
        req.setFrequencyPenalty(2.0);
        req.setModel("Model");
        req.setTemperature(0.3D);
        req.setPrompt("Prompt");
        req.setTopP(2.0);
        Assert.assertEquals("World", req.getResponseFormat().get("Hello"));
        Assert.assertEquals("Model", req.getModel());
        Assert.assertEquals("Prompt", req.getPrompt());
        Assert.assertEquals(req.getTopP(), Double.valueOf(2.0D));
        Assert.assertEquals(req.getTemperature(), Double.valueOf(0.3D));

    }
}
