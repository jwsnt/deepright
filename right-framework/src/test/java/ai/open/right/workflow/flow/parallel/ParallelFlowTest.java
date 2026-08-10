package ai.open.right.workflow.flow.parallel;

import org.junit.Assert;
import org.junit.Test;

public class ParallelFlowTest {

    @Test
    public void testGetSet() {
        ParallelFlow flow = new ParallelFlow();
        flow.setStopOnFailed(false);
        flow.setDynamic("WR");
        Assert.assertEquals("WR", flow.getDynamic());
        Assert.assertFalse(flow.getStopOnFailed());
    }
}
