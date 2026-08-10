package ai.open.right.workflow.flow.assistant.pubsub;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.assistant.media.MediaTransferAssistant;
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

public class PubEventAssistantTest {

    @Test
    public void test() throws Exception {
        PubSubService subService = EasyMock.createMock(PubSubService.class);
        WorkflowConfig _workflowConfig = new WorkflowConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        PubSubConfig pubSubConfig = new PubSubConfig();
        _workflowConfig.setPubSubConfig(pubSubConfig);
        subService.pub(pubSubConfig, workflowTask);
        EasyMock.expectLastCall().anyTimes();
        PubEventAssistant pubEventAssistant = new PubEventAssistant() {
            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
                Assert.assertEquals(workflowTask, workTask);
                Assert.assertEquals(workflowTask.getQuery(), content);
                Assert.assertEquals(_workflowConfig, workflowConfig);
            }
        };
        EasyMock.replay(subService);
        pubEventAssistant.setPubSubService(subService);
        pubEventAssistant.execute(_workflowConfig, workflowTask);
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
        PubEventAssistant.InitConfig assistant = new PubEventAssistant.InitConfig();
        assistant.setNotifierService(notifierManager);
        assistant.setLlmQueryService(llmQueryServices);
        assistant.setSignalFactory(signalFactory);
        assistant.setPubSubService(pubSubService);
        PubEventAssistant empty = assistant.pubEventAssistant();
        Assert.assertEquals(pubSubService, empty.getPubSubService());
        Assert.assertEquals(empty.getLlmQueryService(), llmQueryServices);
        Assert.assertEquals(empty.getNotifierService(), notifierManager);
        Assert.assertEquals(empty.getSignalFactory(), signalFactory);
        EasyMock.verify(pubSubService, notifierManager, signalFactory);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = PubEventAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = PubEventAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
