package ai.open.right.workflow.flow.llm.provider.kimi;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.store.history.History;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;
import java.util.Collections;

public class KimiMessageTest {

    @Test
    public void test() throws Exception {
        OpenAiRequest req = new OpenAiRequest();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setQuery("MY QUERY");
        req.setResponseFormat(Collections.singletonMap("Hello", "World"));
        req.setFrequencyPenalty(2.0D);
        req.setContainHistories(false);
        req.setTemperature(0.3D);
        req.setPrompt("Prompt");
        req.setMessage(message);
        req.setTokenBuffer(10);
        req.setTokenFirst(20);
        req.setModel("moonshot-v1-8k");
        req.setHistories(10);
        req.setStream(false);
        History his = new History();
        his.setCreated(100L);
        his.setContent("Content");
        his.setRole(History.ROLE_ASSISTANT);
        his.setType(History.TYPE_ANSWER);
        message.addHistories(Arrays.asList(his));
        KimiRouter.OpenAiMessage cms = new KimiRouter.OpenAiMessage(req);
        Assert.assertEquals("World", cms.getResponseFormat().get("Hello"));
        Assert.assertEquals(cms.getFrequencyPenalty(), Double.valueOf(2.0D));
        Assert.assertEquals(cms.getTemperature(), Double.valueOf(0.3D));
        Assert.assertEquals(cms.getStream(), false);
        Assert.assertEquals(cms.getModel(), "moonshot-v1-8k");
        Assert.assertEquals(cms.getMessages().size(), 3);
        // History -> Mime -> Query -> System
        Assert.assertEquals(cms.getMessages().get(0).getContent(), "Content");
        Assert.assertEquals(cms.getMessages().get(0).getRole(), "assistant");
        Assert.assertEquals(cms.getMessages().get(1).getContent(), "Prompt");
        Assert.assertEquals(cms.getMessages().get(1).getRole(), "system");
        Assert.assertEquals(cms.getMessages().get(2).getContent(), "MY QUERY");
        Assert.assertEquals(cms.getMessages().get(2).getRole(), "user");
    }
}
