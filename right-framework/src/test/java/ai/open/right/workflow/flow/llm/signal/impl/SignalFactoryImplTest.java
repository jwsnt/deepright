package ai.open.right.workflow.flow.llm.signal.impl;

import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.signal.SignalConfig;
import ai.open.right.workflow.flow.llm.signal.SignalDistributor;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class SignalFactoryImplTest {
    @Test
    public void testWithNull() {
        WorkflowConfig config = new WorkflowConfig();
        SignalFactoryImpl factory = new SignalFactoryImpl();
        Assert.assertEquals(factory.signal(config), SignalStream.EMPTY);
    }

    @Test
    public void testWithEmpty() {
        WorkflowConfig config = new WorkflowConfig();
        config.setSignalConfig(new SignalConfig());
        SignalFactoryImpl factory = new SignalFactoryImpl();
        Assert.assertEquals(factory.signal(config), SignalStream.EMPTY);
    }

    @Test
    public void testValue() {
        WorkflowConfig config = new WorkflowConfig();
        SignalConfig signals = new SignalConfig();
        signals.getConfigs().put("KEY", "VAL");
        config.setSignalConfig(signals);
        SignalFactoryImpl factory = new SignalFactoryImpl();
        Assert.assertEquals(factory.signal(config).getClass(), SignalStreamImpl.class);
    }

    @Test
    public void testInit() throws Exception {
        SignalDistributor signalDistributor = EasyMock.createMock(SignalDistributor.class);
        SignalFactoryImpl.InitConfig service = new SignalFactoryImpl.InitConfig();
        service.setSignalDistributor(signalDistributor);
        SignalFactoryImpl empty = (SignalFactoryImpl) service.signalFactory();
        Assert.assertEquals(signalDistributor, empty.getSignalDistributor());
    }
}
