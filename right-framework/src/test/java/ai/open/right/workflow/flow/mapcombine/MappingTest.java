package ai.open.right.workflow.flow.mapcombine;

import ai.open.right.workflow.flow.parallel.ParallelConfig;
import org.junit.Assert;
import org.junit.Test;

public class MappingTest {

    @Test
    public void test() {
        Mapping mapping = new Mapping();
        Assert.assertFalse(mapping.hasNotifier());
        mapping.setNotifier("NOTIFIER");
        Assert.assertTrue(mapping.hasNotifier());
        Assert.assertEquals("NOTIFIER", mapping.getNotifier());
    }

    @Test
    public void testInit() {
        Mapping config = new Mapping();
        Assert.assertEquals("NOTIFIER1", config.init("NOTIFIER1").getNotifier());
        Assert.assertEquals("NOTIFIER1", config.init("NOTIFIER2").getNotifier());
    }
}
