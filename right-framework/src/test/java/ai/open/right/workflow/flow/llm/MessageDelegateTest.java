package ai.open.right.workflow.flow.llm;
import java.util.Map;
import ai.open.right.ObjectBuilder;
import ai.open.right.context.UserContext;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.Notifier;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;

public class MessageDelegateTest {

    @Test
    public void testSetGetObject() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        MessageDelegate delegate = new MessageDelegate(llmQuery);
        Assert.assertEquals(delegate.getCreated(), llmQuery.getCreated());
        LLMConfig config = new LLMConfig();
        config.setProvider("PROVIDER");
        delegate.setTakeover("TK");
        delegate.setObjectQuery(config);
        config = delegate.getObjectQuery(LLMConfig.class);
        Assert.assertEquals("TK", delegate.getTakeover());
        Assert.assertEquals("PROVIDER", config.getProvider());
        Assert.assertEquals(llmQuery, delegate.getLlmQuery());
        List<MediaContext> mediaContexts = new ArrayList<>();
        delegate.setMediaContext(mediaContexts);
        Assert.assertEquals(mediaContexts, delegate.getMediaContext());
    }

    @Test
    public void testAddMediaContext() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMQueryDelegate llmQuery = new LLMQueryDelegate(workflowTask, "WR", "NO");
        MessageDelegate delegate = new MessageDelegate(llmQuery);
        MediaContext context = new MediaContext();
        delegate.addMediaContext(context);
        Assert.assertEquals(context, delegate.getMediaContext().getFirst());
        Assert.assertEquals(Integer.valueOf(delegate.getMediaContext().size()), Integer.valueOf(1));
    }

    @Test
    public void testSetXxx() {
        MessageDelegate delegate = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        Assert.assertNotNull(delegate.getMetadata());
        delegate.setUpstream("UPSTREAM");
        Assert.assertEquals(Integer.valueOf(0), delegate.getDeepness());
        Assert.assertEquals("UNKNOWN", delegate.getTrace());
        Assert.assertEquals("chat", delegate.getProtocol());
        Assert.assertEquals("ORIGINAL", delegate.getOriginal());
        delegate.callToEndpoint();
        Assert.assertEquals(Notifier.ENDPOINT, delegate.getNotifier());
        delegate.callToLocalHost();
        Assert.assertEquals(Notifier.LOCALHOST, delegate.getNotifier());
        delegate.setQuery("QU");
        delegate.setChat("CHAT_VAL");
        delegate.setBiz("BIZ");
        delegate.setProtocol("PR");
        delegate.setWorkflow("WR");
        delegate.setNotifier("NO");
        Assert.assertEquals("CHAT_VAL", delegate.getChat());
        Assert.assertEquals("BIZ", delegate.getBiz());
        Assert.assertEquals("UPSTREAM", delegate.getUpstream());
        Assert.assertEquals("QU", delegate.getQuery());
        Assert.assertEquals("PR", delegate.getProtocol());
        Assert.assertEquals("NO", delegate.getNotifier());
        Assert.assertEquals("WR", delegate.getWorkflow());
        Assert.assertEquals("BIZ-CHAT_VAL-UNKNOWN", delegate.getDimension());
        UserContext userContext = UserContext.builder().build();
        delegate.setUserContext(userContext);
        Assert.assertEquals(userContext, delegate.getUserContext());
    }

    @Test
    public void testIsEntryDelegatesToLlmQuery() {
        ai.open.right.workflow.flow.llm.LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        MessageDelegate delegate = new MessageDelegate(llmQuery);
        Assert.assertEquals(llmQuery.isEntry(), delegate.isEntry());
    }

    @Test
    public void testSetDeepnessDelegatesToLlmQuery() {
        ai.open.right.workflow.flow.llm.LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        MessageDelegate delegate = new MessageDelegate(llmQuery);
        delegate.setDeepness(2);
        Assert.assertEquals("setDeepness should delegate to llmQuery", Integer.valueOf(2), llmQuery.getDeepness());
        Assert.assertEquals(Integer.valueOf(2), delegate.getDeepness());
    }

    @Test
    public void testMarkQueryAndResetQueryDelegateToLlmQuery() {
        ai.open.right.workflow.flow.llm.LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        llmQuery.setQuery("original");
        MessageDelegate delegate = new MessageDelegate(llmQuery);
        delegate.markQuery();
        delegate.setQuery("changed");
        Assert.assertEquals("changed", delegate.getQuery());
        delegate.resetQuery();
        Assert.assertEquals("original", delegate.getQuery());
    }

    @Test
    public void emptyQuery_delegatesToLlmQuery() {
        ai.open.right.workflow.flow.llm.LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        MessageDelegate delegate = new MessageDelegate(llmQuery);

        Assert.assertSame(delegate, delegate.emptyQuery());
        Assert.assertNull(llmQuery.getQuery());
    }

    @Test
    public void testSetHistoryWithEmpty() {
        MessageDelegate delegate = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        delegate.addHistories(new ArrayList<>());
        Assert.assertTrue(delegate.getHistories().isEmpty());
        delegate.addHistories(null);
        Assert.assertTrue(delegate.getHistories().isEmpty());
    }

    @Test
    public void testSetHistory() {
        MessageDelegate delegate = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        List<History> history = new ArrayList<History>();
        History h1 = new History();
        history.add(h1);
        delegate.addHistories(history);
        Assert.assertTrue(delegate.hasHistory());
        List<History> history2 = new ArrayList<History>();
        History h2 = new History();
        history2.add(h2);
        delegate.addHistories(history2);
        Assert.assertEquals(2, delegate.getHistories().size());
        Assert.assertEquals(h1, delegate.getHistories().get(0));
        Assert.assertEquals(h2, delegate.getHistories().get(1));
        Assert.assertEquals(delegate.getHistories(), delegate.getMessages());
    }

    @Test
    public void testAddHistory() {
        MessageDelegate delegate = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        History h1 = new History();
        delegate.addHistory(h1);
        Assert.assertEquals(1, delegate.getHistories().size());
        History h2 = new History();
        delegate.addHistory(h2);
        Assert.assertEquals(2, delegate.getHistories().size());
        Assert.assertEquals(h1, delegate.getHistories().get(0));
        Assert.assertEquals(h2, delegate.getHistories().get(1));
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
        EasyMock.expect(workflowTask.getBiz()).andReturn("BIZ").anyTimes();
        EasyMock.expect(workflowTask.getUserContext()).andReturn(ObjectBuilder.buildEmpty()).anyTimes();
        workflowTask.setWorkflow("WORKFLOW");
        EasyMock.expectLastCall().anyTimes();
        workflowTask.setNotifier("endpoint");
        EasyMock.expectLastCall().anyTimes();
        Segment segment = EasyMock.createMock(Segment.class);
        workflowTask.writeSource(segment);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(workflowTask.getHistories()).andReturn(new ArrayList<>()).anyTimes();
        EasyMock.replay(segment, workflowTask);
        MessageDelegate message = new MessageDelegate(LLMQuery.build(workflowTask));
        message.writeSource(segment);
        EasyMock.verify(segment, workflowTask);
    }

    @Test
    public void testWriteBack() throws Exception {
        WorkflowTask workflowTask = EasyMock.createMock(WorkflowTask.class);
        EasyMock.expect(workflowTask.getHistories()).andReturn(new ArrayList<>()).anyTimes();
        workflowTask.setWorkflow("WR");
        EasyMock.expectLastCall().anyTimes();
        workflowTask.setNotifier("endpoint");
        EasyMock.expectLastCall().anyTimes();
        workflowTask.writeBack(EasyMock.anyObject());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(workflowTask.getWorkflow()).andReturn("WR").anyTimes();
        EasyMock.replay(workflowTask);
        MessageDelegate delegate = new MessageDelegate(LLMQuery.build(workflowTask));
        delegate.writeBack(Segment.build(ObjectBuilder.buildWorkflowTask(), Segment.SegmentConfig.builder().build()));
        EasyMock.verify(workflowTask);
    }

    @Test
    public void testGetMeta1() throws Exception {
        LLMQueryDelegate delegate = new LLMQueryDelegate(ObjectBuilder.buildWorkflowTask(), "WR", "NO");
        Assert.assertNull(delegate.getMetadata("HELLO", String.class));
        delegate.putMetadata("HELLO", "WORLD");
        Assert.assertEquals("WORLD", delegate.getMetadata("HELLO", String.class));
    }

    @Test
    public void testGetMeta2() throws Exception {
        MessageDelegate delegate = new MessageDelegate(new LLMQueryDelegate(ObjectBuilder.buildWorkflowTask(), "WR", "NO"));
        Assert.assertNull(delegate.getMetadata("HELLO", String.class));
        delegate.putMetadata("HELLO", "WORLD");
        Assert.assertEquals("WORLD", delegate.getMetadata("HELLO", String.class));
    }

    @Test
    public void testDelMeta1() throws Exception {
        MessageDelegate delegate = new MessageDelegate(new LLMQueryDelegate(ObjectBuilder.buildWorkflowTask(), "WR", "NO"));
        Assert.assertFalse(delegate.containMetadata("HELLO"));
        delegate.putMetadata("HELLO", "WORLD");
        Assert.assertTrue(delegate.containMetadata("HELLO"));
        Assert.assertEquals("WORLD", delegate.getMetadata("HELLO", String.class));
        Assert.assertEquals("WORLD", delegate.delMetadata("HELLO", String.class));
        Assert.assertNull(delegate.getMetadata("HELLO", String.class));
    }

    @Test
    public void testDelMeta2() throws Exception {
        MessageDelegate delegate = new MessageDelegate(new LLMQueryDelegate(ObjectBuilder.buildWorkflowTask(), "WR", "NO"));
        delegate.putMetadata("HELLO", "WORLD");
        Assert.assertEquals("WORLD", delegate.getMetadata("HELLO", String.class));
        delegate.delMetadata("HELLO");
        Assert.assertNull(delegate.getMetadata("HELLO", String.class));
    }

    @Test
    public void testSetProviderAndToken() {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        MessageDelegate delegate = new MessageDelegate(llmQuery);

        delegate.setProviderAndToken("provider-x", "token-y");

        Assert.assertEquals("provider-x", delegate.getMetadata().get(ProviderRequestService.KEY_PROVIDER));
        Assert.assertEquals("provider-x", llmQuery.getMetadata().get(ProviderRequestService.KEY_PROVIDER));
        Assert.assertEquals("token-y",
                delegate.getMetadata().get(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN));
        Assert.assertEquals("token-y",
                llmQuery.getMetadata().get(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN));
    }

    @Test
    public void testHistory() {
        LLMQueryDelegate llmQueryDelegate = new LLMQueryDelegate(ObjectBuilder.buildWorkflowTask(), "WR", "NO");
        List<History> histories = new ArrayList<>();
        histories.add(new History());
        llmQueryDelegate.addHistories(histories);
        Message message = new MessageDelegate(llmQueryDelegate);
        Assert.assertEquals(histories, message.getHistories());
    }

    @Test
    public void testFunCall() {
        Message task = new MessageDelegate(ObjectBuilder.buildLLMQuery());
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
        Message task = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        Assert.assertFalse(task.containChatTrack());
        task.beginChatTrack();
        Assert.assertTrue(task.containChatTrack());
    }

    @Test
    public void testFromFunCall1() {
        Message task = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        Assert.assertFalse(task.isFromFunCall());
        task.putMetadata(ProviderRequestService.KEY_FUN_FETCH, new Object());
        Assert.assertTrue(task.isFromFunCall());
    }

    @Test
    public void testFromFunCall2() {
        Message task = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        Assert.assertFalse(task.isFromFunCall());
        task.putMetadata(ProviderRequestService.KEY_FUN_MERGE, new Object());
        Assert.assertTrue(task.isFromFunCall());
    }

    @Test
    public void testFromFunMerge_fetchOnlyFalse_mergeTrue() {
        Message task = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        Assert.assertFalse(task.isFromFunMerge());
        task.putMetadata(ProviderRequestService.KEY_FUN_FETCH, new Object());
        Assert.assertFalse(task.isFromFunMerge());
        task.putMetadata(ProviderRequestService.KEY_FUN_MERGE, new Object());
        Assert.assertTrue(task.isFromFunMerge());
    }

    @Test
    public void testReplaceHistories() {
        MessageDelegate delegate = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        List<History> histories = new ArrayList<>();
        histories.add(new History());
        delegate.replaceHistories(histories);
        Assert.assertEquals(histories, delegate.getHistories());
        delegate.replaceHistories(null);
        Assert.assertTrue(delegate.getHistories().isEmpty());
    }

    /** setHistories 非 null 时存为副本；null 时置为 null，后续 getHistories() 返回空列表 */
    @Test
    public void testSetHistories() {
        MessageDelegate delegate = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        List<History> histories = new ArrayList<>();
        History h = new History();
        histories.add(h);
        delegate.setHistories(histories);
        Assert.assertEquals(1, delegate.getHistories().size());
        Assert.assertEquals(h, delegate.getHistories().get(0));
        Assert.assertNotSame("setHistories 应存副本", histories, delegate.getHistories());
        histories.add(new History());
        Assert.assertEquals("修改原列表不应影响 delegate 内部", 1, delegate.getHistories().size());
        delegate.setHistories(null);
        Assert.assertTrue("setHistories(null) 后 getHistories() 返回空列表", delegate.getHistories().isEmpty());
    }

    @Test
    public void testDelHistoriesNull() {
        MessageDelegate delegate = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        delegate.replaceHistories(null);
        delegate.delHistories(); // Should not throw exception
        Assert.assertTrue(delegate.getHistories().isEmpty());
    }

    @Test
    public void testSetHistoriesAppend() {
        MessageDelegate delegate = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        History h1 = new History();
        delegate.addHistory(h1);
        List<History> hList = new ArrayList<>();
        History h2 = new History();
        hList.add(h2);
        delegate.addHistories(hList);
        Assert.assertEquals(2, delegate.getHistories().size());
        Assert.assertEquals(h1, delegate.getHistories().get(0));
        Assert.assertEquals(h2, delegate.getHistories().get(1));
    }

    @Test
    public void testAddHistoryInit() {
        MessageDelegate delegate = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        delegate.replaceHistories(null);
        History h1 = new History();
        delegate.addHistory(h1);
        Assert.assertNotNull(delegate.getHistories());
        Assert.assertEquals(1, delegate.getHistories().size());
    }

    @Test(expected = Exception.class)
    public void testGetObjectQueryInvalid() throws Exception {
        LLMQuery llmQuery = EasyMock.createMock(LLMQuery.class);
        EasyMock.expect(llmQuery.getQuery()).andReturn("INVALID_JSON").anyTimes();
        EasyMock.expect(llmQuery.containHistories()).andReturn(false).anyTimes();
        EasyMock.expect(llmQuery.getObjectQuery(EasyMock.anyObject())).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(llmQuery);
        MessageDelegate delegate = new MessageDelegate(llmQuery);
        delegate.getObjectQuery(Map.class);
    }

    @Test
    public void testIsClosed_delegatesToLlmQuery() throws Exception {
        LLMQuery llmQuery = EasyMock.createMock(LLMQuery.class);
        EasyMock.expect(llmQuery.containHistories()).andReturn(false).anyTimes();
        EasyMock.expect(llmQuery.isClosed()).andReturn(false).once();
        EasyMock.expect(llmQuery.isClosed()).andReturn(true).once();
        EasyMock.replay(llmQuery);

        MessageDelegate delegate = new MessageDelegate(llmQuery);
        Assert.assertFalse(delegate.isClosed());
        Assert.assertTrue(delegate.isClosed());

        EasyMock.verify(llmQuery);
    }

    @Test
    public void testClose_delegatesToLlmQuery() throws Exception {
        LLMQuery llmQuery = EasyMock.createMock(LLMQuery.class);
        EasyMock.expect(llmQuery.containHistories()).andReturn(false).anyTimes();
        llmQuery.close();
        EasyMock.expectLastCall().once();
        EasyMock.replay(llmQuery);

        MessageDelegate delegate = new MessageDelegate(llmQuery);
        delegate.close();

        EasyMock.verify(llmQuery);
    }

    @Test
    public void testIgnoreClosed_delegatesToLlmQuery() throws Exception {
        LLMQuery llmQuery = EasyMock.createMock(LLMQuery.class);
        EasyMock.expect(llmQuery.containHistories()).andReturn(false).anyTimes();
        llmQuery.ignoreClosed();
        EasyMock.expectLastCall().once();
        EasyMock.replay(llmQuery);

        MessageDelegate delegate = new MessageDelegate(llmQuery);
        delegate.ignoreClosed();

        EasyMock.verify(llmQuery);
    }

    @Test
    public void testCheckClosed_delegatesToLlmQuery() throws Exception {
        LLMQuery llmQuery = EasyMock.createMock(LLMQuery.class);
        EasyMock.expect(llmQuery.containHistories()).andReturn(false).anyTimes();
        llmQuery.checkClosed();
        EasyMock.expectLastCall().once();
        EasyMock.replay(llmQuery);

        MessageDelegate delegate = new MessageDelegate(llmQuery);
        delegate.checkClosed();

        EasyMock.verify(llmQuery);
    }

    @Test(expected = Exception.class)
    public void testCheckClosed_propagatesExceptionFromLlmQuery() throws Exception {
        LLMQuery llmQuery = EasyMock.createMock(LLMQuery.class);
        EasyMock.expect(llmQuery.containHistories()).andReturn(false).anyTimes();
        llmQuery.checkClosed();
        EasyMock.expectLastCall().andThrow(new Exception("closed"));
        EasyMock.replay(llmQuery);

        MessageDelegate delegate = new MessageDelegate(llmQuery);
        delegate.checkClosed();
    }

    @Test
    public void testIncrDeepness_delegatesToLlmQuery() {
        LLMQuery llmQuery = EasyMock.createMock(LLMQuery.class);
        EasyMock.expect(llmQuery.containHistories()).andReturn(false).anyTimes();
        EasyMock.expect(llmQuery.incrDeepness()).andReturn(llmQuery).anyTimes();
        EasyMock.replay(llmQuery);

        MessageDelegate delegate = new MessageDelegate(llmQuery);
        MessageDelegate result = delegate.incrDeepness();

        Assert.assertSame("incrDeepness() 应返回 this 以支持链式调用", delegate, result);
        EasyMock.verify(llmQuery);
    }
}
