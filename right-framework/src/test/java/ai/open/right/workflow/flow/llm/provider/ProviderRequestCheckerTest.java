package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.Message;
import org.junit.Test;

public class ProviderRequestCheckerTest {

    @Test
    public void test() {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        req.setTokenBuffer(100);
        req.setTokenFirst(100);
        req.setToken("Token");
        req.setHistories(10);
        req.setStream(true);
        req.setContainHistories(true);
        ProviderRequestService.ProviderRequestChecker.check(req);
    }
}
