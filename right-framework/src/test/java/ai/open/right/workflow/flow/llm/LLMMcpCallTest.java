package ai.open.right.workflow.flow.llm;

import ai.open.right.workflow.flow.llm.config.LLMMcpCall;
import org.junit.Assert;
import org.junit.Test;

public class LLMMcpCallTest {

    @Test
    public void testRewriter() {
        LLMMcpCall llmMcpCall = new LLMMcpCall();
        Assert.assertFalse(llmMcpCall.hasRewriter());
        llmMcpCall.setRewriter("Rewriter");
        Assert.assertTrue(llmMcpCall.hasRewriter());
    }
}
