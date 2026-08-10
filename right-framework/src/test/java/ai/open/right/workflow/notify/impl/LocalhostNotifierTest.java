package ai.open.right.workflow.notify.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.RedirectContext;
import ai.open.right.listener.impl.EventListenerServiceImpl;
import ai.open.right.workflow.flow.WorkflowQueue;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.impl.WorkflowQueueImpl;
import ai.open.right.workflow.flow.llm.SegmentDelegate;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.NothingWriteBack;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;

public class LocalhostNotifierTest {

    @Test
    public void testWithConfig() throws Exception {
        LocalhostNotifier notifier = new LocalhostNotifier();
        notifier.setEventListenerService(new EventListenerServiceImpl());
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setFinished(true);
        notifier.setWorkflowQueue(new WorkflowQueueImpl() {
            @Override
            public void put(WorkflowTask workTask) {

            }
        });
        notifier.notify(segment, RedirectContext.EMPTY, new NothingWriteBack());
    }

    @Test
    public void testWithConfig2() throws Exception {
        LocalhostNotifier notifier = new LocalhostNotifier();
        notifier.setEventListenerService(new EventListenerServiceImpl());
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setFinished(true);
        notifier.setWorkflowQueue(new WorkflowQueueImpl() {
            @Override
            public void put(WorkflowTask workTask) {
                Assert.assertFalse(workTask.getMediaContext().isEmpty());
            }
        });
        notifier.notify(segment, RedirectContext.EMPTY, new NothingWriteBack(), Arrays.asList(new MediaContext()));
    }

    @Test
    public void testWithDef() throws Exception {
        LocalhostNotifier notifier = new LocalhostNotifier();
        notifier.setEventListenerService(new EventListenerServiceImpl());
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setFinished(true);
        notifier.setWorkflowQueue(new WorkflowQueueImpl() {
            @Override
            public void put(WorkflowTask workTask) {

            }
        });
        notifier.notify(segment, new NothingWriteBack());
    }

    @Test
    public void testWithDef2() throws Exception {
        LocalhostNotifier notifier = new LocalhostNotifier();
        notifier.setEventListenerService(new EventListenerServiceImpl());
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setFinished(true);
        notifier.setWorkflowQueue(new WorkflowQueueImpl() {
            @Override
            public void put(WorkflowTask workTask) {
                Assert.assertFalse(workTask.getMediaContext().isEmpty());
            }
        });
        notifier.notify(segment, new NothingWriteBack(), Arrays.asList(new MediaContext()));
    }

    @Test
    public void testInit() throws Exception {
        EventListenerServiceImpl eventListenerManager = EasyMock.createMock(EventListenerServiceImpl.class);
        WorkflowQueue workflowQueue = EasyMock.createMock(WorkflowQueue.class);
        EasyMock.replay(eventListenerManager, workflowQueue);
        LocalhostNotifier.InitConfig localhostNotifier = new LocalhostNotifier.InitConfig();
        localhostNotifier.setEventListenerService(eventListenerManager);
        localhostNotifier.setWorkflowQueue(workflowQueue);
        LocalhostNotifier empty = localhostNotifier.localhostNotifier();
        Assert.assertEquals(eventListenerManager, empty.getEventListenerService());
        Assert.assertEquals(workflowQueue, empty.getWorkflowQueue());
        EasyMock.verify(eventListenerManager, workflowQueue);
    }

    @Test
    public void testNotifyNotFinished() throws Exception {
        LocalhostNotifier notifier = new LocalhostNotifier();
        WorkflowQueue queue = EasyMock.createMock(WorkflowQueue.class);
        notifier.setWorkflowQueue(queue);

        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setFinished(false);

        EasyMock.replay(queue);
        notifier.notify(segment, null);
        EasyMock.verify(queue); // put should not be called
    }

    @Test(expected = IllegalArgumentException.class)
    public void testWorkflowTaskImplMetadataNull1() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setMetadata(null);
    }

    @Test
    public void testWorkflowTaskImplMetadataNull2() throws Exception {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertNull(task.getMetadata("KEY", String.class));
        Assert.assertFalse(task.containMetadata("KEY"));
    }

    @Test
    public void testWorkflowTaskImplIsFromFunCall() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertFalse(task.isFromFunCall());

        task.putMetadata(ProviderRequestService.KEY_FUN_FETCH, "TRUE");
        Assert.assertTrue(task.isFromFunCall());
    }

    @Test
    public void testWorkflowTaskImplChatTrack() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertFalse(task.containChatTrack());
        task.beginChatTrack();
        Assert.assertTrue(task.containChatTrack());
    }

    @Test
    public void testWorkflowTaskImplSetChat() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        task.setChat("MY_CHAT");
        Assert.assertEquals("MY_CHAT", task.getChat());
    }

    @Test
    public void testWorkflowTaskImplFunCallTrack() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertFalse(task.containFunCallTrack());
        task.beginFunCallTrack("TRACK_ID");
        Assert.assertTrue(task.containFunCallTrack());
        Assert.assertEquals("TRACK_ID", task.getFunCallTrack());
        task.closeFunCallTrack();
        Assert.assertFalse(task.containFunCallTrack());
    }

    @Test
    public void testWorkflowTaskImplIsClosed_delegatesToNotifierWriteBack() throws Exception {
        NothingWriteBack nwb = new NothingWriteBack();
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, nwb);
        Assert.assertFalse(task.isClosed());
        nwb.close();
        Assert.assertTrue(task.isClosed());
    }

    @Test
    public void testWorkflowTaskImplClose_delegatesToNotifierWriteBack() throws Exception {
        NothingWriteBack nwb = new NothingWriteBack();
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, nwb);
        Assert.assertFalse(nwb.isClosed());
        task.close();
        Assert.assertTrue(nwb.isClosed());
    }

    @Test
    public void testWorkflowTaskImplIncrDeepness() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Integer initial = task.getDeepness();
        task.incrDeepness();
        Assert.assertEquals(Integer.valueOf(initial + 1), task.getDeepness());
        task.incrDeepness();
        task.incrDeepness();
        Assert.assertEquals(Integer.valueOf(initial + 3), task.getDeepness());
    }
}
