package ai.open.right.workflow.flow.script;

import org.junit.Assert;
import org.junit.Test;

public class ScriptUsageTest {

    @Test
    public void test() {
        ScriptUsage usage = new ScriptUsage();
        Assert.assertEquals(Integer.valueOf(0), usage.getCache());
        Assert.assertEquals(Integer.valueOf(0), usage.getTotal());
    }
}
