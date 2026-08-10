package ai.open.right.workflow.flow.llm.config;

import org.junit.Assert;
import org.junit.Test;

public class LLMTakeoverTest {

    @Test
    public void test() {
        LLMTakeover llmTakeover = new LLMTakeover();
        Assert.assertFalse(llmTakeover.hasNotifier());
        llmTakeover.setNotifier("A");
        Assert.assertTrue(llmTakeover.hasNotifier());
        Assert.assertEquals("A", llmTakeover.getNotifier());
    }

    @Test
    public void testMerge() throws Exception {
        LLMTakeover llmTakeover1 = new LLMTakeover();
        llmTakeover1.setNotifier("B");
        LLMTakeover llmTakeover2 = new LLMTakeover();
        llmTakeover2.setNotifier("C");
        llmTakeover2.merge(llmTakeover1);
        Assert.assertTrue(llmTakeover2.hasNotifier());
        Assert.assertEquals("C", llmTakeover2.getNotifier());
    }
}
