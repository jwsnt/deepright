package ai.open.right.workflow.flow.llm.provider.reason.impl;

import ai.open.right.workflow.flow.llm.provider.reason.ProviderReason;
import org.junit.Assert;
import org.junit.Test;

public class ProviderReasonImplTest {

    @Test
    public void test() throws Exception {
        ProviderReasonImpl providerReason = new ProviderReasonImpl();
        Assert.assertEquals("OK", providerReason.reason(null, "OK", false, 100));
    }

    @Test
    public void testInit() throws Exception {
        ProviderReasonImpl.InitConfig initConfig = new ProviderReasonImpl.InitConfig();
        ProviderReason providerReason = initConfig.providerReason();
        Assert.assertNotNull(providerReason);
    }
}
