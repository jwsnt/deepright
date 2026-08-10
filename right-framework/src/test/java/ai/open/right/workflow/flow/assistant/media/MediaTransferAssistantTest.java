package ai.open.right.workflow.flow.assistant.media;

import ai.open.right.ObjectBuilder;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.flow.media.MediaContent;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.flow.media.MediaTransferService;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import com.fasterxml.jackson.core.JsonParseException;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class MediaTransferAssistantTest {

    @Test(expected = JsonParseException.class)
    public void testWithJsonError() throws Exception {
        MediaTransferService mediaTransfer = EasyMock.createMock(MediaTransferService.class);
        List<MediaContext> mediaContext = new ArrayList<MediaContext>();
        MediaConfig mediaConfig = new MediaConfig();
        mediaTransfer.transfer(mediaConfig, ObjectBuilder.buildWorkflowTask(), mediaContext);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(mediaTransfer);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        MediaTransferAssistant mediaTransferAssistant = new MediaTransferAssistant();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        mediaTransferAssistant.setMediaTransferService(mediaTransfer);
        try {
            mediaTransferAssistant.execute(workflowConfig, workflowTask);
        } finally {
            EasyMock.verify(mediaTransfer);
        }
    }

    @Test(expected = IllegalArgumentException.class)
    public void testWithEmpty() throws Exception {
        MediaTransferService mediaTransfer = EasyMock.createMock(MediaTransferService.class);
        List<MediaContext> mediaContext = new ArrayList<MediaContext>();
        MediaConfig mediaConfig = new MediaConfig();
        mediaTransfer.transfer(mediaConfig, ObjectBuilder.buildWorkflowTask(), mediaContext);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(mediaTransfer);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        MediaTransferAssistant mediaTransferAssistant = new MediaTransferAssistant();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("{}");
        mediaTransferAssistant.setMediaTransferService(mediaTransfer);
        try {
            mediaTransferAssistant.execute(workflowConfig, workflowTask);
        } finally {
            EasyMock.verify(mediaTransfer);
        }
    }


    @Test
    public void testTransfer1() throws Exception {
        MediaTransferService mediaTransfer = EasyMock.createMock(MediaTransferService.class);
        List<MediaContext> mediaContext = new ArrayList<MediaContext>();
        mediaContext.add(new MediaContext());
        MediaConfig mediaConfig = new MediaConfig();
        mediaTransfer.transfer(mediaConfig, ObjectBuilder.buildWorkflowTask(), mediaContext);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(mediaTransfer);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        workflowConfig.setChain("Chain");
        MediaTransferAssistant mediaTransferAssistant = new MediaTransferAssistant() {
            @Override
            protected void transfer(WorkflowConfig workflowConfig, WorkflowTask workflowTask, MediaContent mediaContent) throws Exception {

            }
        };
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        MediaContent mediaContent = new MediaContent();
        mediaContent.setMediaContext(mediaContext);
        mediaContent.setQuery("HELLO WORLD");
        workflowTask.setQuery(JsonUtils.write(mediaContent));
        mediaTransferAssistant.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithMediaContext());
        mediaTransferAssistant.setMediaTransferService(mediaTransfer);
        mediaTransferAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(mediaTransfer);
    }

    /**
     * 覆盖 buildMetadata：当 mediaContent.hasMetadata() 为 true 时，将 metadata 写入 workTask。
     */
    @Test
    public void testBuildMetadata_CopiesMetadataToWorkTask() throws Exception {
        MediaTransferService mediaTransfer = EasyMock.createMock(MediaTransferService.class);
        EasyMock.replay(mediaTransfer);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setChain("Chain");
        MediaTransferAssistant mediaTransferAssistant = new MediaTransferAssistant() {
            @Override
            protected void transfer(WorkflowConfig workflowConfig, WorkflowTask workflowTask, MediaContent mediaContent) throws Exception {
            }
        };
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        MediaContent mediaContent = new MediaContent();
        mediaContent.setQuery("q1");
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("k1", "v1");
        metadata.put("k2", 2);
        mediaContent.setMetadata(metadata);
        mediaContent.setMediaContext(new ArrayList<>());
        workflowTask.setQuery(JsonUtils.write(mediaContent));
        mediaTransferAssistant.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithMediaContext());
        mediaTransferAssistant.setMediaTransferService(mediaTransfer);
        mediaTransferAssistant.execute(workflowConfig, workflowTask);
        Assert.assertNotNull(workflowTask.getMetadata());
        Assert.assertEquals("v1", workflowTask.getMetadata().get("k1"));
        Assert.assertEquals(2, workflowTask.getMetadata().get("k2"));
        EasyMock.verify(mediaTransfer);
    }

    @Test
    public void testTransfer2() throws Exception {
        MediaTransferService mediaTransfer = EasyMock.createMock(MediaTransferService.class);
        List<MediaContext> mediaContext = new ArrayList<MediaContext>();
        mediaContext.add(new MediaContext());
        MediaConfig mediaConfig = new MediaConfig();
        MediaContent mediaContent = new MediaContent();
        mediaContent.setMediaContext(mediaContext);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        mediaTransfer.transfer(mediaConfig, workflowTask, mediaContext);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(mediaTransfer);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        MediaTransferAssistant mediaTransferAssistant = new MediaTransferAssistant();
        mediaTransferAssistant.setMediaTransferService(mediaTransfer);
        mediaTransferAssistant.transfer(workflowConfig, workflowTask, mediaContent);
        EasyMock.verify(mediaTransfer);
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);
        Map<String, LLMQueryService> llmQueryServices = new HashMap<>();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        MediaConfig mediaConfig = new MediaConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        MediaTransferService mediaTransferService = EasyMock.createMock(MediaTransferService.class);
        EasyMock.replay(mediaTransferService, notifierManager, signalFactory);
        MediaTransferAssistant.InitConfig assistant = new MediaTransferAssistant.InitConfig();
        assistant.setNotifierService(notifierManager);
        assistant.setLlmQueryService(llmQueryServices);
        assistant.setSignalFactory(signalFactory);
        assistant.setMediaTransferService(mediaTransferService);
        MediaTransferAssistant empty = assistant.mediaTransferAssistant();
        Assert.assertEquals(mediaTransferService, empty.getMediaTransferService());
        Assert.assertEquals(empty.getLlmQueryService(), llmQueryServices);
        Assert.assertEquals(empty.getNotifierService(), notifierManager);
        Assert.assertEquals(empty.getSignalFactory(), signalFactory);
        EasyMock.verify(mediaTransferService, notifierManager, signalFactory);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = MediaTransferAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = MediaTransferAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
