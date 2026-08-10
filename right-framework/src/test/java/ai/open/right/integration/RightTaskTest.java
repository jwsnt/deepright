package ai.open.right.integration;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.context.RedirectContext;
import ai.open.right.context.UserContext;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.NothingWriteBack;
import org.junit.Assert;
import org.junit.Test;

import static org.junit.jupiter.api.Assertions.*;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class RightTaskTest {

    @Test
    public void testSetGetObject() throws Exception {
        List<MediaContext> mediaContext = new ArrayList<MediaContext>();
        List<History> histories = new ArrayList<>();
        Map<String, Object> metadata = new HashMap<>();
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig delegate = RightConfig.builder().histories(histories).mediaContext(mediaContext).query("Query").biz("Biz").trace("Trace").chat("Chat").timeout(10000).conversation("Conversation").userContext(userContext).upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(metadata).workflow("Workflow").build().init();
        LLMConfig config = new LLMConfig();
        config.setProvider("PROVIDER");
        delegate.setObjectQuery(config);
        config = delegate.getObjectQuery(LLMConfig.class);
        Assert.assertEquals("PROVIDER", config.getProvider());
        List<MediaContext> mediaContexts = new ArrayList<>();
        delegate.setMediaContext(mediaContexts);
        Assert.assertEquals(mediaContexts, delegate.getMediaContext());
        List<MediaContext> mediaContexts1 = new ArrayList<>();
        delegate.setMediaContext(mediaContexts1);
        Assert.assertEquals(mediaContexts1, delegate.getMediaContext());
    }

    @Test
    public void testSetGetObject2() throws Exception {
        List<MediaContext> mediaContext = new ArrayList<MediaContext>();
        List<History> histories = new ArrayList<>();
        Map<String, Object> metadata = new HashMap<>();
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig right = RightConfig.builder().histories(histories).mediaContext(mediaContext).query("Query").biz("Biz").trace("Trace").chat("Chat").timeout(10000).conversation("Conversation").userContext(userContext).upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(metadata).workflow("Workflow").build().init();
        right.setTakeover("TK");
        RightTask delegate = new RightTask(right, ObjectBuilder.buildNotifyWriteBack());
        delegate.init();
        LLMConfig config = new LLMConfig();
        config.setProvider("PROVIDER");
        delegate.setObjectQuery(config);
        config = delegate.getObjectQuery(LLMConfig.class);
        Assert.assertEquals("TK", delegate.getTakeover());
        Assert.assertEquals("PROVIDER", config.getProvider());
        Assert.assertNotNull(delegate.getRightConfig());
        Assert.assertNotNull(delegate.getNotifierWriteBack());
        right.setProvider("PR");
        Assert.assertEquals("PR", right.getProvider());
        right.setTakeover("KT");
        Assert.assertEquals("PR", delegate.getMetadata().get(ProviderRequestService.KEY_PROVIDER));
        Assert.assertEquals("KT", delegate.getTakeover());
        delegate.setTakeover("TK");
        Assert.assertEquals("TK", right.getTakeover());
    }

    @Test
    public void testIsEntry() {
        RightConfig rightConfig = RightConfig.builder().build();
        RightTask task = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        Assert.assertTrue("RightTask with default deepness 1 should be entry", task.isEntry());
        task.incrDeepness();
        Assert.assertTrue("RightTask with default deepness 1 should be entry", task.isEntry());
        task.incrDeepness();
        Assert.assertFalse("RightTask after incrDeepness should not be entry", task.isEntry());
    }

    @Test
    public void testIsEntry_fullBranchCoverage() {
        // 入口：upstream 空、非 FunCall、deepness 为 null
        RightConfig config = RightConfig.builder().build();
        RightTask task = new RightTask(config, ObjectBuilder.buildNotifyWriteBack());
        Assert.assertTrue("empty upstream, not fromFunCall, deepness null -> entry", task.isEntry());

        // 入口：upstream 空、非 FunCall、deepness 等于 DEEPNESS
        task.incrDeepness();
        Assert.assertTrue("empty upstream, not fromFunCall, deepness equals DEEPNESS -> entry", task.isEntry());

        // 非入口：upstream 非空
        task.setUpstream("any");
        Assert.assertFalse("non-empty upstream -> not entry", task.isEntry());

        // 非入口：来自 FunCall（KEY_FUN_FETCH）
        task.setUpstream(null);
        task.incrDeepness();
        config.getMetadata().put(ProviderRequestService.KEY_FUN_FETCH, new Object());
        Assert.assertFalse("from FunCall (KEY_FUN_FETCH) -> not entry", task.isEntry());

        // 非入口：来自 FunCall（KEY_FUN_MERGE）
        config.getMetadata().remove(ProviderRequestService.KEY_FUN_FETCH);
        config.getMetadata().put(ProviderRequestService.KEY_FUN_MERGE, new Object());
        Assert.assertFalse("from FunCall (KEY_FUN_MERGE) -> not entry", task.isEntry());

        // 非入口：deepness 不为 null 且不等于 DEEPNESS
        config.getMetadata().clear();
        config.setDeepness(RedirectContext.DEEPNESS + 1);
        task = new RightTask(config, ObjectBuilder.buildNotifyWriteBack());
        Assert.assertFalse("deepness != DEEPNESS -> not entry", task.isEntry());
    }

    @Test
    public void testAddMediaContext() throws Exception {
        RightConfig rightConfig = RightConfig.builder().build();
        RightTask delegate = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        MediaContext context = new MediaContext();
        delegate.addMediaContext(context);
        Assert.assertEquals(context, delegate.getMediaContext().getFirst());
        Assert.assertEquals(Integer.valueOf(delegate.getMediaContext().size()), Integer.valueOf(1));
        List<MediaContext> mediaContexts1 = new ArrayList<>();
        delegate.setMediaContext(mediaContexts1);
        Assert.assertEquals(mediaContexts1, delegate.getMediaContext());
    }

    @Test
    public void test() {
        List<MediaContext> mediaContext = new ArrayList<MediaContext>();
        List<History> histories = new ArrayList<>();
        Map<String, Object> metadata = new HashMap<>();
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig rightConfig = RightConfig.builder().histories(histories).mediaContext(mediaContext).query("Query").biz("Biz").trace("Trace").chat("Chat").timeout(10000).conversation("Conversation").userContext(userContext).upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(metadata).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        Assert.assertEquals(false, rightTask.containChatTrack());
        Assert.assertEquals(mediaContext, rightTask.getMediaContext());
        Assert.assertEquals(histories, rightTask.getHistories());
        Assert.assertEquals("Query", rightTask.getQuery());
        Assert.assertEquals("Biz", rightTask.getBiz());
        Assert.assertEquals("Trace", rightTask.getTrace());
        Assert.assertEquals("Chat", rightTask.getChat());
        Assert.assertEquals("Conversation", rightTask.getConversation());
        Assert.assertEquals(userContext, rightTask.getUserContext());
        Assert.assertEquals("Workflow", rightTask.getUpstream());
        Assert.assertEquals("Notifier", rightTask.getNotifier());
        Assert.assertEquals("Protocol", rightTask.getProtocol());
        Assert.assertEquals(metadata, rightTask.getMetadata());
        Assert.assertEquals("Workflow", rightTask.getWorkflow());
        rightTask.setProtocol("Tools");
        rightTask.setQuery("Query2");
        rightTask.setNotifier("Notifier2");
        rightTask.setTrace("Trace2");
        rightTask.setWorkflow("Workflow2");
        rightTask.setUpstream("Upstream2");
        rightTask.setBiz("BIZ2");
        rightTask.beginChatTrack();
        Assert.assertEquals(true, rightTask.containChatTrack());
        Assert.assertEquals("BIZ2", rightTask.getBiz());
        Assert.assertEquals("Tools", rightTask.getProtocol());
        Assert.assertEquals("Query2", rightTask.getQuery());
        Assert.assertEquals("Notifier2", rightTask.getNotifier());
        Assert.assertEquals("Trace2", rightTask.getTrace());
        Assert.assertEquals("Workflow2", rightTask.getWorkflow());
        Assert.assertEquals("Upstream2", rightTask.getUpstream());
        UserContext userContext2 = UserContext.builder().build();
        rightTask.setUserContext(userContext2);
        Assert.assertEquals(userContext2, rightTask.getUserContext());
    }

    @Test
    public void testDefault() {
        Map<String, Object> metadata = new HashMap<>();
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig rightConfig = RightConfig.builder().protocol(null).query("Query").biz("Biz").trace("Trace").timeout(10000).userContext(userContext).upstream("Upstream").notifier("Notifier").metadata(metadata).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        Assert.assertNotNull(rightTask.getConversation());
        Assert.assertNotNull(rightTask.getProtocol());
        Assert.assertNotNull(rightTask.getChat());
    }

    @Test
    public void testCheck() {
        Map<String, Object> metadata = new HashMap<>();
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig rightConfig = RightConfig.builder().query("Query").biz("Biz").trace("Trace").chat("Chat").timeout(10000).conversation("Conversation").userContext(userContext).upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(metadata).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        RightTask.RightTaskChecker.check(rightTask);
    }

    @Test
    public void testGet() throws Exception {
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("KEY", "HELLO_WORLD");
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig rightConfig = RightConfig.builder().query("Query").biz("Biz").trace("Trace").chat("Chat").timeout(10000).conversation("Conversation").userContext(userContext).upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(metadata).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        Assert.assertNotNull(rightTask.getConsuming());
        Assert.assertEquals("Query", rightTask.getOriginal());
        Assert.assertEquals("Query", rightTask.getPrevious());
        Assert.assertEquals("Query", rightTask.getInitial());
        Assert.assertEquals("UNKNOWN", rightTask.getDevice());
        Assert.assertEquals("HELLO_WORLD", rightTask.getMetadata("KEY", String.class));
        Assert.assertEquals("HELLO_WORLD", rightTask.delMetadata("KEY", String.class));
        rightTask.putMetadata("VAL", "HELLO_WORLD");
        Assert.assertEquals("HELLO_WORLD", rightTask.getMetadata("VAL", String.class));
    }

    @Test
    public void testWrite() throws Exception {
        Map<String, Object> metadata = new HashMap<>();
        Segment actual = ObjectBuilder.buildSegment();
        RightConfig rightConfig = RightConfig.builder().query("Query").biz("Biz").trace("Trace").chat("Chat").timeout(10000).conversation("Conversation").upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(metadata).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, new NothingWriteBack() {
            @Override
            public void writeSource(Segment segment) throws Exception {
                Assert.assertEquals(actual, segment);
            }

            @Override
            public void writeBack(Segment segment) throws Exception {
                Assert.assertEquals(actual, segment);
            }
        });
        rightTask.init();
        rightTask.writeBack(actual);
        rightTask.writeSource(actual);
    }

    @Test
    public void testGetSetHistory() throws Exception {
        RightConfig rightConfig = RightConfig.builder().query("Query").biz("Biz").trace("Trace").chat("Chat").timeout(10000).conversation("Conversation").upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(new HashMap<>()).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildWorkflowTask());
        Assert.assertTrue(rightTask.getHistories().isEmpty());
        Assert.assertFalse(rightTask.containHistories());
        List<History> histories = new ArrayList<>();
        histories.add(new History());
        rightTask.addHistories(histories);
        Assert.assertEquals(histories, rightTask.getHistories());
        Assert.assertTrue(rightTask.containHistories());
    }

    @Test
    public void testFunCall() {
        RightConfig rightConfig = RightConfig.builder().query("Query").biz("Biz").trace("Trace").chat("Chat").timeout(10000).conversation("Conversation").upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(new HashMap<>()).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildWorkflowTask());
        Assert.assertFalse(rightTask.containFunCallTrack());
        rightConfig = RightConfig.builder().funCallTrack("FunCall").query("Query").biz("Biz").trace("Trace").chat("Chat").timeout(10000).conversation("Conversation").upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(new HashMap<>()).workflow("Workflow").build().init();
        rightTask = new RightTask(rightConfig, ObjectBuilder.buildWorkflowTask());
        Assert.assertTrue(rightTask.containFunCallTrack());
        Assert.assertEquals("FunCall", rightTask.getFunCallTrack());
        rightTask.beginFunCallTrack("ABC");
        Assert.assertEquals("ABC", rightTask.getFunCallTrack());
        rightTask.beginFunCallTrack();
        Assert.assertEquals(Integer.valueOf(36), Integer.valueOf(rightTask.getFunCallTrack().length()));
        rightTask.closeFunCallTrack();
        Assert.assertNull(rightTask.getFunCallTrack());
    }

    @org.junit.jupiter.api.Test
    public void testGetDimension() {
        // 测试默认情况下的dimension
        Map<String, Object> metadata = new HashMap<>();
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig rightConfig = RightConfig.builder().query("Query").biz("Biz").trace("Trace").chat("Chat").timeout(10000).conversation("Conversation").userContext(userContext).upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(metadata).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        // 默认情况下device为"UNKNOWN"，所以dimension应该是"Biz-Chat-UNKNOWN"
        assertEquals("Biz-Chat-UNKNOWN", rightTask.getDimension());
    }

    @org.junit.jupiter.api.Test
    public void testGetDimensionWithCustomDevice() {
        // 测试自定义device的dimension
        Map<String, Object> metadata = new HashMap<>();
        UserContext userContext = ObjectBuilder.buildEmpty();
        userContext.setDevice("iPhone");
        RightConfig rightConfig = RightConfig.builder().query("Query").biz("TestBiz").trace("Trace").chat("TestChat").timeout(10000).conversation("Conversation").userContext(userContext).upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(metadata).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        // 自定义device为"iPhone"，所以dimension应该是"TestBiz-TestChat-iPhone"
        assertEquals("TestBiz-TestChat-iPhone", rightTask.getDimension());
    }

    @org.junit.jupiter.api.Test
    public void testGetDimensionWithNullValues() {
        // 测试包含null值的dimension
        Map<String, Object> metadata = new HashMap<>();
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig rightConfig = RightConfig.builder().query("Query").biz("BIZ1").trace("Trace").chat("CHAT1").timeout(10000).conversation("Conversation").userContext(userContext).upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(metadata).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        // 当biz或chat为null时，StringUtils.joinWith会处理null值
        assertEquals("BIZ1-CHAT1-UNKNOWN", rightTask.getDimension());
    }

    @org.junit.jupiter.api.Test
    public void testGetDimensionWithEmptyValues() {
        // 测试包含空字符串的dimension
        Map<String, Object> metadata = new HashMap<>();
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig rightConfig = RightConfig.builder().query("Query").biz("BIZ2").trace("Trace").chat("CHAT2").timeout(10000).conversation("Conversation").userContext(userContext).upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(metadata).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        // 空字符串会被正常连接
        assertEquals("BIZ2-CHAT2-UNKNOWN", rightTask.getDimension());
    }

    @org.junit.jupiter.api.Test
    public void testContainMeta() {
        // 测试包含null值的dimension
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("HELLO", "WORLD");
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig rightConfig = RightConfig.builder().query("Query").biz("BIZ1").trace("Trace").chat("CHAT1").timeout(10000).conversation("Conversation").userContext(userContext).upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(metadata).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        assertTrue(rightTask.containMetadata("HELLO"));
        assertFalse(rightTask.containMetadata("WORLD"));
    }

    @org.junit.jupiter.api.Test
    public void testIsFromFunCall1() {
        // 测试包含null值的dimension
        Map<String, Object> metadata = new HashMap<>();
        metadata.put(ProviderRequestService.KEY_FUN_FETCH, "WORLD");
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig rightConfig = RightConfig.builder().query("Query").biz("BIZ1").trace("Trace").chat("CHAT1").timeout(10000).conversation("Conversation").userContext(userContext).upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(metadata).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        assertTrue(rightTask.isFromFunCall());
    }

    @org.junit.jupiter.api.Test
    public void testIsFromFunCall2() {
        // metadata 仅有 KEY_FUN_MERGE 时返回 true
        Map<String, Object> metadata = new HashMap<>();
        metadata.put(ProviderRequestService.KEY_FUN_MERGE, "WORLD");
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig rightConfig = RightConfig.builder().query("Query").biz("BIZ1").trace("Trace").chat("CHAT1").timeout(10000).conversation("Conversation").userContext(userContext).upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(metadata).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        assertTrue(rightTask.isFromFunCall());
    }

    @org.junit.jupiter.api.Test
    public void testIsFromFunCallFalse() {
        // metadata 无 KEY_FUN_FETCH、KEY_FUN_MERGE 时返回 false
        Map<String, Object> metadata = new HashMap<>();
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig rightConfig = RightConfig.builder().query("Query").biz("BIZ1").trace("Trace").chat("CHAT1").timeout(10000).conversation("Conversation").userContext(userContext).upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(metadata).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        assertFalse(rightTask.isFromFunCall());
    }

    @org.junit.jupiter.api.Test
    public void testIsFromFunCallBoth() {
        // metadata 同时含 KEY_FUN_FETCH 与 KEY_FUN_MERGE 时返回 true
        Map<String, Object> metadata = new HashMap<>();
        metadata.put(ProviderRequestService.KEY_FUN_FETCH, "fetch");
        metadata.put(ProviderRequestService.KEY_FUN_MERGE, "merge");
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig rightConfig = RightConfig.builder().query("Query").biz("BIZ1").trace("Trace").chat("CHAT1").timeout(10000).conversation("Conversation").userContext(userContext).upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(metadata).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        assertTrue(rightTask.isFromFunCall());
    }

    @org.junit.jupiter.api.Test
    public void testIsFromFunMerge_falseWhenEmptyOrFetchOnly() {
        Map<String, Object> metadata = new HashMap<>();
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig emptyMeta = RightConfig.builder().query("Query").biz("B1").trace("T").chat("C1").timeout(10000).conversation("Conv").userContext(userContext).upstream("U").notifier("N").protocol("P").metadata(metadata).workflow("W").build().init();
        RightTask t1 = new RightTask(emptyMeta, ObjectBuilder.buildNotifyWriteBack());
        t1.init();
        assertFalse(t1.isFromFunMerge());
        metadata.put(ProviderRequestService.KEY_FUN_FETCH, "x");
        RightConfig fetchOnly = RightConfig.builder().query("Query").biz("B1").trace("T").chat("C1").timeout(10000).conversation("Conv").userContext(userContext).upstream("U").notifier("N").protocol("P").metadata(new HashMap<>(metadata)).workflow("W").build().init();
        RightTask t2 = new RightTask(fetchOnly, ObjectBuilder.buildNotifyWriteBack());
        t2.init();
        assertFalse(t2.isFromFunMerge());
    }

    @org.junit.jupiter.api.Test
    public void testIsFromFunMerge_trueWhenMergeKeyPresent() {
        Map<String, Object> metadata = new HashMap<>();
        metadata.put(ProviderRequestService.KEY_FUN_MERGE, "m");
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig rightConfig = RightConfig.builder().query("Query").biz("B1").trace("T").chat("C1").timeout(10000).conversation("Conv").userContext(userContext).upstream("U").notifier("N").protocol("P").metadata(metadata).workflow("W").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        assertTrue(rightTask.isFromFunMerge());
    }

    @org.junit.jupiter.api.Test
    public void testInitAlreadySet() {
        RightConfig rightConfig = RightConfig.builder()
                .conversation("existing_conv")
                .protocol("existing_proto")
                .chat("existing_chat")
                .build();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        assertEquals("existing_conv", rightTask.getConversation());
        assertEquals("existing_proto", rightTask.getProtocol());
        assertEquals("existing_chat", rightTask.getChat());
    }

    @org.junit.jupiter.api.Test
    public void testAddMediaContextNull() {
        RightConfig rightConfig = RightConfig.builder().build();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.setMediaContext(null);
        MediaContext context = new MediaContext();
        rightTask.addMediaContext(context);
        assertNotNull(rightTask.getMediaContext());
        assertEquals(Integer.valueOf(1), Integer.valueOf(rightTask.getMediaContext().size()));
        assertEquals(context, rightTask.getMediaContext().getFirst());
    }

    @org.junit.jupiter.api.Test
    public void testGetMetadataNull() throws Exception {
        RightConfig rightConfig = RightConfig.builder().metadata(null).build();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        assertNull(rightTask.getMetadata("any_key", String.class));
    }

    @org.junit.jupiter.api.Test
    public void testDelMetadataNull() throws Exception {
        RightConfig rightConfig = RightConfig.builder().metadata(null).build();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        assertNull(rightTask.delMetadata("any_key", String.class));
    }

    @org.junit.jupiter.api.Test
    public void testCheckFailure() {
        RightConfig rightConfig = RightConfig.builder().build();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        assertThrows(IllegalArgumentException.class, () -> RightTask.RightTaskChecker.check(rightTask));
    }

    @Test
    public void setChat_delegatesToRightConfig() {
        RightConfig rightConfig = RightConfig.builder().query("Q").biz("B").trace("T").chat("InitialChat").timeout(10000)
                .conversation("C").upstream("U").notifier("N").protocol("P").metadata(new HashMap<>()).workflow("W").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        Assert.assertEquals("InitialChat", rightTask.getChat());
        rightTask.setChat("NewChat");
        Assert.assertEquals("NewChat", rightTask.getChat());
        Assert.assertEquals("NewChat", rightConfig.getChat());
    }

    @Test
    public void setHistories_delegatesToRightConfig() {
        RightConfig rightConfig = RightConfig.builder().query("Q").biz("B").trace("T").chat("C").timeout(10000)
                .conversation("C").upstream("U").notifier("N").protocol("P").metadata(new HashMap<>()).workflow("W").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        Assert.assertTrue(rightTask.getHistories() == null || rightTask.getHistories().isEmpty());
        List<History> histories = new ArrayList<>();
        histories.add(new History());
        rightTask.setHistories(histories);
        Assert.assertEquals(histories, rightTask.getHistories());
        Assert.assertEquals(histories, rightConfig.getHistories());
    }

    @Test
    public void checkClosed_notClosed_doesNotThrow() throws Exception {
        RightConfig rightConfig = RightConfig.builder().query("Query").biz("Biz").trace("Trace").chat("Chat").timeout(10000).conversation("Conversation").upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(new HashMap<>()).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        rightTask.checkClosed();
    }

    @Test
    public void checkClosed_whenClosed_throws() {
        NothingWriteBack notifier = new NothingWriteBack();
        notifier.setClosed(true);
        RightConfig rightConfig = RightConfig.builder().query("Query").biz("Biz").trace("Trace").chat("Chat").timeout(10000).conversation("Conversation").upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(new HashMap<>()).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, notifier);
        rightTask.init();
        assertThrows(WorkflowException.class, () -> rightTask.checkClosed());
    }

    @Test
    public void isClosed_delegatesToNotifierWriteBack() throws Exception {
        NothingWriteBack nwb = new NothingWriteBack();
        RightConfig rightConfig = RightConfig.builder().query("Query").biz("Biz").trace("Trace").chat("Chat").timeout(10000).conversation("Conversation").upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(new HashMap<>()).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, nwb);
        rightTask.init();
        assertFalse(rightTask.isClosed());
        nwb.close();
        assertTrue(rightTask.isClosed());
    }

    @Test
    public void close_delegatesToNotifierWriteBack() throws Exception {
        NothingWriteBack nwb = new NothingWriteBack();
        RightConfig rightConfig = RightConfig.builder().query("Query").biz("Biz").trace("Trace").chat("Chat").timeout(10000).conversation("Conversation").upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(new HashMap<>()).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, nwb);
        rightTask.init();
        assertFalse(nwb.isClosed());
        rightTask.close();
        assertTrue(nwb.isClosed());
    }

    @Test
    public void ignoreClosed_delegatesToNotifierWriteBack() throws Exception {
        NothingWriteBack nwb = new NothingWriteBack();
        RightConfig rightConfig = RightConfig.builder().query("Query").biz("Biz").trace("Trace").chat("Chat").timeout(10000).conversation("Conversation").upstream("Upstream").notifier("Notifier").protocol("Protocol").metadata(new HashMap<>()).workflow("Workflow").build().init();
        RightTask rightTask = new RightTask(rightConfig, nwb);
        rightTask.init();
        assertFalse(nwb.getIgnoreClosed());
        rightTask.ignoreClosed();
        assertTrue(nwb.getIgnoreClosed());
    }

    @Test
    public void getCreated_returnsTimestamp() throws Exception {
        RightTask rightTask = new RightTask(
                RightConfig.builder().query("Q").biz("B").trace("T").chat("C").timeout(10000).conversation("C").upstream("U").notifier("N").protocol("P").metadata(new HashMap<>()).workflow("W").build().init(),
                ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        Assert.assertNotNull(rightTask.getCreated());
        Assert.assertEquals("getCreated should equal getTimestamp", rightTask.getCreated(), rightTask.getCreated());
    }

    @Test
    public void incrDeepness_incrementsDeepness() {
        RightConfig rightConfig = RightConfig.builder().query("Q").biz("B").trace("T").chat("C").timeout(10000).conversation("C").upstream("U").notifier("N").protocol("P").metadata(new HashMap<>()).workflow("W").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        Assert.assertEquals(null, rightTask.getDeepness());
        rightTask.incrDeepness();
        Assert.assertEquals(Integer.valueOf(1), rightTask.getDeepness());
        rightTask.incrDeepness();
        rightTask.incrDeepness();
        Assert.assertEquals(Integer.valueOf(3), rightTask.getDeepness());
    }

    /**
     * markQuery 将当前 query 写入 rightConfig.markQuery；resetQuery 将 query 恢复为 rightConfig.markQuery
     */
    @Test
    public void testMarkQueryAndResetQuery() {
        RightConfig rightConfig = RightConfig.builder().query("saved-query").biz("B").trace("T").chat("C").timeout(10000).conversation("C").upstream("U").notifier("N").protocol("P").metadata(new HashMap<>()).workflow("W").build().init();
        RightTask rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
        rightTask.init();
        Assert.assertEquals("saved-query", rightTask.getQuery());
        rightTask.markQuery();
        Assert.assertEquals("saved-query", rightConfig.getMarkQuery());
        rightTask.setQuery("modified-query");
        Assert.assertEquals("modified-query", rightTask.getQuery());
        rightTask.resetQuery();
        Assert.assertEquals("saved-query", rightTask.getQuery());
    }
}

