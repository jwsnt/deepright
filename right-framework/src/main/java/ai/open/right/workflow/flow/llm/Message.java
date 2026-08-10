package ai.open.right.workflow.flow.llm;

import ai.open.right.context.RedirectContext;
import ai.open.right.context.UserContext;
import ai.open.right.workflow.flow.llm.store.history.History;
import org.springframework.util.Assert;

import java.util.List;
public interface Message extends LLMQuery, RedirectContext {

    public Boolean hasHistory();

    public List<History> getHistories();

    public void delHistories();

    public void addHistory(History history);

    public void addHistories(List<History> histories);

    public void replaceHistories(List<History> histories);

    public static Message build(LLMQuery llmQuery) {
        return new MessageDelegate(llmQuery);
    }

    public static class MessageChecker {

        public static void check(Message message) {
            Assert.hasText(message.getConversation(), "Conversation can not be empty");
            Assert.notNull(message.getUserContext(), "User Context can not be empty");
            Assert.notNull(message.getCreated(), "Timestamp can not be empty");
            Assert.hasText(message.getWorkflow(), "Workflow can not be empty");
            Assert.hasText(message.getNotifier(), "Notifier can not be empty");
            Assert.hasText(message.getQuery(), "Query can not be empty");
            Assert.hasText(message.getChat(), "Chat can not be empty");
            Assert.hasText(message.getBiz(), "Biz can not be empty");
            UserContext.UserContextChecker.check(message.getUserContext());
        }
    }
}
