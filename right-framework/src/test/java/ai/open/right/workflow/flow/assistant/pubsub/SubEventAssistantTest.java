package ai.open.right.workflow.flow.assistant.pubsub;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.flow.pubsub.PubSubConfig;
import ai.open.right.workflow.flow.pubsub.PubSubService;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class SubEventAssistantTest {

    @Test
    public void test() throws Exception {
        PubSubService subService = EasyMock.createMock(PubSubService.class);
        WorkflowConfig _workflowConfig = new WorkflowConfig();
        PubSubConfig pubSubConfig = new PubSubConfig();
        _workflowConfig.setPubSubConfig(pubSubConfig);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(subService.sub(pubSubConfig, workflowTask)).andReturn("HELLO").anyTimes();
        SubEventAssistant subEventAssistant = new SubEventAssistant() {
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
                Assert.assertEquals(workflowTask, workTask);
                Assert.assertEquals("HELLO", content);
                Assert.assertEquals(_workflowConfig, workflowConfig);
            }
        };
        EasyMock.replay(subService);
        subEventAssistant.setPubSubService(subService);
        subEventAssistant.execute(_workflowConfig, workflowTask);
        EasyMock.verify(subService);
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);
        Map<String, LLMQueryService> llmQueryServices = new HashMap<>();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        MediaConfig mediaConfig = new MediaConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        PubSubService pubSubService = EasyMock.createMock(PubSubService.class);
        EasyMock.replay(pubSubService, notifierManager, signalFactory);
        SubEventAssistant.InitConfig assistant = new SubEventAssistant.InitConfig();
        assistant.setNotifierService(notifierManager);
        assistant.setLlmQueryService(llmQueryServices);
        assistant.setSignalFactory(signalFactory);
        assistant.setPubSubService(pubSubService);
        SubEventAssistant empty = assistant.subEventAssistant();
        Assert.assertEquals(pubSubService, empty.getPubSubService());
        Assert.assertEquals(empty.getLlmQueryService(), llmQueryServices);
        Assert.assertEquals(empty.getNotifierService(), notifierManager);
        Assert.assertEquals(empty.getSignalFactory(), signalFactory);
        EasyMock.verify(pubSubService, notifierManager, signalFactory);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = SubEventAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = SubEventAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
