package ai.open.right.workflow.flow.llm.provider.coze;

import ai.open.right.workflow.flow.llm.store.history.History;
import org.junit.Assert;
import org.junit.Test;

public class CozeHistoryTest {

    @Test
    public void test() {
        History history = new History();
        history.setRole(History.ROLE_USER);
        history.setType(History.TYPE_ANSWER);
        history.setContent("HELLO");
        CozeRouter.CozeHistory cozeHistory = new CozeRouter.CozeHistory(history);
        Assert.assertEquals("text", cozeHistory.getContentType());
    }
}
