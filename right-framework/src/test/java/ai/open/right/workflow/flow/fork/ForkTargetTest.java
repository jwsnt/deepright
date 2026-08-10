package ai.open.right.workflow.flow.fork;

import org.junit.Assert;
import org.junit.Test;

public class ForkTargetTest {
    @Test
    public void test() {
        ForkTarget forkTarget = new ForkTarget();
        Assert.assertFalse(forkTarget.hasCondition());
        forkTarget.setCondition("CONDITION");
        forkTarget.setDynamic("DYNAMIC");
        Assert.assertEquals("CONDITION", forkTarget.getCondition());
        Assert.assertEquals("DYNAMIC", forkTarget.getDynamic());
        Assert.assertTrue(forkTarget.hasCondition());
    }
}
