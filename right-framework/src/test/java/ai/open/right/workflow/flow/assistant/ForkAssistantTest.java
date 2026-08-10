package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.fork.ForkConfig;
import ai.open.right.workflow.flow.fork.ForkService;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class ForkAssistantTest {
    @Test
    public void test() throws Exception {
        WorkflowConfig _workflowConfig = new WorkflowConfig();
        _workflowConfig.setForkConfig(new ForkConfig());
        WorkflowTask _workTask = ObjectBuilder.buildWorkflowTask();
        ForkService forkService = EasyMock.createMock(ForkService.class);
        forkService.fork(_workflowConfig, _workTask);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(forkService);
        ForkAssistant forkAssistant = new ForkAssistant();
        forkAssistant.setForkService(forkService);
        forkAssistant.execute(_workflowConfig, _workTask);
        EasyMock.verify(forkService);
    }

    @Test
    public void testWithChainAndWarn() throws Exception {
        WorkflowConfig _workflowConfig = new WorkflowConfig();
        _workflowConfig.setForkConfig(new ForkConfig());
        _workflowConfig.setChain("CHAIN");
        WorkflowTask _workTask = ObjectBuilder.buildWorkflowTask();
        ForkService forkService = EasyMock.createMock(ForkService.class);
        forkService.fork(_workflowConfig, _workTask);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(forkService);
        ForkAssistant forkAssistant = new ForkAssistant();
        forkAssistant.setForkService(forkService);
        forkAssistant.execute(_workflowConfig, _workTask);
        EasyMock.verify(forkService);
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);
        Map<String, LLMQueryService> llmQueryServices = new HashMap<>();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        MediaConfig mediaConfig = new MediaConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        ForkService service = EasyMock.createMock(ForkService.class);
        EasyMock.replay(service, notifierManager, signalFactory);
        ForkAssistant.InitConfig assistant = new ForkAssistant.InitConfig();
        assistant.setNotifierService(notifierManager);
        assistant.setLlmQueryService(llmQueryServices);
        assistant.setSignalFactory(signalFactory);
        assistant.setForkService(service);
        ForkAssistant empty = assistant.forkAssistant();
        Assert.assertEquals(service, empty.getForkService());
        Assert.assertEquals(empty.getLlmQueryService(), llmQueryServices);
        Assert.assertEquals(empty.getNotifierService(), notifierManager);
        Assert.assertEquals(empty.getSignalFactory(), signalFactory);
        EasyMock.verify(service, notifierManager, signalFactory);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = ForkAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = ForkAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}

