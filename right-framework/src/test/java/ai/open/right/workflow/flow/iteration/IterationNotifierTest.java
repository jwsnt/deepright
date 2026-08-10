package ai.open.right.workflow.flow.iteration;

import org.junit.Assert;
import org.junit.Test;

public class IterationNotifierTest {

    @Test
    public void test() {
        IterationNotifier iterationNotifier = new IterationNotifier();
        Assert.assertFalse(iterationNotifier.hasProcessor());
        Assert.assertFalse(iterationNotifier.hasRefection());
        iterationNotifier.setProcessor("PROCESSOR");
        iterationNotifier.setRefection("REFECTION");
        Assert.assertEquals("PROCESSOR", iterationNotifier.getProcessor());
        Assert.assertEquals("REFECTION", iterationNotifier.getRefection());
    }

    @Test
    public void testMerge() throws Exception {
        IterationNotifier notifier1 = new IterationNotifier();
        notifier1.setRefection("Localhost");
        notifier1.setProcessor("Endpoint");
        IterationNotifier result1 = notifier1.merge(null);
        Assert.assertEquals("Localhost", result1.getRefection());
        Assert.assertEquals("Endpoint", result1.getProcessor());
        IterationNotifier notifier2 = new IterationNotifier();
        IterationNotifier target2 = new IterationNotifier();
        target2.setRefection("Source");
        target2.setProcessor("Localhost");
        IterationNotifier result2 = notifier2.merge(target2);
        Assert.assertEquals("Source", result2.getRefection());
        Assert.assertEquals("Localhost", result2.getProcessor());
        IterationNotifier notifier3 = new IterationNotifier();
        notifier3.setRefection("Endpoint");
        notifier3.setProcessor("Source");
        IterationNotifier target3 = new IterationNotifier();
        IterationNotifier result3 = notifier3.merge(target3);
        Assert.assertEquals("Endpoint", result3.getRefection());
        Assert.assertEquals("Source", result3.getProcessor());
        IterationNotifier notifier4 = new IterationNotifier();
        notifier4.setRefection("Localhost");
        IterationNotifier target4 = new IterationNotifier();
        target4.setProcessor("Endpoint");
        IterationNotifier result4 = notifier4.merge(target4);
        Assert.assertEquals("Localhost", result4.getRefection());
        Assert.assertEquals("Endpoint", result4.getProcessor());
        IterationNotifier notifier5 = new IterationNotifier();
        notifier5.setRefection("Source");
        notifier5.setProcessor("Localhost");
        IterationNotifier target5 = new IterationNotifier();
        target5.setRefection("Endpoint");
        target5.setProcessor("Source");
        IterationNotifier result5 = notifier5.merge(target5);
        Assert.assertEquals("Source", result5.getRefection());
        Assert.assertEquals("Localhost", result5.getProcessor());
    }
}
