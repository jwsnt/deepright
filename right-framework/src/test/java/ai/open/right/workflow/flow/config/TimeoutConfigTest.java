package ai.open.right.workflow.flow.config;

import org.junit.Assert;
import org.junit.Test;

public class TimeoutConfigTest {

    @Test
    public void test() {
        TimeoutConfig timeoutConfig = new TimeoutConfig();
        timeoutConfig.setTimeout4Condition(100);
        timeoutConfig.setTimeout4Corrector(200);
        timeoutConfig.setTimeout4Service(300);
        timeoutConfig.setTimeout4Llm(400);
        timeoutConfig.setTimeout(500);
        Assert.assertEquals(Integer.valueOf(100), timeoutConfig.getTimeout4Condition());
        Assert.assertEquals(Integer.valueOf(200), timeoutConfig.getTimeout4Corrector());
        Assert.assertEquals(Integer.valueOf(300), timeoutConfig.getTimeout4Service());
        Assert.assertEquals(Integer.valueOf(400), timeoutConfig.getTimeout4Llm());
        Assert.assertEquals(Integer.valueOf(500), timeoutConfig.getTimeout());
    }
}
