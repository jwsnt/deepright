package ai.open.right.workflow.flow.mapcombine;

import org.junit.Assert;
import org.junit.Test;

public class CombineTest {
    @Test
    public void test() {
        Combine combine = new Combine();
        Assert.assertFalse(combine.hasNotifier());
        combine.setNotifier("NOTIFIER");
        Assert.assertTrue(combine.hasNotifier());
        Assert.assertEquals("NOTIFIER", combine.getNotifier());
    }

    @Test
    public void testInit() {
        Combine config = new Combine();
        Assert.assertEquals("NOTIFIER1", config.init("NOTIFIER1").getNotifier());
        Assert.assertEquals("NOTIFIER1", config.init("NOTIFIER2").getNotifier());
    }
}
