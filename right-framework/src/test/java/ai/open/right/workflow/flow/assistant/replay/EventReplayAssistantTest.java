package ai.open.right.workflow.flow.assistant.replay;

import ai.open.right.ObjectBuilder;
import ai.open.right.listener.Event;
import ai.open.right.listener.EventImpl;
import ai.open.right.listener.EventReplay;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.*;

public class EventReplayAssistantTest {

    @Test
    public void testExecute() throws Exception {
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(10086L);
        EventReplay eventReplay = EasyMock.createMock(EventReplay.class);
        List<Event> bodies = new ArrayList<>();
        EventImpl event = new EventImpl();
        event.setData(10087L);
        event.setNow(10086L);
        event.setChat("Chat");
        event.setType("Type");
        event.setDevice("Device");
        event.setBiz("Biz");
        bodies.add(event);
        EasyMock.expect(eventReplay.replay(_workflowTask)).andReturn(bodies).anyTimes();
        EasyMock.replay(eventReplay);
        EventReplayAssistant assistant = new EventReplayAssistant() {

            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
                Assert.assertEquals(_workflowTask, workTask);
                Assert.assertEquals("{\"query\":\"UNKNOWN\",\"events\":[{\"device\":\"Device\",\"type\":\"Type\",\"data\":10087,\"chat\":\"Chat\",\"biz\":\"Biz\",\"now\":10086,\"dimension\":\"Biz-Chat-Device\"}]}", content);
            }
        };
        assistant.setEventReplay(eventReplay);
        assistant.execute(new WorkflowConfig(), _workflowTask);
    }

    @Test
    public void testExecuteWithNoQuery() throws Exception {
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        _workflowTask.setQuery("");
        EventReplay eventReplay = EasyMock.createMock(EventReplay.class);
        List<Event> bodies = new ArrayList<>();
        EventImpl event = new EventImpl();
        event.setData(new Date());
        event.setNow(10086L);
        event.setChat("Chat");
        event.setType("Type");
        event.setDevice("Device");
        event.setBiz("Biz");
        bodies.add(event);
        EasyMock.expect(eventReplay.replay(_workflowTask)).andReturn(bodies).anyTimes();
        EasyMock.replay(eventReplay);
        EventReplayAssistant assistant = new EventReplayAssistant() {

            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
                Assert.assertEquals(_workflowTask, workTask);
                Assert.assertEquals(JsonUtils.write(bodies), content);
            }
        };
        assistant.setEventReplay(eventReplay);
        assistant.execute(new WorkflowConfig(), _workflowTask);
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);
        Map<String, LLMQueryService> llmQueryServices = new HashMap<>();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        MediaConfig mediaConfig = new MediaConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        EventReplay eventReplay = EasyMock.createMock(EventReplay.class);
        EasyMock.replay(eventReplay, notifierManager, signalFactory);
        EventReplayAssistant.InitConfig assistant = new EventReplayAssistant.InitConfig();
        assistant.setNotifierService(notifierManager);
        assistant.setLlmQueryService(llmQueryServices);
        assistant.setSignalFactory(signalFactory);
        assistant.setEventReplay(eventReplay);
        EventReplayAssistant empty = assistant.eventReplayAssistant();
        Assert.assertEquals(eventReplay, empty.getEventReplay());
        Assert.assertEquals(empty.getLlmQueryService(), llmQueryServices);
        Assert.assertEquals(empty.getNotifierService(), notifierManager);
        Assert.assertEquals(empty.getSignalFactory(), signalFactory);
        EasyMock.verify(eventReplay, notifierManager, signalFactory);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = EventReplayAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = EventReplayAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
