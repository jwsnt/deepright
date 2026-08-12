package ai.open.right.workflow.notify.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.context.UserContext;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.SegmentDelegate;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.NothingWriteBack;
import ai.open.right.workflow.notify.Notifier;
import com.google.common.collect.ImmutableMap;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.Collections;
import java.util.Date;
import java.util.List;
import java.util.Map;

public class WorkflowTaskImplTest {

    @Test
    public void testSetGetObject() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        Assert.assertEquals("INITIAL", workflowTask.getInitial());
        workflowTask.setTakeover("TK1");
        SegmentDelegate segment = new SegmentDelegate(workflowTask);
        segment.setContent("CONTENT_");
        Assert.assertEquals("INITIAL", segment.getInitial());
        NothingWriteBack nothingWriteBack = new NothingWriteBack();
        LocalhostNotifier.WorkflowTaskImpl delegate = new LocalhostNotifier.WorkflowTaskImpl(segment, nothingWriteBack);
        Assert.assertEquals(segment.getContent(), delegate.getInitial());
        Assert.assertEquals("CONTENT_", delegate.getInitial());
        Assert.assertEquals(segment.getOriginal(), delegate.getOriginal());
        Assert.assertEquals("INITIAL", delegate.getPrevious());
        delegate.setTakeover("TAKEOVER");
        Assert.assertNotNull(delegate.getConsuming());
        Assert.assertNotNull(delegate.getCreated());
        Assert.assertEquals("TAKEOVER", delegate.getTakeover());
        LLMConfig config = new LLMConfig();
        config.setProvider("PROVIDER");
        delegate.setObjectQuery(config);
        config = delegate.getObjectQuery(LLMConfig.class);
        Assert.assertEquals("PROVIDER", config.getProvider());
        Assert.assertEquals("ORIGINAL", delegate.getOriginal());
        Assert.assertEquals("INITIAL", delegate.getPrevious());
        Assert.assertEquals("CONTENT_", delegate.getInitial());
        Assert.assertNotNull(delegate.getTakeover());
        delegate.setTakeover("TK2");
        Assert.assertEquals("TK2", delegate.getTakeover());
        Assert.assertEquals(workflowTask.getCreated(), delegate.getCreated());
        Assert.assertNotNull(delegate.getConsuming());
        delegate.setObjectQuery(ImmutableMap.of("A", "B"));
        Assert.assertEquals("B", delegate.getObjectQuery(Map.class).get("A"));
    }

    @Test
    public void testAddMediaContext() throws Exception {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        NothingWriteBack nothingWriteBack = new NothingWriteBack();
        LocalhostNotifier.WorkflowTaskImpl delegate = new LocalhostNotifier.WorkflowTaskImpl(segment, nothingWriteBack);
        MediaContext context = new MediaContext();
        delegate.addMediaContext(context);
        Assert.assertEquals(context, delegate.getMediaContext().getFirst());
        Assert.assertEquals(Integer.valueOf(delegate.getMediaContext().size()), Integer.valueOf(1));
    }

    @Test
    public void testFromFunCall1() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertFalse(task.isFromFunCall());
        task.putMetadata(ProviderRequestService.KEY_FUN_FETCH, new Object());
        Assert.assertTrue(task.isFromFunCall());
    }

    @Test
    public void testFromFunCall2() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertFalse(task.isFromFunCall());
        task.putMetadata(ProviderRequestService.KEY_FUN_MERGE, new Object());
        Assert.assertTrue(task.isFromFunCall());
    }

    @Test
    public void testFromFunMerge_fetchOnlyFalse_mergeTrue() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertFalse(task.isFromFunMerge());
        task.putMetadata(ProviderRequestService.KEY_FUN_FETCH, new Object());
        Assert.assertFalse(task.isFromFunMerge());
        task.putMetadata(ProviderRequestService.KEY_FUN_MERGE, new Object());
        Assert.assertTrue(task.isFromFunMerge());
    }

    @Test
    public void testIsEntryDelegatesToSegment() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertEquals(segment.isEntry(), task.isEntry());
    }

    @Test
    public void testSetDeepnessDelegatesToSegment() {
        Segment segment = EasyMock.createMock(Segment.class);
        EasyMock.expect(segment.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(segment.getWorkflow()).andReturn("W").anyTimes();
        EasyMock.expect(segment.getBiz()).andReturn("B").anyTimes();
        EasyMock.expect(segment.getOriginal()).andReturn("O").anyTimes();
        EasyMock.expect(segment.getInitial()).andReturn("I").anyTimes();
        EasyMock.expect(segment.getContent()).andReturn("C").anyTimes();
        EasyMock.expect(segment.getChat()).andReturn("CH").anyTimes();
        EasyMock.expect(segment.copy()).andReturn(segment).once();
        segment.init();
        EasyMock.expectLastCall().once();
        segment.setDeepness(3);
        EasyMock.expectLastCall().once();
        EasyMock.expect(segment.getDeepness()).andReturn(3).anyTimes();
        EasyMock.replay(segment);
        NothingWriteBack nwb = new NothingWriteBack();
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, nwb);
        task.setDeepness(3);
        Assert.assertEquals("setDeepness should delegate to segment", Integer.valueOf(3), task.getDeepness());
        EasyMock.verify(segment);
    }

    @Test
    public void testGet() {
        Long time = System.currentTimeMillis();
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        task.setProtocol("Pr");
        task.setBiz("BIZ");
        Assert.assertEquals("BIZ", task.getBiz());
        Assert.assertEquals(task.getDeepness(), Integer.valueOf(1));
        Assert.assertEquals(task.getWorkflow(), segment.getWorkflow());
        Assert.assertNotEquals(task.getBiz(), segment.getBiz());
        Assert.assertEquals(task.getChat(), segment.getChat());
        Assert.assertEquals(task.getConversation(), segment.getConversation());
        Assert.assertEquals(task.getMetadata(), segment.getMetadata());
        // 在LocalhostNotifier最后一步init重置
        Assert.assertEquals(Notifier.LOCALHOST, segment.getNotifier());
        Assert.assertEquals(task.getProtocol(), "Pr");
        Assert.assertEquals(task.getDevice(), "UNKNOWN");
        Assert.assertEquals(task.getQuery(), segment.getContent());
        Assert.assertTrue(segment.getTimestamp() >= time);
        Assert.assertEquals(task.getTrace(), segment.getTrace());
        Assert.assertEquals(task.getUserContext(), segment.getUserContext());
        Assert.assertEquals(task.getUpstream(), segment.getUpstream());
        Assert.assertEquals("BIZ-UNKNOWN-UNKNOWN", task.getDimension());
        UserContext userContext = UserContext.builder().build();
        task.setUserContext(userContext);
        Assert.assertEquals(userContext, task.getUserContext());
    }

    @Test
    public void testSet() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        task.setWorkflow("WORKFLOW");
        task.setNotifier("NOTIFY");
        task.setQuery("QUERY");
        task.setUpstream("UPSTREAM");
        Assert.assertEquals("UPSTREAM", task.getUpstream());
        Assert.assertEquals("WORKFLOW", task.getWorkflow());
        Assert.assertEquals("NOTIFY", task.getNotifier());
        Assert.assertEquals("QUERY", task.getQuery());
    }

    @Test
    public void testMarkQueryAndResetQuery() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setContent("original");
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        task.markQuery();
        task.setQuery("changed");
        Assert.assertEquals("changed", task.getQuery());
        task.resetQuery();
        Assert.assertEquals("original", task.getQuery());
    }

    @Test
    public void emptyQuery_clearsSegmentContent() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());

        Assert.assertSame(task, task.emptyQuery());
        Assert.assertNull(task.getQuery());
    }

    @Test
    public void ignoreClosed_delegatesToNotifierWriteBack() throws Exception {
        NothingWriteBack nwb = new NothingWriteBack();
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, nwb);
        Assert.assertFalse(nwb.getIgnoreClosed());
        task.ignoreClosed();
        Assert.assertTrue(nwb.getIgnoreClosed());
    }

    @Test
    public void testSetMediaContext() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        List<MediaContext> mediaContext = new ArrayList<>();
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack(), mediaContext);
        Assert.assertEquals(mediaContext, task.getMediaContext());
        List<MediaContext> mediaContext2 = new ArrayList<>();
        task.setMediaContext(mediaContext2);
        Assert.assertEquals(mediaContext2, task.getMediaContext());
    }


    @Test
    public void testWriteBack() throws Exception {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack() {

            @Override
            public void writeSource(Segment segment) throws Exception {
                Assert.assertEquals("UNKNOWN", segment.getWorkflow());
            }

            @Override
            public void writeBack(Segment segment) throws Exception {
                Assert.assertEquals("UNKNOWN", segment.getWorkflow());
            }
        });
        task.writeBack(new SegmentDelegate(ObjectBuilder.buildWorkflowTask()));
        task.writeSource(new SegmentDelegate(ObjectBuilder.buildWorkflowTask()));
    }

    @Test(expected = IllegalArgumentException.class)
    public void testSplitWithInvalid() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setWorkflow("biz1@");
        new LocalhostNotifier.WorkflowTaskImpl(segment, null);
    }

    @Test
    public void testMetadata() throws Exception {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertFalse(task.containMetadata("HELLO"));
        task.putMetadata("HELLO", "WORLD");
        Assert.assertTrue(task.containMetadata("HELLO"));
        Assert.assertEquals("WORLD", task.getMetadata("HELLO", String.class));
        task.delMetadata("HELLO");
        Assert.assertNull(task.getMetadata("HELLO", String.class));
    }

    @Test
    public void testMetadataWithNull() throws Exception {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setMetadata(Collections.singletonMap("HELLO", "WORLD"));
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        task.putMetadata("HELLO", "WORLD");
        Assert.assertEquals("WORLD", task.getMetadata("HELLO", String.class));
        task.delMetadata("HELLO");
        Assert.assertNull(task.getMetadata("HELLO", String.class));
    }

    @Test
    public void testSetProviderAndToken() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());

        task.setProviderAndToken("provider-x", "token-y");

        Assert.assertEquals("provider-x", task.getMetadata().get(ProviderRequestService.KEY_PROVIDER));
        Assert.assertEquals("provider-x", segment.getMetadata().get(ProviderRequestService.KEY_PROVIDER));
        Assert.assertEquals("token-y",
                task.getMetadata().get(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN));
        Assert.assertEquals("token-y",
                segment.getMetadata().get(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN));
    }

    @Test
    public void testGetMetadataReturnsNullWhenMetadataEmpty() throws Exception {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setMetadata(new java.util.HashMap<>());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertNull(task.getMetadata("HELLO", String.class));
    }

    @Test
    public void testGetMetadataReturnsNullWhenValueNull() throws Exception {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        java.util.HashMap<String, Object> metadata = new java.util.HashMap<>();
        metadata.put("HELLO", null);
        segment.setMetadata(metadata);
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertNull(task.getMetadata("HELLO", String.class));
    }

    @Test
    public void testGetMetadataReturnsOriginalTypeWhenAssignable() throws Exception {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        Date date = new Date();
        segment.setMetadata(Collections.singletonMap("HELLO", date));
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertSame(date, task.getMetadata("HELLO", Date.class));
    }

    @Test
    public void testGetMetadataTransfersWhenTypeMismatch() throws Exception {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setMetadata(Collections.singletonMap("CONFIG", Collections.singletonMap("provider", "PROVIDER")));
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        LLMConfig config = task.getMetadata("CONFIG", LLMConfig.class);
        Assert.assertEquals("PROVIDER", config.getProvider());
    }

    @Test
    public void testDelMetadata() throws Exception {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setMetadata(Collections.singletonMap("HELLO", "WORLD"));
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertEquals("WORLD", task.delMetadata("HELLO", String.class));
    }

    @Test
    public void testHistory() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setMetadata(Collections.singletonMap("HELLO", "WORLD"));
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertFalse(task.containHistories());
        Assert.assertTrue(task.getHistories().isEmpty());
        List<History> histories = new ArrayList<>();
        histories.add(new History());
        task.addHistories(histories);
        Assert.assertEquals(histories, task.getHistories());
        Assert.assertTrue(task.containHistories());
    }

    /**
     * addHistories：空列表直接 return；当前无 histories 时赋引用；当前有 histories 时 addAll
     */
    @Test
    public void testAddHistories() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        task.addHistories(null);
        Assert.assertTrue("addHistories(null) 不改变内部状态", task.getHistories().isEmpty());
        task.addHistories(new ArrayList<>());
        Assert.assertTrue("addHistories(empty) 不改变内部状态", task.getHistories().isEmpty());
        List<History> first = new ArrayList<>();
        History h1 = new History();
        first.add(h1);
        task.addHistories(first);
        Assert.assertSame("当前无 histories 时直接赋引用", first, task.getHistories());
        Assert.assertEquals(1, task.getHistories().size());
        List<History> second = new ArrayList<>();
        History h2 = new History();
        second.add(h2);
        task.addHistories(second);
        Assert.assertEquals("当前有 histories 时 addAll", 2, task.getHistories().size());
        Assert.assertEquals(h1, task.getHistories().get(0));
        Assert.assertEquals(h2, task.getHistories().get(1));
    }

    @Test
    public void testFunCall() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setMetadata(Collections.singletonMap("HELLO", "WORLD"));
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertFalse(task.containFunCallTrack());
        Assert.assertNull(task.getFunCallTrack());
        task.beginFunCallTrack("ABC");
        Assert.assertEquals("ABC", task.getFunCallTrack());
        task.beginFunCallTrack();
        Assert.assertEquals(Integer.valueOf(36), Integer.valueOf(task.getFunCallTrack().length()));
        task.closeFunCallTrack();
        Assert.assertNull(task.getFunCallTrack());
    }

    @Test
    public void testChat() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        segment.setMetadata(Collections.singletonMap("HELLO", "WORLD"));
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertFalse(task.containChatTrack());
        task.beginChatTrack();
        Assert.assertTrue(task.containChatTrack());
    }

    @Test
    public void testCheckClosed_notClosed_doesNotThrow() throws Exception {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        task.checkClosed();
    }

    @Test(expected = WorkflowException.class)
    public void testCheckClosed_whenClosed_throwsWorkflowException() throws Exception {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        NothingWriteBack writeBack = new NothingWriteBack();
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, writeBack);
        writeBack.close();
        task.checkClosed();
    }

    // ---------- 委托 segment 的 getter/setter 单测 ----------

    @Test
    public void testGetUserContext_delegatesToSegment() {
        Segment segment = EasyMock.createMock(Segment.class);
        EasyMock.expect(segment.getHistories()).andReturn(null).anyTimes();
        expectSegmentConstructorCalls(segment);
        UserContext ctx = UserContext.builder().device("D").build();
        EasyMock.expect(segment.getUserContext()).andReturn(ctx).atLeastOnce();
        EasyMock.replay(segment);
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertSame("getUserContext should delegate to segment", ctx, task.getUserContext());
        EasyMock.verify(segment);
    }

    @Test
    public void testGetConversation_delegatesToSegment() {
        Segment segment = EasyMock.createMock(Segment.class);
        EasyMock.expect(segment.getHistories()).andReturn(null).anyTimes();
        expectSegmentConstructorCalls(segment);
        EasyMock.expect(segment.getConversation()).andReturn("conv-1").atLeastOnce();
        EasyMock.replay(segment);
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertEquals("getConversation should delegate to segment", "conv-1", task.getConversation());
        EasyMock.verify(segment);
    }

    @Test
    public void testGetDimension_joinsBizChatDevice() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        task.setBiz("Biz");
        task.setChat("Chat");
        task.setUserContext(UserContext.builder().device("Device").build());
        Assert.assertEquals("getDimension should be biz-chat-device joined by '-'", "Biz-Chat-Device", task.getDimension());
    }

    @Test
    public void testGetProtocol_delegatesToSegment() {
        Segment segment = EasyMock.createMock(Segment.class);
        EasyMock.expect(segment.getHistories()).andReturn(null).anyTimes();
        expectSegmentConstructorCalls(segment);
        EasyMock.expect(segment.getProtocol()).andReturn("HTTPS").atLeastOnce();
        EasyMock.replay(segment);
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertEquals("getProtocol should delegate to segment", "HTTPS", task.getProtocol());
        EasyMock.verify(segment);
    }

    @Test
    public void testGetDevice_delegatesToUserContext() {
        Segment segment = EasyMock.createMock(Segment.class);
        EasyMock.expect(segment.getHistories()).andReturn(null).anyTimes();
        expectSegmentConstructorCalls(segment);
        UserContext ctx = UserContext.builder().device("PHONE").build();
        EasyMock.expect(segment.getUserContext()).andReturn(ctx).atLeastOnce();
        EasyMock.replay(segment);
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertEquals("getDevice should come from getUserContext().getDevice()", "PHONE", task.getDevice());
        EasyMock.verify(segment);
    }

    @Test
    public void testGetTrace_delegatesToSegment() {
        Segment segment = EasyMock.createMock(Segment.class);
        EasyMock.expect(segment.getHistories()).andReturn(null).anyTimes();
        expectSegmentConstructorCalls(segment);
        EasyMock.expect(segment.getTrace()).andReturn("trace-id-1").atLeastOnce();
        EasyMock.replay(segment);
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        Assert.assertEquals("getTrace should delegate to segment", "trace-id-1", task.getTrace());
        EasyMock.verify(segment);
    }

    @Test
    public void testSetUserContext_delegatesToSegment() {
        Segment segment = EasyMock.createMock(Segment.class);
        EasyMock.expect(segment.getHistories()).andReturn(null).anyTimes();
        expectSegmentConstructorCalls(segment);
        UserContext ctx = UserContext.builder().device("TAB").build();
        segment.setUserContext(ctx);
        EasyMock.expectLastCall().once();
        EasyMock.replay(segment);
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        task.setUserContext(ctx);
        EasyMock.verify(segment);
    }

    /**
     * 构造 WorkflowTaskImpl 时 Segment 必须提供的最小 stub，供 mock 类单测复用
     */
    private static void expectSegmentConstructorCalls(Segment segment) {
        EasyMock.expect(segment.getWorkflow()).andReturn("W").anyTimes();
        EasyMock.expect(segment.getBiz()).andReturn("B").anyTimes();
        EasyMock.expect(segment.getOriginal()).andReturn("O").anyTimes();
        EasyMock.expect(segment.getInitial()).andReturn("I").anyTimes();
        EasyMock.expect(segment.getContent()).andReturn("C").anyTimes();
        EasyMock.expect(segment.getChat()).andReturn("CH").anyTimes();
        EasyMock.expect(segment.copy()).andReturn(segment).once();
        segment.init();
        EasyMock.expectLastCall().once();
    }
}
