package ai.open.right.workflow.flow.parallel;

import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;

public class ParallelConfigTest {

    @Test
    public void testGetSet() {
        ParallelConfig config = new ParallelConfig();
        Assert.assertFalse(config.hasNotifier());
        config.setNotifier("NOTIFIER");
        config.setTimeout4Llm(1000);
        ParallelFlow flow = new ParallelFlow();
        config.setParallelFlow(Arrays.asList(flow));
        Assert.assertTrue(config.hasNotifier());
        Assert.assertEquals("NOTIFIER", config.getNotifier());
        Assert.assertEquals(1, config.getParallelFlow().size());
        Assert.assertEquals(Integer.valueOf(1000), config.getTimeout4Llm());
    }

    @Test
    public void testInit() {
        ParallelConfig config = new ParallelConfig();
        Assert.assertEquals("NOTIFIER1", config.init("NOTIFIER1").getNotifier());
        Assert.assertEquals("NOTIFIER1", config.init("NOTIFIER2").getNotifier());
    }

    @Test
    public void testMerge() throws Exception {
        ParallelConfig target = new ParallelConfig();
        ParallelConfig source = new ParallelConfig();
        source.setNotifier("SOURCE_NOTIFIER");
        source.setTimeout4Llm(2000);
        ParallelFlow sourceFlow = new ParallelFlow();
        source.setParallelFlow(Arrays.asList(sourceFlow));
        ParallelConfig merged = target.merge(source);
        Assert.assertEquals("SOURCE_NOTIFIER", merged.getNotifier());
        Assert.assertEquals(Integer.valueOf(2000), merged.getTimeout4Llm());
        Assert.assertEquals(1, merged.getParallelFlow().size());
        ParallelConfig target2 = new ParallelConfig();
        target2.setNotifier("TARGET_NOTIFIER");
        target2.setTimeout4Llm(1000);
        ParallelFlow targetFlow = new ParallelFlow();
        target2.setParallelFlow(Arrays.asList(targetFlow));
        ParallelConfig merged2 = target2.merge(source);
        Assert.assertEquals("TARGET_NOTIFIER", merged2.getNotifier());
        Assert.assertEquals(Integer.valueOf(1000), merged2.getTimeout4Llm());
        Assert.assertEquals(2, merged2.getParallelFlow().size());
        Assert.assertSame(targetFlow, merged2.getParallelFlow().get(0));
        ParallelConfig target3 = new ParallelConfig();
        target3.setNotifier("TARGET3_NOTIFIER");
        ParallelConfig merged3 = target3.merge(null);
        Assert.assertEquals("TARGET3_NOTIFIER", merged3.getNotifier());
        ParallelConfig target4 = new ParallelConfig();
        target4.setTimeout4Llm(3000);
        ParallelConfig source4 = new ParallelConfig();
        source4.setNotifier("SOURCE4_NOTIFIER");
        source4.setParallelFlow(Arrays.asList(new ParallelFlow()));
        ParallelConfig merged4 = target4.merge(source4);
        Assert.assertEquals("SOURCE4_NOTIFIER", merged4.getNotifier());
        Assert.assertEquals(Integer.valueOf(3000), merged4.getTimeout4Llm());
        Assert.assertEquals(1, merged4.getParallelFlow().size());
    }

    @Test
    public void testGetTimeout4Llm() {
        ParallelConfig config = new ParallelConfig();
        Assert.assertEquals(Integer.valueOf(5000), config.getTimeout4Llm(5000));
        config.setTimeout4Llm(3000);
        Assert.assertEquals(Integer.valueOf(3000), config.getTimeout4Llm(5000));
    }

    @Test
    public void testHasParallelFlow() {
        ParallelConfig config = new ParallelConfig();
        Assert.assertFalse(config.hasParallelFlow());
        config.setParallelFlow(Arrays.asList(new ParallelFlow()));
        Assert.assertTrue(config.hasParallelFlow());
        config.setParallelFlow(null);
        Assert.assertFalse(config.hasParallelFlow());
    }
}
