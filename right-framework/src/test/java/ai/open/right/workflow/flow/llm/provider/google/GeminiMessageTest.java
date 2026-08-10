package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.ObjectBuilder;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.Message;
import com.google.common.collect.ImmutableMap;
import org.junit.Assert;
import org.junit.Test;

public class GeminiMessageTest {

    @Test
    public void test() throws Exception {
        GoogleRequest req = new GoogleRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        req.setThinkingConfig(ImmutableMap.of("think", "low"));
        req.setImageConfig(ImmutableMap.of("image", "low"));
        req.setFrequencyPenalty(1.0);
        req.setPresencePenalty(2.0);
        req.setMaxOutputTokens(10);
        req.setTemperature(0.3D);
        req.setPrompt("Prompt");
        req.setTopP(0.3D);
        req.setTopK(2);
        GoogleRouter.GoogleMessage message = new GoogleRouter.GoogleMessage(req);
        String actual = "{\"topK\":2,\"presencePenalty\":2.0,\"thinkingConfig\":{\"think\":\"low\"},\"temperature\":0.3,\"frequencyPenalty\":1.0,\"maxOutputTokens\":10,\"topP\":0.3,\"imageConfig\":{\"image\":\"low\"}}";
        Assert.assertEquals(JsonUtils.write(message.getConfigs()), actual);
        Assert.assertEquals(Integer.valueOf(message.getContents().size()), Integer.valueOf(1));
        Assert.assertEquals(message.getInstruction().getPart().getText(), "Prompt");

    }
}
