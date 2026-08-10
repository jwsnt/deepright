package ai.open.right.workflow.flow.llm.provider.coze;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.store.history.History;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;

public class CozeMessageTest {

    @Test
    public void test() {
        CozeRequest cozeRequest = new CozeRequest();
        cozeRequest.setBotId("BotID");
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        cozeRequest.setContainHistories(false);
        cozeRequest.setMessage(message);
        cozeRequest.setTokenBuffer(10);
        cozeRequest.setTokenFirst(20);
        cozeRequest.setHistories(10);
        cozeRequest.setStream(false);
        History history = new History();
        history.setContent("Content DiscordConfigTest");
        history.setRole(History.ROLE_ASSISTANT);
        history.setType(History.TYPE_ANSWER);
        message.addHistories(Arrays.asList(history));
        CozeRouter.CozeMessage cozeMessage = new CozeRouter.CozeMessage(cozeRequest);
        Assert.assertEquals(cozeMessage.getBotId(), "BotID");
        Assert.assertEquals(cozeMessage.getConversation(), "UNKNOWN");
        Assert.assertEquals(cozeMessage.getStream(), false);
        Assert.assertEquals(cozeMessage.getQuery(), "UNKNOWN");
        Assert.assertEquals(cozeMessage.getUser(), "UNKNOWN");
        Assert.assertEquals(cozeMessage.getHistories().size(), 1);
        Assert.assertEquals(cozeMessage.getHistories().get(0).getContent(), "Content DiscordConfigTest");
        Assert.assertEquals(cozeMessage.getHistories().get(0).getRole(), "assistant");
        Assert.assertEquals(cozeMessage.getHistories().get(0).getType(), "answer");
    }
}
