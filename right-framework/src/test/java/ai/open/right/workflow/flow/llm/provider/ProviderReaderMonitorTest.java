package ai.open.right.workflow.flow.llm.provider;

import org.junit.Assert;
import org.junit.Test;
import org.junit.jupiter.api.Assertions;

public class ProviderReaderMonitorTest {

    @Test
    public void testMonitor() throws Exception {
        ProviderReaderMonitor router = new ProviderReaderMonitor();
        Assert.assertTrue(router.monitor().contains("The reader status="));
    }

    @org.junit.jupiter.api.Test
    public void testMonitorWithCounter() throws Exception {
        ProviderReaderMonitor router = new ProviderReaderMonitor();

        // 验证 RUNNER_COUNTER 为 0
        ProviderReaderCallback.RUNNER_COUNTER.set(0);
        Assertions.assertEquals("The reader status=0", router.monitor());

        // 验证 RUNNER_COUNTER 为 5
        ProviderReaderCallback.RUNNER_COUNTER.set(5);
        Assertions.assertEquals("The reader status=5", router.monitor());

        // 验证 RUNNER_COUNTER 为 -1
        ProviderReaderCallback.RUNNER_COUNTER.set(-1);
        Assertions.assertEquals("The reader status=-1", router.monitor());
    }
}

