package ai.open.right.workflow.flow.fork;

import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.List;

public class ForkConfigTest {

    @Test
    public void test() {
        List<ForkTarget> forkTarget = new ArrayList<ForkTarget>();
        ForkConfig forkConfig = new ForkConfig();
        Assert.assertFalse(forkConfig.getStopOnFailed());
        forkConfig.setStopOnFailed(true);
        forkConfig.setTarget(forkTarget);
        forkConfig.setTimeout(100);
        Assert.assertEquals(forkTarget, forkConfig.getTarget());
        Assert.assertEquals(Integer.valueOf(100), forkConfig.getTimeout());
        Assert.assertEquals(Integer.valueOf(100), forkConfig.getTimeout(1000));
        Assert.assertTrue(forkConfig.getStopOnFailed());
    }

    @Test
    public void testMergeWithNullArg_doesNothing() throws Exception {
        ForkConfig self = new ForkConfig();
        self.setStopOnFailed(true);
        self.setTimeout(123);
        List<ForkTarget> targets = new ArrayList<ForkTarget>();
        self.setTarget(targets);
        ForkConfig result = self.merge(null);
        Assert.assertSame(self, result);
        Assert.assertTrue(result.getStopOnFailed());
        Assert.assertEquals(Integer.valueOf(123), result.getTimeout());
        Assert.assertSame(targets, result.getTarget());
    }

    @Test
    public void testMergeCopiesValuesWhenSelfHasNulls() throws Exception {
        ForkConfig source = new ForkConfig();
        source.setStopOnFailed(true);
        source.setTimeout(456);
        List<ForkTarget> srcTargets = new ArrayList<ForkTarget>();
        ForkTarget t = new ForkTarget();
        t.setCondition("x");
        srcTargets.add(t);
        source.setTarget(srcTargets);
        ForkConfig self = new ForkConfig();
        ForkConfig result = self.merge(source);
        Assert.assertSame(self, result);
        Assert.assertTrue(result.getStopOnFailed());
        Assert.assertEquals(Integer.valueOf(456), result.getTimeout());
        Assert.assertSame(srcTargets, result.getTarget());
    }

    @Test
    public void testMergeDoesNotOverrideExistingValues() throws Exception {
        List<ForkTarget> selfTargets = new ArrayList<ForkTarget>();
        ForkConfig self = new ForkConfig();
        self.setStopOnFailed(false);
        self.setTimeout(111);
        self.setTarget(selfTargets);
        ForkConfig source = new ForkConfig();
        source.setStopOnFailed(true);
        source.setTimeout(999);
        List<ForkTarget> srcTargets = new ArrayList<ForkTarget>();
        source.setTarget(srcTargets);
        ForkConfig result = self.merge(source);
        Assert.assertSame(self, result);
        Assert.assertFalse(result.getStopOnFailed());
        Assert.assertEquals(Integer.valueOf(111), result.getTimeout());
        Assert.assertSame(selfTargets.size(), result.getTarget().size());
    }

    @Test
    public void testMergeWhenBothNulls_keepsDefaults() throws Exception {
        ForkConfig self = new ForkConfig();
        ForkConfig source = new ForkConfig();
        ForkConfig result = self.merge(source);
        Assert.assertSame(self, result);
        Assert.assertFalse(result.getStopOnFailed());
        Assert.assertEquals(Integer.valueOf(1000), result.getTimeout(1000));
        Assert.assertNull(result.getTarget());
    }
}
