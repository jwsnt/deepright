package ai.open.right.workflow.sync;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.RedirectContext;
import ai.open.right.context.UserContext;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import ai.open.right.workflow.sync.impl.NotifierCallable;
import com.google.common.collect.ImmutableMap;
import org.apache.commons.collections.CollectionUtils;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.util.*;

public class SyncWorkflowTaskTest {

    private static WorkflowTask newTaskForConstructors() {
        WorkflowTask t = ObjectBuilder.buildWorkflowTask();
        t.setChat("WT_CHAT");
        t.setNotifier("WT_NOTIFIER");
        t.setTakeover("WT_TAKEOVER");
        t.putMetadata("META_K", "META_V");
        return t;
    }

    private static SyncCallable noopSyncCallable() {
        return new SyncCallable() {
            @Override
            public SyncCallable setNotifierWriteBack(NotifierWriteBack notifierWriteBack) {
                return this;
            }

            @Override
            public SyncCallable setNotifierService(NotifierService notifierService) {
                return this;
            }

            @Override
            public SyncCallable setRedirectContext(RedirectContext redirectContext) {
                return this;
            }

            @Override
            public void call(Segment segment) {
            }
        };
    }

    @Test
    public void isClosed_delegatesToWorkTask() throws Exception {
        ai.open.right.workflow.notify.NothingWriteBack nwb = new ai.open.right.workflow.notify.NothingWriteBack();
        ai.open.right.integration.RightTask rightTask = new ai.open.right.integration.RightTask(
                ai.open.right.integration.RightConfig.builder().query("Q").biz("B").trace("T").chat("C").timeout(10000)
                        .conversation("CO").upstream("U").notifier("N").protocol("P").metadata(new HashMap<>()).workflow("W").build().init(),
                nwb);
        rightTask.init();
        SyncWorkflowTask task = new SyncWorkflowTask(rightTask, null, 1000);
        Assertions.assertFalse(task.isClosed());
        nwb.close();
        Assertions.assertTrue(task.isClosed());
    }

    @Test
    public void close_delegatesToWorkTask() throws Exception {
        ai.open.right.workflow.notify.NothingWriteBack nwb = new ai.open.right.workflow.notify.NothingWriteBack();
        ai.open.right.integration.RightTask rightTask = new ai.open.right.integration.RightTask(
                ai.open.right.integration.RightConfig.builder().query("Q").biz("B").trace("T").chat("C").timeout(10000)
                        .conversation("CO").upstream("U").notifier("N").protocol("P").metadata(new HashMap<>()).workflow("W").build().init(),
                nwb);
        rightTask.init();
        SyncWorkflowTask task = new SyncWorkflowTask(rightTask, null, 1000);
        Assertions.assertFalse(nwb.isClosed());
        task.close();
        Assertions.assertTrue(nwb.isClosed());
    }

    @Test
    public void testSetGetObject() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQuery();
        SyncWorkflowTask delegate = new SyncWorkflowTask(workflowTask, null, 1000);
        LLMConfig config = new LLMConfig();
        config.setProvider("PROVIDER");
        delegate.setObjectQuery(config);
        config = delegate.getObjectQuery(LLMConfig.class);
        Assertions.assertNotNull(workflowTask.getConsuming());
        Assertions.assertEquals("PROVIDER", config.getProvider());
        Assertions.assertNotNull(delegate.getConsuming());
    }

    @Test
    public void testAddMediaContext() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQuery();
        SyncWorkflowTask delegate = new SyncWorkflowTask(workflowTask, null, 1000);
        MediaContext context = new MediaContext();
        delegate.addMediaContext(context);
        Assertions.assertEquals(context, delegate.getMediaContext().getFirst());
        Assertions.assertEquals(Integer.valueOf(delegate.getMediaContext().size()), Integer.valueOf(1));
    }

    @Test
    public void isFromFunMerge_usesSyncMetadata_notFetchAlone() {
        WorkflowTask inner = ObjectBuilder.buildWorkflowTask();
        SyncWorkflowTask sync = new SyncWorkflowTask(inner, null, 1000);
        Assertions.assertFalse(sync.isFromFunMerge());
        sync.putMetadata(ProviderRequestService.KEY_FUN_FETCH, new Object());
        Assertions.assertFalse(sync.isFromFunMerge());
        sync.putMetadata(ProviderRequestService.KEY_FUN_MERGE, new Object());
        Assertions.assertTrue(sync.isFromFunMerge());
    }

    @Test
    public void setDeepness_delegatesToWorkTask() {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        SyncWorkflowTask task = new SyncWorkflowTask(workflowTask, null, 1000);
        task.setDeepness(2);
        Assertions.assertEquals(Integer.valueOf(2), workflowTask.getDeepness());
        Assertions.assertEquals(Integer.valueOf(2), task.getDeepness());
    }

    @Test
    public void testFromFunCall1() {
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQuery();
        SyncWorkflowTask task = new SyncWorkflowTask(workflowTask, null, 1000);
        Assertions.assertFalse(task.isFromFunCall());
        task.putMetadata(ProviderRequestService.KEY_FUN_FETCH, new Object());
        Assertions.assertTrue(task.isFromFunCall());
    }

    @Test
    public void testFromFunCall2() {
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQuery();
        SyncWorkflowTask task = new SyncWorkflowTask(workflowTask, null, 1000);
        Assertions.assertFalse(task.isFromFunCall());
        task.putMetadata(ProviderRequestService.KEY_FUN_MERGE, new Object());
        Assertions.assertTrue(task.isFromFunCall());
    }

    @Test
    public void testIsEntryDelegatesToWorkTask() {
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQuery();
        SyncWorkflowTask syncWorkflowTask = new SyncWorkflowTask(workflowTask, null, 1000);
        Assertions.assertEquals(workflowTask.isEntry(), syncWorkflowTask.isEntry());
    }

    @Test
    public void testGetSet() {
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQuery();
        SyncWorkflowTask syncWorkflowTask = new SyncWorkflowTask(workflowTask, null, 1000);
        Assertions.assertEquals(syncWorkflowTask.getCreated(), workflowTask.getCreated());
        List<MediaContext> mediaContex = new ArrayList<>();
        syncWorkflowTask.setMediaContext(mediaContex);
        syncWorkflowTask.setProtocol("PR");
        syncWorkflowTask.setNotifier("NO");
        syncWorkflowTask.setQuery("QR");
        syncWorkflowTask.setChat("CHAT_VAL");
        syncWorkflowTask.setUpstream("UP");
        Assertions.assertEquals(workflowTask, syncWorkflowTask.getWorkTask());
        Assertions.assertEquals("CHAT_VAL", syncWorkflowTask.getChat());
        Assertions.assertEquals("UNKNOWN", workflowTask.getChat());
        Assertions.assertEquals(mediaContex, syncWorkflowTask.getMediaContext());
        Assertions.assertEquals("PR", syncWorkflowTask.getProtocol());
        Assertions.assertEquals("NO", syncWorkflowTask.getNotifier());
        Assertions.assertEquals("QR", syncWorkflowTask.getQuery());
        Assertions.assertEquals("UP", syncWorkflowTask.getUpstream());
        // 原始Query
        Assertions.assertEquals("ORIGINAL", syncWorkflowTask.getOriginal());
        Assertions.assertEquals("UNKNOWN", syncWorkflowTask.getTrace());
        Assertions.assertEquals("UP", syncWorkflowTask.getUpstream());
        Assertions.assertEquals("UNKNOWN", syncWorkflowTask.getDevice());
        Assertions.assertNotNull(syncWorkflowTask.getCreated());
        Assertions.assertEquals("UNKNOWN-UNKNOWN-UNKNOWN", syncWorkflowTask.getDimension());
        UserContext userContext = UserContext.builder().build();
        syncWorkflowTask.setUserContext(userContext);
        Assertions.assertEquals(userContext, syncWorkflowTask.getUserContext());
    }

    @Test
    public void testMetadata() throws Exception {
        SyncWorkflowTask syncWorkflowTask = new SyncWorkflowTask(ObjectBuilder.buildLLMQuery(), null, 1000);
        Assertions.assertFalse(syncWorkflowTask.containMetadata("HELLO"));
        syncWorkflowTask.putMetadata("HELLO", "WORLD");
        Assertions.assertTrue(syncWorkflowTask.containMetadata("HELLO"));
        Assertions.assertEquals("WORLD", syncWorkflowTask.getMetadata("HELLO", String.class));
        syncWorkflowTask.delMetadata("HELLO");
        Assertions.assertNull(syncWorkflowTask.getMetadata("HELLO", String.class));
    }

    @Test
    public void tesDelMetadata() throws Exception {
        SyncWorkflowTask syncWorkflowTask = new SyncWorkflowTask(ObjectBuilder.buildLLMQuery(), null, 1000);
        syncWorkflowTask.putMetadata("HELLO", "WORLD");
        Assertions.assertEquals("WORLD", syncWorkflowTask.delMetadata("HELLO", String.class));
    }

    @Test
    public void testSetProviderAndToken() {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        SyncWorkflowTask syncWorkflowTask = new SyncWorkflowTask(workflowTask, null, 1000);

        syncWorkflowTask.setProviderAndToken("provider-x", "token-y");

        Assertions.assertEquals("provider-x", syncWorkflowTask.getMetadata().get(ProviderRequestService.KEY_PROVIDER));
        Assertions.assertEquals("token-y",
                syncWorkflowTask.getMetadata().get(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN));
        Assertions.assertNull(workflowTask.getMetadata().get(ProviderRequestService.KEY_PROVIDER));
        Assertions.assertNull(
                workflowTask.getMetadata().get(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN));
    }

    @Test
    public void testExeWorkflowWithSyncNotifier() throws Exception {
        SyncWorkflowTask.exeWorkflow(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect(), null, ObjectBuilder.buildWorkflowTask(), "BIZ", "WORLD", new HashMap<>(), new ArrayList<>(), "query", null, null, 1000, 1000, "CHAT", true);
    }

    @Test
    public void testExeWorkflowWithSyncNotifierUsesExplicitBiz() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        SyncWorkflowTask.exeWorkflow(new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assertions.assertEquals("biz-explicit", segment.getBiz());
                Assertions.assertEquals("workflow-explicit", segment.getWorkflow());
                Assertions.assertEquals("query-explicit", segment.getContent());
                Assertions.assertSame(notifierWriteBack, redirectContext);
                Assertions.assertEquals("biz-explicit", SyncWorkflowTask.class.cast(notifierWriteBack).getBiz());
                Assertions.assertEquals("workflow-explicit", SyncWorkflowTask.class.cast(notifierWriteBack).getWorkflow());
            }
        }, null, workflowTask, "biz-explicit", "workflow-explicit", new HashMap<>(), new ArrayList<>(), "query-explicit", null, null, 1000, 1000, "CHAT", true);
    }

    @Test
    public void testExeWorkflowWithSyncNotifierSceneOverridesExplicitBiz() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        SyncWorkflowTask.exeWorkflow(new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assertions.assertEquals("biz-scene", segment.getBiz());
                Assertions.assertEquals("workflow-scene", segment.getWorkflow());
                Assertions.assertEquals("query-scene", segment.getContent());
                Assertions.assertEquals("biz-scene", SyncWorkflowTask.class.cast(notifierWriteBack).getBiz());
                Assertions.assertEquals("workflow-scene", SyncWorkflowTask.class.cast(notifierWriteBack).getWorkflow());
            }
        }, null, workflowTask, "biz-fallback", "biz-scene@workflow-scene", new HashMap<>(), new ArrayList<>(), "query-scene", null, null, 1000, 1000, "CHAT", true);
    }

    @Test
    public void testGetSetHistory() {
        SyncWorkflowTask syncWorkflowTask = new SyncWorkflowTask(ObjectBuilder.buildLLMQuery(), null, 1000);
        Assertions.assertFalse(syncWorkflowTask.containHistories());
        Assertions.assertTrue(CollectionUtils.isEmpty(syncWorkflowTask.getHistories()));
        List<History> histories = new ArrayList<History>();
        histories.add(new History());
        syncWorkflowTask.addHistories(histories);
        Assertions.assertTrue(syncWorkflowTask.containHistories());
        Assertions.assertNotNull(syncWorkflowTask.getHistories());
    }

    @Test
    public void testConfigPure1() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.putMetadata("YES", "NO");
        SyncConfig syncConfig = SyncConfig.builder()
                .workTask(workflowTask)
                .metadata(Collections.singletonMap("HELLO", "WORLD"))
                .pure(true)
                .build();
        SyncWorkflowTask.exeWorkflow(new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assertions.assertEquals(Integer.valueOf(1), Integer.valueOf(segment.getMetadata().size()));
                Assertions.assertEquals("WORLD", segment.getMetadata().get("HELLO"));
                Assertions.assertNull(segment.getMetadata().get("YES"));
            }
        }, syncConfig);
    }

    @Test
    public void testConfigPure2() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.putMetadata("YES", "NO");
        SyncConfig syncConfig = SyncConfig.builder()
                .workTask(workflowTask)
                .pure(true)
                .build();
        SyncWorkflowTask.exeWorkflow(new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assertions.assertEquals(Integer.valueOf(0), Integer.valueOf(segment.getMetadata().size()));
            }
        }, syncConfig);
    }

    @Test
    public void testConfigMetadata1() throws Exception {
        SyncConfig syncConfig = SyncConfig.builder()
                .workTask(ObjectBuilder.buildWorkflowTask())
                .metadata(Collections.singletonMap("HELLO", "WORLD"))
                .build();
        SyncWorkflowTask.exeWorkflow(new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assertions.assertEquals("WORLD", segment.getMetadata().get("HELLO"));
            }
        }, syncConfig);
    }

    @Test
    public void testConfigMetadata2() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.putMetadata("YES", "NO");
        SyncConfig syncConfig = SyncConfig.builder()
                .workTask(workflowTask)
                .metadata(Collections.singletonMap("HELLO", "WORLD"))
                .build();
        SyncWorkflowTask.exeWorkflow(new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assertions.assertEquals("WORLD", segment.getMetadata().get("HELLO"));
                Assertions.assertEquals("NO", segment.getMetadata().get("YES"));
            }
        }, syncConfig);
    }

    @Test
    public void testConfigMetadata3() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.putMetadata("YES", "NO");
        SyncConfig syncConfig = SyncConfig.builder()
                .workTask(workflowTask)
                .pure(true)
                .metadata(Collections.singletonMap("HELLO", "WORLD"))
                .build();
        SyncWorkflowTask.exeWorkflow(new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assertions.assertEquals("WORLD", segment.getMetadata().get("HELLO"));
                Assertions.assertNull(segment.getMetadata().get("YES"));
            }
        }, syncConfig);
    }

    @Test
    public void testConfigCallable() throws Exception {
        SyncCallable syncCallable = new NotifierCallable("HELLO") {
            public void call(Segment segment) throws Exception {
                Assertions.assertEquals("WORLD", segment.getContent());
            }
        };
        SyncConfig syncConfig = SyncConfig.builder()
                .workTask(ObjectBuilder.buildWorkflowTask())
                .syncCallable(syncCallable)
                .build();
        SyncWorkflowTask.exeWorkflow(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"), syncConfig);
    }

    @Test
    public void testConfigReQuery() throws Exception {
        SyncConfig syncConfig = SyncConfig.builder()
                .workTask(ObjectBuilder.buildWorkflowTask())
                .reQuery("HELLO WORLD")
                .build();
        SyncWorkflowTask.exeWorkflow(new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assertions.assertEquals("HELLO WORLD", segment.getContent());
            }
        }, syncConfig);
    }

    @Test
    public void testConfigTimeout() throws Exception {
        SyncConfig syncConfig = SyncConfig.builder()
                .workTask(ObjectBuilder.buildWorkflowTask())
                .timeout(1000)
                .build();
        SyncWorkflowTask.exeWorkflow(new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assertions.assertEquals(Integer.valueOf(1000), SyncWorkflowTask.class.cast(notifierWriteBack).getTimeout());
            }
        }, syncConfig);
    }

    @Test
    public void testConfigWorkflow() throws Exception {
        SyncConfig syncConfig = SyncConfig.builder()
                .workTask(ObjectBuilder.buildWorkflowTask())
                .workflow("WORKFLOW_1")
                .build();
        SyncWorkflowTask.exeWorkflow(new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assertions.assertEquals("WORKFLOW_1", segment.getWorkflow());
            }
        }, syncConfig);
    }

    @Test
    public void testConfigBiz() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setBiz("TASK_BIZ");
        SyncConfig syncConfig = SyncConfig.builder()
                .workTask(workflowTask)
                .biz("CONFIG_BIZ")
                .workflow("WORKFLOW_BIZ")
                .build();
        SyncWorkflowTask.exeWorkflow(new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assertions.assertEquals("CONFIG_BIZ", segment.getBiz());
                Assertions.assertEquals("WORKFLOW_BIZ", segment.getWorkflow());
                Assertions.assertEquals("CONFIG_BIZ", SyncWorkflowTask.class.cast(notifierWriteBack).getBiz());
            }
        }, syncConfig);
    }

    @Test
    public void testConfigSceneWorkflowOverridesBiz() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setBiz("TASK_BIZ");
        SyncConfig syncConfig = SyncConfig.builder()
                .workTask(workflowTask)
                .biz("CONFIG_BIZ")
                .workflow("SCENE_BIZ@WORKFLOW_SCENE")
                .build();
        SyncWorkflowTask.exeWorkflow(new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assertions.assertEquals("SCENE_BIZ", segment.getBiz());
                Assertions.assertEquals("WORKFLOW_SCENE", segment.getWorkflow());
                Assertions.assertEquals("SCENE_BIZ", SyncWorkflowTask.class.cast(notifierWriteBack).getBiz());
            }
        }, syncConfig);
    }

    @Test
    public void testConfigWithoutWorkTaskThrowsHelpfulMessage() {
        SyncConfig syncConfig = SyncConfig.builder().build();
        IllegalArgumentException ex = Assertions.assertThrows(IllegalArgumentException.class, () ->
                SyncWorkflowTask.exeWorkflow(new NotifierServiceImpl(), syncConfig));
        Assertions.assertEquals("WorkTask can not be empty", ex.getMessage());
    }

    @Test
    public void testConfigBlankBizThrowsHelpfulMessage() {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setBiz("");
        SyncConfig syncConfig = SyncConfig.builder()
                .workTask(workflowTask)
                .biz("")
                .build();
        IllegalArgumentException ex = Assertions.assertThrows(IllegalArgumentException.class, () ->
                SyncWorkflowTask.exeWorkflow(new NotifierServiceImpl(), syncConfig));
        Assertions.assertEquals("Biz can not be empty", ex.getMessage());
    }

    @Test
    public void testConfigMediaContext() throws Exception {
        List<MediaContext> _mediaContexts = new ArrayList<MediaContext>();
        SyncConfig syncConfig = SyncConfig.builder()
                .workTask(ObjectBuilder.buildWorkflowTask())
                .mediaContext(_mediaContexts)
                .build();
        SyncWorkflowTask.exeWorkflow(new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assertions.assertEquals(_mediaContexts, mediaContext);
            }
        }, syncConfig);
    }

    @Test
    public void testWriteSource() throws Exception {
        WorkflowTask workflowTask = EasyMock.createMock(WorkflowTask.class);
        EasyMock.expect(workflowTask.getNotifier()).andReturn("NOTIFY").anyTimes();
        EasyMock.expect(workflowTask.getWorkflow()).andReturn("WORKFLOW").anyTimes();
        EasyMock.expect(workflowTask.getDeepness()).andReturn(1).anyTimes();
        EasyMock.expect(workflowTask.getOriginal()).andReturn("ORIGINAL").anyTimes();
        EasyMock.expect(workflowTask.getPrevious()).andReturn("PREVIOUS").anyTimes();
        EasyMock.expect(workflowTask.getInitial()).andReturn("INITIAL").anyTimes();
        EasyMock.expect(workflowTask.getQuery()).andReturn("QUERY").anyTimes();
        EasyMock.expect(workflowTask.getMetadata()).andReturn(new HashMap<>()).anyTimes();
        EasyMock.expect(workflowTask.getConversation()).andReturn("CONVERSATION").anyTimes();
        EasyMock.expect(workflowTask.getChat()).andReturn("CHAT").anyTimes();
        EasyMock.expect(workflowTask.getTakeover()).andReturn(null).anyTimes();
        EasyMock.expect(workflowTask.getBiz()).andReturn("BIZ").anyTimes();
        EasyMock.expect(workflowTask.getUpstream()).andReturn("UPSTREAM").anyTimes();
        EasyMock.expect(workflowTask.getProtocol()).andReturn("PROTOCOL").anyTimes();
        EasyMock.expect(workflowTask.getUserContext()).andReturn(ObjectBuilder.buildEmpty()).anyTimes();
        EasyMock.expect(workflowTask.containFunCallTrack()).andReturn(false).anyTimes();
        EasyMock.expect(workflowTask.getFunCallTrack()).andReturn(null).anyTimes();
        EasyMock.expect(workflowTask.getCreated()).andReturn(10086L).anyTimes();
        workflowTask.setWorkflow("WORKFLOW");
        EasyMock.expectLastCall().anyTimes();
        workflowTask.setNotifier("endpoint");
        EasyMock.expectLastCall().anyTimes();
        Segment segment = EasyMock.createMock(Segment.class);
        workflowTask.writeSource(segment);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(workflowTask.getHistories()).andReturn(new ArrayList<>()).anyTimes();
        EasyMock.replay(segment, workflowTask);
        SyncWorkflowTask syncWorkflowTask = new SyncWorkflowTask(workflowTask, null, 1000);
        syncWorkflowTask.writeSource(segment);
        EasyMock.verify(segment, workflowTask);
    }

    @Test
    public void testMeta() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.putMetadata("A", "B");
        SyncConfig syncConfig = SyncConfig.builder()
                .workTask(workflowTask)
                .metadata(ImmutableMap.of("C", "D"))
                .build();
        SyncWorkflowTask.exeWorkflow(ObjectBuilder.buildActualNotifierManagerWithNothing(), syncConfig);
        Assertions.assertEquals(Integer.valueOf(1), Integer.valueOf(workflowTask.getMetadata().size()));
        Assertions.assertEquals("B", workflowTask.getMetadata().get("A"));
    }

    @Test
    public void testTakeover() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.putMetadata("YES", "NO");
        SyncConfig syncConfig = SyncConfig.builder()
                .workTask(workflowTask)
                .takeover("TK")
                .metadata(Collections.singletonMap("HELLO", "WORLD"))
                .pure(true)
                .build();
        SyncWorkflowTask.exeWorkflow(new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assertions.assertEquals("TK", notifierWriteBack.getTakeover());
            }
        }, syncConfig);
    }

    @Test
    public void testConstructorPure() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        task.putMetadata("K", "V");
        SyncWorkflowTask swt = new SyncWorkflowTask(task, null, 100, true);
        Assertions.assertTrue(swt.getMetadata().isEmpty());
    }

    @Test
    public void testAddMediaContextNull() {
        SyncWorkflowTask swt = new SyncWorkflowTask(ObjectBuilder.buildWorkflowTask(), null, 100);
        swt.setMediaContext(null);
        swt.addMediaContext(new MediaContext());
        Assertions.assertEquals(1, swt.getMediaContext().size());
    }

    @Test
    public void testConstructors() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        SyncWorkflowTask swt1 = new SyncWorkflowTask(task, "TK", 100);
        Assertions.assertEquals("TK", swt1.getTakeover());

        SyncWorkflowTask swt2 = new SyncWorkflowTask(task, "TK", 100, true);
        Assertions.assertTrue(swt2.getMetadata().isEmpty());
    }

    @Test
    public void testExeWorkflowDefaults() throws Exception {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        SyncConfig config = SyncConfig.builder()
                .workTask(task)
                .build();
        // reQuery and workflow will use task's values
        SyncWorkflowTask.exeWorkflow(ObjectBuilder.buildActualNotifierManagerWithNothing(), config);
    }

    @Test
    public void testIncrDeepness_delegatesToWorkTask() {
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQuery();
        SyncWorkflowTask syncTask = new SyncWorkflowTask(workflowTask, null, 1000);
        Integer before = workflowTask.getDeepness();
        syncTask.incrDeepness();
        Assertions.assertEquals(before != null ? before + 1 : 1, workflowTask.getDeepness());
    }

    /**
     * markQuery 记录当前 query 到 markQuery；resetQuery 将 query 恢复为 markQuery
     */
    @Test
    public void testMarkQueryAndResetQuery() {
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQuery();
        workflowTask.setQuery("initial");
        SyncWorkflowTask syncTask = new SyncWorkflowTask(workflowTask, null, 1000);
        Assertions.assertEquals("initial", syncTask.getQuery());
        syncTask.markQuery();
        syncTask.setQuery("modified");
        Assertions.assertEquals("modified", syncTask.getQuery());
        syncTask.resetQuery();
        Assertions.assertEquals("initial", syncTask.getQuery());
    }

    @Test
    public void getCreated_returnsNonNullCreationTime() {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        SyncWorkflowTask syncTask = new SyncWorkflowTask(workflowTask, null, 1000);
        Assertions.assertNotNull(syncTask.getCreated());
        Assertions.assertTrue(Math.abs(syncTask.getCreated() - System.currentTimeMillis()) < 2000L, "getCreated should be around current time");
    }

    /**
     * 六参构造：Created 用于计算超时（来自 super 的 System.currentTimeMillis），Timestamp 来自 WorkTask 起始时间
     */
    @Test
    public void constructorSixArgs_setsCreatedForTimeoutAndTimestampFromWorkTask() {
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        long before = System.currentTimeMillis();
        SyncWorkflowTask task = new SyncWorkflowTask(workTask, null, "TK", 500, 2000, false);
        long after = System.currentTimeMillis();
        Assertions.assertNotNull(task.getCreated(), "Created used for timeout calculation");
        Assertions.assertTrue(task.getCreated() >= before && task.getCreated() <= after + 50,
                "getCreated should be the construction time passed to super");
        Assertions.assertEquals(workTask.getCreated(), task.getCreated(), "Timestamp from WorkTask start time");
        Assertions.assertEquals(Integer.valueOf(500), task.getInterval());
        Assertions.assertEquals(Integer.valueOf(2000), task.getTimeout());
        Assertions.assertEquals("TK", task.getTakeover());
    }

    @Test
    public void constructor_fullEightArgs_setsChatTakeoverNotifierIntervalTimeoutAndPure() {
        WorkflowTask wt = newTaskForConstructors();
        SyncCallable sc = noopSyncCallable();
        SyncWorkflowTask t = new SyncWorkflowTask(wt, sc, "T_O", "N_O", 111, 2222, "CH_O", true);
        Assertions.assertEquals("CH_O", t.getChat());
        Assertions.assertEquals("T_O", t.getTakeover());
        Assertions.assertEquals("N_O", t.getNotifier());
        Assertions.assertEquals(Integer.valueOf(111), t.getInterval());
        Assertions.assertEquals(Integer.valueOf(2222), t.getTimeout());
        Assertions.assertTrue(t.getMetadata().isEmpty());
    }

    @Test
    public void constructor_fullEightArgs_pureFalse_copiesWorkTaskMetadata() {
        WorkflowTask wt = newTaskForConstructors();
        SyncWorkflowTask t = new SyncWorkflowTask(wt, null, "T", "N", 1, 2, "C", false);
        Assertions.assertEquals("META_V", t.getMetadata().get("META_K"));
    }

    @Test
    public void constructor_syncCallableSevenArgs_usesWorkTaskChat() {
        WorkflowTask wt = newTaskForConstructors();
        SyncCallable sc = noopSyncCallable();
        SyncWorkflowTask t = new SyncWorkflowTask(wt, sc, "T1", "N1", 55, 66, true);
        Assertions.assertEquals("WT_CHAT", t.getChat());
        Assertions.assertEquals(Integer.valueOf(55), t.getInterval());
        Assertions.assertTrue(t.getMetadata().isEmpty());
    }

    @Test
    public void constructor_syncCallableSixArgs_defaultsNotifierFromWorkTask() {
        WorkflowTask wt = newTaskForConstructors();
        SyncCallable sc = noopSyncCallable();
        SyncWorkflowTask t = new SyncWorkflowTask(wt, sc, "T1", 77, 88, false);
        Assertions.assertEquals("WT_NOTIFIER", t.getNotifier());
        Assertions.assertEquals(Integer.valueOf(77), t.getInterval());
        Assertions.assertEquals("META_V", t.getMetadata().get("META_K"));
    }

    @Test
    public void constructor_takeoverNotifierTimeoutChatPure() {
        WorkflowTask wt = newTaskForConstructors();
        SyncWorkflowTask t = new SyncWorkflowTask(wt, "TA", "NO", 900, "CX", true);
        Assertions.assertEquals("TA", t.getTakeover());
        Assertions.assertEquals("NO", t.getNotifier());
        Assertions.assertEquals(Integer.valueOf(900), t.getTimeout());
        Assertions.assertEquals("CX", t.getChat());
        Assertions.assertTrue(t.getMetadata().isEmpty());
    }

    @Test
    public void constructor_takeoverNotifierTimeoutPure_usesWorkTaskChat() {
        WorkflowTask wt = newTaskForConstructors();
        SyncWorkflowTask t = new SyncWorkflowTask(wt, "TA", "NO", 901, false);
        Assertions.assertEquals("WT_CHAT", t.getChat());
        Assertions.assertEquals("META_V", t.getMetadata().get("META_K"));
    }

    @Test
    public void constructor_takeoverNotifierTimeoutChat_notPureCopiesMetadata() {
        WorkflowTask wt = newTaskForConstructors();
        SyncWorkflowTask t = new SyncWorkflowTask(wt, "TA", "NO", 902, "CZ");
        Assertions.assertEquals("CZ", t.getChat());
        Assertions.assertEquals("META_V", t.getMetadata().get("META_K"));
    }

    @Test
    public void constructor_takeoverTimeoutChatPure_defaultsNotifier() {
        WorkflowTask wt = newTaskForConstructors();
        SyncWorkflowTask t = new SyncWorkflowTask(wt, "TB", 903, "CY", true);
        Assertions.assertEquals("WT_NOTIFIER", t.getNotifier());
        Assertions.assertEquals("CY", t.getChat());
        Assertions.assertTrue(t.getMetadata().isEmpty());
    }

    @Test
    public void constructor_takeoverNotifierTimeout_defaultsChatAndNotPure() {
        WorkflowTask wt = newTaskForConstructors();
        SyncWorkflowTask t = new SyncWorkflowTask(wt, "TC", "N2", 904);
        Assertions.assertEquals("WT_CHAT", t.getChat());
        Assertions.assertEquals("N2", t.getNotifier());
        Assertions.assertEquals("META_V", t.getMetadata().get("META_K"));
    }

    @Test
    public void constructor_takeoverTimeoutPure_defaultsNotifierAndChat() {
        WorkflowTask wt = newTaskForConstructors();
        SyncWorkflowTask t = new SyncWorkflowTask(wt, "TD", 905, true);
        Assertions.assertEquals("WT_NOTIFIER", t.getNotifier());
        Assertions.assertEquals("WT_CHAT", t.getChat());
        Assertions.assertTrue(t.getMetadata().isEmpty());
    }

    @Test
    public void constructor_takeoverTimeoutChat_defaultsNotifierNotPure() {
        WorkflowTask wt = newTaskForConstructors();
        SyncWorkflowTask t = new SyncWorkflowTask(wt, "TE", 906, "CZZ");
        Assertions.assertEquals("WT_NOTIFIER", t.getNotifier());
        Assertions.assertEquals("CZZ", t.getChat());
        Assertions.assertEquals("META_V", t.getMetadata().get("META_K"));
    }

    @Test
    public void constructor_takeoverTimeout_defaultsNotifierChatAndNotPure() {
        WorkflowTask wt = newTaskForConstructors();
        SyncWorkflowTask t = new SyncWorkflowTask(wt, "TF", 907);
        Assertions.assertEquals("WT_NOTIFIER", t.getNotifier());
        Assertions.assertEquals("WT_CHAT", t.getChat());
        Assertions.assertEquals("META_V", t.getMetadata().get("META_K"));
    }

    @Test
    public void metadata_pureTrue_ignoresWorkTaskMetadata() throws Exception {
        WorkflowTask wt = newTaskForConstructors();
        SyncWorkflowTask sync = new SyncWorkflowTask(wt, null, "T", "N", 1, 2, "C", true);
        Assertions.assertTrue(sync.getMetadata().isEmpty());
        Assertions.assertNull(sync.getMetadata("META_K", String.class));
        sync.putMetadata("NEW", "OK");
        Assertions.assertNull(wt.getMetadata("NEW", String.class));
    }

    /**
     * 构造时 this.histories = getReferenceHistory(workTask.getHistories(), REFERENCE_CLIENT)，仅保留客户端 reference 的 History
     */
    @Test
    public void constructor_histories_keepsOnlyClientReferenceHistories() {
        WorkflowTask wt = ObjectBuilder.buildWorkflowTask();
        History serverH = new History();
        serverH.setReference(History.REFERENCE_SERVER);
        serverH.setContent("srv");
        History clientH = new History();
        clientH.setReference(History.REFERENCE_CLIENT);
        clientH.setContent("cli");
        wt.addHistories(Arrays.asList(serverH, clientH));
        SyncWorkflowTask sync = new SyncWorkflowTask(wt, null, 1000);
        List<History> out = sync.getHistories();
        Assertions.assertEquals(1, out.size());
        Assertions.assertSame(clientH, out.get(0));
        Assertions.assertEquals("cli", out.get(0).getContent());
    }

    @Test
    public void constructor_histories_noClientMatch_yieldsEmptyHistories() {
        WorkflowTask wt = ObjectBuilder.buildWorkflowTask();
        History serverOnly = new History();
        serverOnly.setReference(History.REFERENCE_SERVER);
        serverOnly.setContent("srv");
        wt.addHistories(Collections.singletonList(serverOnly));
        SyncWorkflowTask sync = new SyncWorkflowTask(wt, null, 1000);
        Assertions.assertTrue(sync.getHistories().isEmpty());
    }

    @Test
    public void constructor_histories_workTaskWithoutHistories_yieldsEmpty() {
        WorkflowTask wt = ObjectBuilder.buildWorkflowTask();
        SyncWorkflowTask sync = new SyncWorkflowTask(wt, null, 1000);
        Assertions.assertTrue(sync.getHistories().isEmpty());
    }

    @Test
    public void constructor_histories_multipleClientReferences_allRetained() {
        WorkflowTask wt = ObjectBuilder.buildWorkflowTask();
        History c1 = new History();
        c1.setReference(History.REFERENCE_CLIENT);
        c1.setContent("c1");
        History c2 = new History();
        c2.setReference(History.REFERENCE_CLIENT);
        c2.setContent("c2");
        wt.addHistories(Arrays.asList(c1, c2));
        SyncWorkflowTask sync = new SyncWorkflowTask(wt, null, 1000);
        List<History> out = sync.getHistories();
        Assertions.assertEquals(2, out.size());
        Assertions.assertTrue(out.contains(c1));
        Assertions.assertTrue(out.contains(c2));
    }
}
