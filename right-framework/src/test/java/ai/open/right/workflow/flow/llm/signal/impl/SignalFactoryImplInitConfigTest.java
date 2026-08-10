package ai.open.right.workflow.flow.llm.signal.impl;

import org.junit.Assert;
import org.junit.Test;

public class SignalFactoryImplInitConfigTest {

    @Test
    public void shouldCreateSignalFactory() throws Exception {
        SignalFactoryImpl.InitConfig init = new SignalFactoryImpl.InitConfig();
        SignalFactoryImpl bean = (SignalFactoryImpl) init.signalFactory();
        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof SignalFactoryImpl);
    }
}
