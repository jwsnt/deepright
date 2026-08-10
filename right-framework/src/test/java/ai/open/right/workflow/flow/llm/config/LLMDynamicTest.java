package ai.open.right.workflow.flow.llm.config;

import org.junit.Assert;
import org.junit.Test;

public class LLMDynamicTest {

    @Test
    public void test() {
        LLMDynamic llmDynamic = new LLMDynamic();
        llmDynamic.setDynamic("Workflow");
        llmDynamic.setStopOnFailed(false);
        Assert.assertTrue(llmDynamic.toString().contains("stopOnFailed"));
        Assert.assertTrue(llmDynamic.toString().contains("dynamic"));
        Assert.assertTrue(llmDynamic.toString().contains("notifier"));
    }

    @Test
    public void testNotifier() {
        LLMDynamic llmDynamic = new LLMDynamic();
        Assert.assertNull(llmDynamic.getNotifier());
        llmDynamic.setNotifier("NOTIFIER");
        Assert.assertEquals("NOTIFIER", llmDynamic.getNotifier());
    }

    @Test
    public void testInit() {
        LLMDynamic llmDynamic = new LLMDynamic();
        Assert.assertNull(llmDynamic.getNotifier());
        llmDynamic.init("NOTIFIER");
        Assert.assertEquals("NOTIFIER", llmDynamic.getNotifier());
    }

    @Test
    public void testMerge() throws Exception {
        LLMDynamic target = new LLMDynamic();
        target.setStopOnFailed(true);
        target.setNotifier("target");
        target.setDynamic("targetDynamic");
        target.setTimeout(100);
        LLMDynamic source = new LLMDynamic();
        source.setStopOnFailed(false);
        source.setNotifier("source");
        source.setDynamic("sourceDynamic");
        source.setTimeout(200);
        target.merge(source);
        Assert.assertTrue(target.getStopOnFailed());
        Assert.assertEquals("target", target.getNotifier());
        Assert.assertEquals("targetDynamic", target.getDynamic());
        Assert.assertEquals(Integer.valueOf(100), target.getTimeout());
        LLMDynamic target2 = new LLMDynamic();
        target2.merge(source);
        Assert.assertFalse(target2.getStopOnFailed());
        Assert.assertEquals("source", target2.getNotifier());
        Assert.assertEquals("sourceDynamic", target2.getDynamic());
        Assert.assertEquals(Integer.valueOf(200), target2.getTimeout());
        LLMDynamic target3 = new LLMDynamic();
        target3.setStopOnFailed(null);
        target3.setNotifier(null);
        target3.setDynamic(null);
        target3.setTimeout(null);
        target3.merge(source);
        Assert.assertFalse(target3.getStopOnFailed());
        Assert.assertEquals("source", target3.getNotifier());
        Assert.assertEquals("sourceDynamic", target3.getDynamic());
        Assert.assertEquals(Integer.valueOf(200), target3.getTimeout());
        LLMDynamic target4 = new LLMDynamic();
        target4.merge(null);
        Assert.assertNull(target4.getNotifier());
        Assert.assertNull(target4.getDynamic());
        Assert.assertNull(target4.getTimeout());
    }

    @Test
    public void testGetTimeout() {
        LLMDynamic llmDynamic = new LLMDynamic();
        Assert.assertEquals(Integer.valueOf(500), llmDynamic.getTimeout(500));
        llmDynamic.setTimeout(1000);
        Assert.assertEquals(Integer.valueOf(1000), llmDynamic.getTimeout(500));
    }

    @Test
    public void testGetStopOnFailed() {
        LLMDynamic llmDynamic = new LLMDynamic();
        Assert.assertTrue(llmDynamic.getStopOnFailed());
        llmDynamic.setStopOnFailed(false);
        Assert.assertFalse(llmDynamic.getStopOnFailed());
        llmDynamic.setStopOnFailed(null);
        Assert.assertTrue(llmDynamic.getStopOnFailed());
    }

    @Test
    public void testHasNotifier() {
        LLMDynamic llmDynamic = new LLMDynamic();
        Assert.assertFalse(llmDynamic.hasNotifier());
        llmDynamic.setNotifier("");
        Assert.assertFalse(llmDynamic.hasNotifier());
        llmDynamic.setNotifier("notifier");
        Assert.assertTrue(llmDynamic.hasNotifier());
    }
}
