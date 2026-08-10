package ai.open.right.workflow.flow.plan;

import org.junit.Assert;
import org.junit.Test;

public class PlanNotifierTest {

    @Test
    public void testPlan() {
        PlanNotifier notifier = new PlanNotifier();
        Assert.assertFalse(notifier.hasPlan());
        notifier.setPlan("NOTIFIER");
        Assert.assertTrue(notifier.hasPlan());
        Assert.assertEquals("NOTIFIER", notifier.getPlan());
    }

    @Test
    public void testSummary() {
        PlanNotifier notifier = new PlanNotifier();
        Assert.assertFalse(notifier.hasSummary());
        notifier.setSummary("NOTIFIER");
        Assert.assertTrue(notifier.hasSummary());
        Assert.assertEquals("NOTIFIER", notifier.getSummary());
    }

    @Test
    public void testException() {
        PlanNotifier notifier = new PlanNotifier();
        Assert.assertFalse(notifier.hasException());
        notifier.setException("NOTIFIER");
        Assert.assertTrue(notifier.hasException());
        Assert.assertEquals("NOTIFIER", notifier.getException());
    }

    @Test
    public void testMerge() throws Exception {
        PlanNotifier target = new PlanNotifier();
        PlanNotifier source = new PlanNotifier();
        PlanNotifier result = target.merge(null);
        Assert.assertSame(target, result);
        Assert.assertNull(target.getException());
        Assert.assertNull(target.getSummary());
        Assert.assertNull(target.getPlan());
        source.setException("EXCEPTION");
        source.setSummary("SUMMARY");
        source.setPlan("PLAN");
        target = new PlanNotifier();
        target.merge(source);
        Assert.assertEquals("EXCEPTION", target.getException());
        Assert.assertEquals("SUMMARY", target.getSummary());
        Assert.assertEquals("PLAN", target.getPlan());
        target = new PlanNotifier();
        target.setException("TARGET_EXCEPTION");
        target.setSummary("TARGET_SUMMARY");
        target.merge(source);
        Assert.assertEquals("TARGET_EXCEPTION", target.getException());
        Assert.assertEquals("TARGET_SUMMARY", target.getSummary());
        Assert.assertEquals("PLAN", target.getPlan());
        source = new PlanNotifier();
        source.setException("NEW_EXCEPTION");
        target = new PlanNotifier();
        target.merge(source);
        Assert.assertEquals("NEW_EXCEPTION", target.getException());
        Assert.assertNull(target.getSummary());
        Assert.assertNull(target.getPlan());
    }
}
