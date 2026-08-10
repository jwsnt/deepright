package ai.open.right.workflow.flow.llm.provider.qwen;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRouter;
import ai.open.right.workflow.flow.llm.store.history.History;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;
import java.util.Collections;

public class QwenMessageTest {

    @Test
    public void test() throws Exception {
        OpenAiRequest req = new OpenAiRequest();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setQuery("MY QUERY");
        req.setResponseFormat(Collections.singletonMap("Hello", "World"));
        req.setFrequencyPenalty(2.0);
        req.setContainHistories(false);
        req.setModel("qwen-plus");
        req.setTemperature(0.3D);
        req.setPrompt("Prompt");
        req.setMessage(message);
        req.setTokenBuffer(10);
        req.setTokenFirst(20);
        req.setHistories(10);
        req.setStream(false);
        History his = new History();
        his.setCreated(0L);
        his.setContent("Content");
        his.setRole(History.ROLE_ASSISTANT);
        his.setType(History.TYPE_ANSWER);
        message.addHistories(Arrays.asList(his));
        OpenAiRouter.OpenAiMessage qwenMessage = new OpenAiRouter.OpenAiMessage(req);
        Assert.assertEquals("World", qwenMessage.getResponseFormat().get("Hello"));
        Assert.assertEquals(qwenMessage.getFrequencyPenalty(), Double.valueOf(2.0D));
        Assert.assertEquals(qwenMessage.getTemperature(), Double.valueOf(0.3D));
        Assert.assertEquals(qwenMessage.getStream(), false);
        Assert.assertEquals(qwenMessage.getModel(), "qwen-plus");
        Assert.assertEquals(qwenMessage.getMessages().size(), 3);
        // History -> Mime -> Query -> System
        Assert.assertEquals(qwenMessage.getMessages().get(0).getContent(), "Content");
        Assert.assertEquals(qwenMessage.getMessages().get(0).getRole(), "assistant");
        Assert.assertEquals(qwenMessage.getMessages().get(1).getContent(), "Prompt");
        Assert.assertEquals(qwenMessage.getMessages().get(1).getRole(), "system");
        Assert.assertEquals(qwenMessage.getMessages().get(2).getContent(), "MY QUERY");
        Assert.assertEquals(qwenMessage.getMessages().get(2).getRole(), "user");
    }
}
