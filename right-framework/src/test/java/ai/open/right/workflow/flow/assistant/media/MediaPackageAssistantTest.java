package ai.open.right.workflow.flow.assistant.media;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.flow.media.MediaPackageService;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.Map;

public class MediaPackageAssistantTest {

    @Test
    public void test() throws Exception {
        WorkflowConfig workflowConfig = new WorkflowConfig();
        MediaConfig mediaConfig = new MediaConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        MediaPackageService mediaPackageService = EasyMock.createMock(MediaPackageService.class);
        EasyMock.expect(mediaPackageService.pack(mediaConfig, workflowTask)).andReturn(new ArrayList<>()).anyTimes();
        EasyMock.replay(mediaPackageService);
        MediaPackageAssistant mediaPackageAssistant = new MediaPackageAssistant() {
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
            }
        };
        mediaPackageAssistant.setMediaPackageService(mediaPackageService);
        mediaPackageAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(mediaPackageService);
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);
        Map<String, LLMQueryService> llmQueryServices = new HashMap<>();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        MediaConfig mediaConfig = new MediaConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        MediaPackageService mediaPackageService = EasyMock.createMock(MediaPackageService.class);
        EasyMock.replay(mediaPackageService, notifierManager, signalFactory);
        MediaPackageAssistant.InitConfig mediaPackageAssistant = new MediaPackageAssistant.InitConfig();
        mediaPackageAssistant.setNotifierService(notifierManager);
        mediaPackageAssistant.setLlmQueryService(llmQueryServices);
        mediaPackageAssistant.setSignalFactory(signalFactory);
        mediaPackageAssistant.setMediaPackageService(mediaPackageService);
        MediaPackageAssistant empty = mediaPackageAssistant.mediaPackageAssistant();
        Assert.assertEquals(mediaPackageService, empty.getMediaPackageService());
        Assert.assertEquals(empty.getLlmQueryService(), llmQueryServices);
        Assert.assertEquals(empty.getNotifierService(), notifierManager);
        Assert.assertEquals(empty.getSignalFactory(), signalFactory);
        EasyMock.verify(mediaPackageService, notifierManager, signalFactory);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = MediaPackageAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = MediaPackageAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
