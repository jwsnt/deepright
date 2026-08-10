package ai.open.right.workflow.flow.assistant.replay;

import ai.open.right.ObjectBuilder;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.flow.track.TrackChatBody;
import ai.open.right.workflow.flow.track.TrackChatService;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class ChatReplayAssistantTest {

    @Test
    public void testExecute() throws Exception {
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        TrackChatService trackChatService = EasyMock.createMock(TrackChatService.class);
        List<TrackChatBody> bodies = new ArrayList<>();
        TrackChatBody trackChatBody = new TrackChatBody();
        trackChatBody.setBiz("BIZ");
        trackChatBody.setCode(500);
        trackChatBody.setContent("CONTENT");
        trackChatBody.setMetadata(new HashMap<>());
        trackChatBody.setConversation("CONVERSATION");
        trackChatBody.setWorkflow("WORKFLOW");
        bodies.add(trackChatBody);
        EasyMock.expect(trackChatService.restore(_workflowTask)).andReturn(bodies).anyTimes();
        EasyMock.replay(trackChatService);
        ChatReplayAssistant assistant = new ChatReplayAssistant() {

            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
                Assert.assertEquals(_workflowTask, workTask);
                Assert.assertEquals("{\"query\":\"UNKNOWN\",\"chat\":[{\"metadata\":{},\"conversation\":\"CONVERSATION\",\"workflow\":\"WORKFLOW\",\"content\":\"CONTENT\",\"code\":500,\"biz\":\"BIZ\"}]}", content);
            }
        };
        assistant.setTrackChatService(trackChatService);
        assistant.execute(new WorkflowConfig(), _workflowTask);
    }

    @Test
    public void testExecuteWithNoQuery() throws Exception {
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        _workflowTask.setQuery("");
        TrackChatService trackChatService = EasyMock.createMock(TrackChatService.class);
        List<TrackChatBody> bodies = new ArrayList<>();
        TrackChatBody trackChatBody = new TrackChatBody();
        trackChatBody.setBiz("BIZ");
        trackChatBody.setCode(500);
        trackChatBody.setContent("CONTENT");
        trackChatBody.setMetadata(new HashMap<>());
        trackChatBody.setConversation("CONVERSATION");
        trackChatBody.setWorkflow("WORKFLOW");
        bodies.add(trackChatBody);
        EasyMock.expect(trackChatService.restore(_workflowTask)).andReturn(bodies).anyTimes();
        EasyMock.replay(trackChatService);
        ChatReplayAssistant assistant = new ChatReplayAssistant() {

            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
                Assert.assertEquals(_workflowTask, workTask);
                Assert.assertEquals(JsonUtils.write(bodies), content);
            }
        };
        assistant.setTrackChatService(trackChatService);
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
        TrackChatService trackChatService = EasyMock.createMock(TrackChatService.class);
        EasyMock.replay(trackChatService, notifierManager, signalFactory);
        ChatReplayAssistant.InitConfig assistant = new ChatReplayAssistant.InitConfig();
        assistant.setNotifierService(notifierManager);
        assistant.setLlmQueryService(llmQueryServices);
        assistant.setSignalFactory(signalFactory);
        assistant.setTrackChatService(trackChatService);
        ChatReplayAssistant empty = assistant.chatReplayAssistant();
        Assert.assertEquals(trackChatService, empty.getTrackChatService());
        Assert.assertEquals(empty.getLlmQueryService(), llmQueryServices);
        Assert.assertEquals(empty.getNotifierService(), notifierManager);
        Assert.assertEquals(empty.getSignalFactory(), signalFactory);
        EasyMock.verify(trackChatService, notifierManager, signalFactory);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = ChatReplayAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = ChatReplayAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
