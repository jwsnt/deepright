package ai.open.right.workflow.notify;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.SegmentDelegate;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class NotifierManagerTest {

    @Test
    public void test1() throws Exception {
        NotifierServiceImpl manager = new NotifierServiceImpl();
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setNotifier(Notifier.LOCALHOST);
        Map<String, Notifier> notifier = new HashMap<String, Notifier>();
        manager.setNotifier(notifier);
        notifier.put(Notifier.LOCALHOST, new Notifier() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {

            }

            @Override
            public void notify(Segment segment1, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) {
                Assert.assertEquals(segment1, segment);
                Assert.assertEquals(redirectContext, RedirectContext.EMPTY);
            }

            @Override
            public void notify(Segment segment, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {

            }

            @Override
            public void notify(Segment segment, NotifierWriteBack notifierWriteBack) {

            }
        });
        manager.notify(segment, RedirectContext.EMPTY, null);
    }

    @Test
    public void test2() throws Exception {
        NotifierServiceImpl manager = new NotifierServiceImpl();
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setNotifier(Notifier.LOCALHOST);
        Map<String, Notifier> notifier = new HashMap<String, Notifier>();
        manager.setNotifier(notifier);
        notifier.put(Notifier.LOCALHOST, new Notifier() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {

            }

            @Override
            public void notify(Segment segment1, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) {
                Assert.assertEquals(segment1, segment);
                Assert.assertEquals(redirectContext, RedirectContext.EMPTY);
            }

            @Override
            public void notify(Segment segment, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {

            }

            @Override
            public void notify(Segment segment, NotifierWriteBack notifierWriteBack) {

            }
        });
        manager.notify(segment, null);
    }

    @Test
    public void test3() throws Exception {
        NotifierServiceImpl manager = new NotifierServiceImpl();
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setNotifier(Notifier.LOCALHOST);
        Map<String, Notifier> notifier = new HashMap<String, Notifier>();
        manager.setNotifier(notifier);
        List<MediaContext> mediaContexts = new ArrayList<>();
        mediaContexts.add(new MediaContext());
        notifier.put(Notifier.LOCALHOST, new Notifier() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assert.assertEquals(mediaContexts, mediaContext);
            }

            @Override
            public void notify(Segment segment1, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) {
            }

            @Override
            public void notify(Segment segment, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {

            }

            @Override
            public void notify(Segment segment, NotifierWriteBack notifierWriteBack) {

            }
        });
        manager.notify(Notifier.LOCALHOST, segment, RedirectContext.EMPTY, null, mediaContexts);
    }

    @Test
    public void test4() throws Exception {
        NotifierServiceImpl manager = new NotifierServiceImpl();
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setNotifier(Notifier.LOCALHOST);
        Map<String, Notifier> notifier = new HashMap<String, Notifier>();
        manager.setNotifier(notifier);
        List<MediaContext> mediaContexts = new ArrayList<>();
        mediaContexts.add(new MediaContext());
        notifier.put(Notifier.LOCALHOST, new Notifier() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
            }

            @Override
            public void notify(Segment segment1, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) {
            }

            @Override
            public void notify(Segment segment, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assert.assertEquals(mediaContexts, mediaContext);
            }

            @Override
            public void notify(Segment segment, NotifierWriteBack notifierWriteBack) {

            }
        });
        manager.notify(segment, null, mediaContexts);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testWithEmpty() throws Exception {
        NotifierServiceImpl manager = new NotifierServiceImpl();
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setNotifier(Notifier.LOCALHOST);
        Map<String, Notifier> notifier = new HashMap<String, Notifier>();
        manager.setNotifier(notifier);
        manager.notify(segment, RedirectContext.EMPTY, null);
    }

    @Test
    public void testNotifyEach() throws Exception {
        NotifierServiceImpl manager = new NotifierServiceImpl();
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setNotifier(Notifier.LOCALHOST);
        Map<String, Notifier> notifier = new HashMap<String, Notifier>();
        notifier.put(Notifier.LOCALHOST, new Notifier() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {

            }

            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
            }

            @Override
            public void notify(Segment segment, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {

            }

            @Override
            public void notify(Segment segment, NotifierWriteBack notifierWriteBack) throws Exception {

            }
        });
        manager.setNotifier(notifier);
        manager.notify(Notifier.LOCALHOST, segment, RedirectContext.EMPTY, null);
    }

    @Test
    public void testInit() throws Exception {
        Map<String, Notifier> notifier = new HashMap<>();
        NotifierServiceImpl.InitConfig notifierManager = new NotifierServiceImpl.InitConfig();
        notifierManager.setNotifier(notifier);
        NotifierServiceImpl empty = (NotifierServiceImpl) notifierManager.notifierService();
        Assert.assertEquals(notifier, empty.getNotifier());
    }
}
