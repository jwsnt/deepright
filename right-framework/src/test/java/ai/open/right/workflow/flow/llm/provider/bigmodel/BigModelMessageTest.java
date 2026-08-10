package ai.open.right.workflow.flow.llm.provider.bigmodel;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.store.history.History;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;
import java.util.Collections;

public class BigModelMessageTest {

    @Test
    public void test() throws Exception {
        OpenAiRequest req = new OpenAiRequest();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setQuery("MY QUERY");
        req.setResponseFormat(Collections.singletonMap("Hello", "World"));
        req.setFrequencyPenalty(2.0);
        req.setModel("deepseek-chat");
        req.setContainHistories(false);
        req.setTemperature(0.3D);
        req.setPrompt("Prompt");
        req.setMessage(message);
        req.setTokenBuffer(10);
        req.setTokenFirst(20);
        req.setHistories(10);
        req.setStream(false);
        History his = new History();
        his.setCreated(1L);
        his.setContent("Content");
        his.setRole(History.ROLE_ASSISTANT);
        his.setType(History.TYPE_ANSWER);
        message.addHistories(Arrays.asList(his));
        BigModelRouter.OpenAiMessage deepSeekMessage = new BigModelRouter.OpenAiMessage(req);
        Assert.assertEquals("World", deepSeekMessage.getResponseFormat().get("Hello"));
        Assert.assertEquals(deepSeekMessage.getFrequencyPenalty(), Double.valueOf(2.0D));
        Assert.assertEquals(deepSeekMessage.getTemperature(), Double.valueOf(0.3D));
        Assert.assertEquals(deepSeekMessage.getStream(), false);
        Assert.assertEquals(deepSeekMessage.getModel(), "deepseek-chat");
        Assert.assertEquals(deepSeekMessage.getMessages().size(), 3);
        // History -> Mime -> Query -> System
        Assert.assertEquals(deepSeekMessage.getMessages().get(0).getContent(), "Content");
        Assert.assertEquals(deepSeekMessage.getMessages().get(0).getRole(), "assistant");
        Assert.assertEquals(deepSeekMessage.getMessages().get(1).getContent(), "Prompt");
        Assert.assertEquals(deepSeekMessage.getMessages().get(1).getRole(), "system");
        Assert.assertEquals(deepSeekMessage.getMessages().get(2).getContent(), "MY QUERY");
        Assert.assertEquals(deepSeekMessage.getMessages().get(2).getRole(), "user");
    }
}
