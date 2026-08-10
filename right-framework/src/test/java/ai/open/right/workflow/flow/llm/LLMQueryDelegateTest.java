package ai.open.right.workflow.flow.llm;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.UserContext;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.Notifier;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class LLMQueryDelegateTest {

    @Test
    public void testSetGetObject() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMQueryDelegate delegate = new LLMQueryDelegate(workflowTask, "WR", "NO");
        Assertions.assertEquals(workflowTask.getCreated(), delegate.getCreated());
        LLMConfig config = new LLMConfig();
        config.setProvider("PROVIDER");
        delegate.setObjectQuery(config);
        delegate.setTakeover("TK");
        config = delegate.getObjectQuery(LLMConfig.class);
        Assertions.assertEquals("PROVIDER", config.getProvider());
        Assertions.assertEquals("TK", delegate.getTakeover());
        Assertions.assertEquals(workflowTask, delegate.getWorkTask());
        List<MediaContext> mediaContexts = new ArrayList<>();
        delegate.setMediaContext(mediaContexts);
        Assertions.assertEquals(mediaContexts, delegate.getMediaContext());
    }

    @Test
    public void testAddMediaContext() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMQueryDelegate delegate = new LLMQueryDelegate(workflowTask, "WR", "NO");
        MediaContext context = new MediaContext();
        delegate.addMediaContext(context);
        Assertions.assertEquals(context, delegate.getMediaContext().get(0));
        Assertions.assertEquals(1, delegate.getMediaContext().size());
    }

    @Test
    public void testCallToXXX() {
        LLMQueryDelegate delegate = new LLMQueryDelegate(ObjectBuilder.buildWorkflowTask(), "WR", "NO");
        delegate.callToLocalHost();
        Assertions.assertEquals(Notifier.LOCALHOST, delegate.getNotifier());
        delegate.callToEndpoint();
        Assertions.assertEquals(Notifier.ENDPOINT, delegate.getNotifier());
        Assertions.assertEquals("UNKNOWN", delegate.getUpstream());
    }

    @Test
    public void testSetXXX() {
        LLMQueryDelegate delegate = new LLMQueryDelegate(ObjectBuilder.buildWorkflowTask(), "WR", "NO");
        delegate.setQuery("QUERY");
        delegate.setChat("CHAT");
        delegate.setBiz("BIZ");
        delegate.setNotifier("NOTIFIER");
        delegate.setWorkflow("WORKFLOW");
        delegate.setProtocol("PROTOCOL");
        Assertions.assertEquals("CHAT", delegate.getChat());
        Assertions.assertEquals("BIZ", delegate.getBiz());
        Assertions.assertEquals("QUERY", delegate.getQuery());
        Assertions.assertEquals("PROTOCOL", delegate.getProtocol());
        Assertions.assertEquals("NOTIFIER", delegate.getNotifier());
        Assertions.assertEquals("WORKFLOW", delegate.getWorkflow());
        UserContext userContext = UserContext.builder().device("IPHONE").build();
        delegate.setUserContext(userContext);
        Assertions.assertEquals(userContext, delegate.getUserContext());
        Assertions.assertEquals("IPHONE", delegate.getDevice());
    }

    @Test
    public void testSetDeepnessDelegatesToWorkTask() {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMQueryDelegate delegate = new LLMQueryDelegate(workflowTask, "WR", "NO");
        delegate.setDeepness(2);
        Assertions.assertEquals(Integer.valueOf(2), workflowTask.getDeepness());
        Assertions.assertEquals(Integer.valueOf(2), delegate.getDeepness());
    }

    @Test
    public void testSetHistories_delegatesToWorkTask() {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMQueryDelegate delegate = new LLMQueryDelegate(workflowTask, "WR", "NO");
        List<History> histories = new ArrayList<>();
        histories.add(new History());
        delegate.setHistories(histories);
        Assertions.assertEquals(histories, delegate.getHistories());
        Assertions.assertEquals(histories, workflowTask.getHistories());
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
        workflowTask.setWorkflow("WR");
        EasyMock.expectLastCall().anyTimes();
        workflowTask.setNotifier("NO");
        EasyMock.expectLastCall().anyTimes();
        Segment segment = EasyMock.createMock(Segment.class);
        workflowTask.writeSource(segment);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(segment, workflowTask);
        LLMQueryDelegate delegate = new LLMQueryDelegate(workflowTask, "WR", "NO");
        delegate.writeSource(segment);
        EasyMock.verify(workflowTask);
    }

    @Test
    public void testWriteBack() throws Exception {
        WorkflowTask workflowTask = EasyMock.createMock(WorkflowTask.class);
        workflowTask.setWorkflow("WR");
        EasyMock.expectLastCall().anyTimes();
        workflowTask.setNotifier("NO");
        EasyMock.expectLastCall().anyTimes();
        workflowTask.writeBack(EasyMock.anyObject());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(workflowTask);
        LLMQueryDelegate delegate = new LLMQueryDelegate(workflowTask, "WR", "NO");
        delegate.writeBack(Segment.build(ObjectBuilder.buildWorkflowTask(), Segment.SegmentConfig.builder().build()));
        EasyMock.verify(workflowTask);
    }

    @Test
    public void testIsEntryDelegatesToWorkTask() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        LLMQueryDelegate delegate = new LLMQueryDelegate(task, "WR", "NO");
        Assertions.assertEquals(task.isEntry(), delegate.isEntry());
    }

    @Test
    public void testUpStream() {
        LLMQueryDelegate delegate = new LLMQueryDelegate(ObjectBuilder.buildWorkflowTask(), "WR", "NO");
        delegate.setUpstream("UPSTREAM");
        Assertions.assertEquals("UPSTREAM", delegate.getUpstream());
    }

    @Test
    public void testMarkQueryAndResetQueryDelegateToWorkTask() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        task.setQuery("original");
        LLMQueryDelegate delegate = new LLMQueryDelegate(task, "WR", "NO");
        delegate.markQuery();
        delegate.setQuery("changed");
        Assertions.assertEquals("changed", delegate.getQuery());
        delegate.resetQuery();
        Assertions.assertEquals("original", delegate.getQuery());
    }

    @Test
    public void testDelMeta1() throws Exception {
        LLMQueryDelegate delegate = new LLMQueryDelegate(ObjectBuilder.buildWorkflowTask(), "WR", "NO");
        Assertions.assertNull(delegate.delMetadata("HELLO", String.class));
        Assertions.assertFalse(delegate.containMetadata("HELLO"));
        delegate.putMetadata("HELLO", "WORLD");
        Assertions.assertTrue(delegate.containMetadata("HELLO"));
        Assertions.assertEquals("WORLD", delegate.delMetadata("HELLO", String.class));
    }

    @Test
    public void testDelMeta2() throws Exception {
        LLMQueryDelegate delegate = new LLMQueryDelegate(ObjectBuilder.buildWorkflowTask(), "WR", "NO");
        delegate.putMetadata("HELLO", "WORLD");
        Assertions.assertEquals("WORLD", delegate.delMetadata("HELLO", String.class));
        delegate.delMetadata("HELLO");
        Assertions.assertNull(delegate.getMetadata().get("HELLO"));
    }

    @Test
    public void testSetProviderAndToken() {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMQueryDelegate delegate = new LLMQueryDelegate(workflowTask, "WR", "NO");

        delegate.setProviderAndToken("provider-x", "token-y");

        Assertions.assertEquals("provider-x", delegate.getMetadata().get(ProviderRequestService.KEY_PROVIDER));
        Assertions.assertEquals("provider-x", workflowTask.getMetadata().get(ProviderRequestService.KEY_PROVIDER));
        Assertions.assertEquals("token-y",
                delegate.getMetadata().get(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN));
        Assertions.assertEquals("token-y",
                workflowTask.getMetadata().get(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN));
    }

    @Test
    public void testHistory() {
        LLMQueryDelegate llmQueryDelegate = new LLMQueryDelegate(ObjectBuilder.buildWorkflowTask(), "WR", "NO");
        List<History> histories = new ArrayList<>();
        Assertions.assertFalse(llmQueryDelegate.containHistories());
        llmQueryDelegate.addHistories(histories);
        Assertions.assertEquals(histories, llmQueryDelegate.getHistories());
        histories.add(new History());
        // Copy而不是引用
        Assertions.assertFalse(llmQueryDelegate.containHistories());
    }

    @Test
    public void testFunCall() {
        LLMQueryDelegate task = new LLMQueryDelegate(ObjectBuilder.buildWorkflowTask(), "WR", "NO");
        Assertions.assertFalse(task.containFunCallTrack());
        Assertions.assertNull(task.getFunCallTrack());
        task.beginFunCallTrack("ABC");
        Assertions.assertEquals("ABC", task.getFunCallTrack());
        task.beginFunCallTrack();
        Assertions.assertEquals(36, task.getFunCallTrack().length());
        task.closeFunCallTrack();
        Assertions.assertNull(task.getFunCallTrack());
    }

    @Test
    public void testChat() {
        LLMQueryDelegate task = new LLMQueryDelegate(ObjectBuilder.buildWorkflowTask(), "WR", "NO");
        Assertions.assertFalse(task.containChatTrack());
        task.beginChatTrack();
        Assertions.assertTrue(task.containChatTrack());
    }

    @Test
    public void testFromFunCall1() {
        LLMQueryDelegate task = new LLMQueryDelegate(ObjectBuilder.buildWorkflowTask(), "WR", "NO");
        Assertions.assertFalse(task.isFromFunCall());
        task.putMetadata(ProviderRequestService.KEY_FUN_FETCH, new Object());
        Assertions.assertTrue(task.isFromFunCall());
    }

    @Test
    public void testFromFunCall2() {
        LLMQueryDelegate task = new LLMQueryDelegate(ObjectBuilder.buildWorkflowTask(), "WR", "NO");
        Assertions.assertFalse(task.isFromFunCall());
        task.putMetadata(ProviderRequestService.KEY_FUN_MERGE, new Object());
        Assertions.assertTrue(task.isFromFunCall());
    }

    @Test
    public void testFromFunMerge_fetchOnlyFalse_mergeTrue() {
        LLMQueryDelegate task = new LLMQueryDelegate(ObjectBuilder.buildWorkflowTask(), "WR", "NO");
        Assertions.assertFalse(task.isFromFunMerge());
        task.putMetadata(ProviderRequestService.KEY_FUN_FETCH, new Object());
        Assertions.assertFalse(task.isFromFunMerge());
        task.putMetadata(ProviderRequestService.KEY_FUN_MERGE, new Object());
        Assertions.assertTrue(task.isFromFunMerge());
    }

    @Test
    public void testDimension() {
        LLMQueryDelegate delegate = new LLMQueryDelegate(ObjectBuilder.buildWorkflowTask(), "WR", "NO");
        Assertions.assertEquals("UNKNOWN-UNKNOWN-UNKNOWN", delegate.getDimension());
    }

    @Test
    public void testGetMetadataNull() {
        WorkflowTask task = EasyMock.createMock(WorkflowTask.class);
        EasyMock.expect(task.getMetadata()).andReturn(null).anyTimes();
        task.setWorkflow(EasyMock.anyString());
        task.setNotifier(EasyMock.anyString());
        EasyMock.replay(task);
        LLMQueryDelegate delegate = new LLMQueryDelegate(task, "W", "N");
        Assertions.assertNull(delegate.getMetadata());
    }

    @Test
    public void testDelMetadataClassNull() throws Exception {
        WorkflowTask task = EasyMock.createMock(WorkflowTask.class);
        EasyMock.expect(task.getMetadata()).andReturn(null).anyTimes();
        task.setWorkflow(EasyMock.anyString());
        task.setNotifier(EasyMock.anyString());
        EasyMock.expect(task.getMetadata(EasyMock.anyString(), EasyMock.anyObject())).andReturn(null).anyTimes();
        EasyMock.expect(task.delMetadata("K", String.class)).andReturn(null).anyTimes();
        task.delMetadata("K");
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(task);
        LLMQueryDelegate delegate = new LLMQueryDelegate(task, "W", "N");
        Assertions.assertNull(delegate.delMetadata("K", String.class));
    }

    @Test
    public void testGetObjectQueryInvalid() throws Exception {
        WorkflowTask task = EasyMock.createMock(WorkflowTask.class);
        EasyMock.expect(task.getQuery()).andReturn("INVALID").anyTimes();
        EasyMock.expect(task.getObjectQuery(EasyMock.anyObject())).andThrow(new RuntimeException()).anyTimes();
        task.setWorkflow(EasyMock.anyString());
        task.setNotifier(EasyMock.anyString());
        EasyMock.replay(task);
        LLMQueryDelegate delegate = new LLMQueryDelegate(task, "W", "N");
        Assertions.assertThrows(Exception.class, () -> delegate.getObjectQuery(Map.class));
    }

    @Test
    public void testBeginFunCallTrackNull() {
        WorkflowTask task = EasyMock.createMock(WorkflowTask.class);
        task.beginFunCallTrack(null);
        EasyMock.expectLastCall().once();
        task.setWorkflow(EasyMock.anyString());
        task.setNotifier(EasyMock.anyString());
        EasyMock.replay(task);
        LLMQueryDelegate delegate = new LLMQueryDelegate(task, "W", "N");
        delegate.beginFunCallTrack(null);
        EasyMock.verify(task);
    }

    @Test
    public void testSetObjectQueryNull() throws Exception {
        WorkflowTask task = EasyMock.createMock(WorkflowTask.class);
        task.setObjectQuery(null);
        EasyMock.expectLastCall().anyTimes();
        task.setQuery(null);
        EasyMock.expectLastCall().anyTimes();
        task.setWorkflow(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        task.setNotifier(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(task);
        LLMQueryDelegate delegate = new LLMQueryDelegate(task, "W", "N");
        delegate.setObjectQuery(null);
        EasyMock.verify(task);
    }

    @Test
    public void testProxyMethods() {
        WorkflowTask task = EasyMock.createMock(WorkflowTask.class);
        EasyMock.expect(task.getConversation()).andReturn("conv");
        EasyMock.expect(task.getDeepness()).andReturn(5);
        EasyMock.expect(task.getOriginal()).andReturn("orig");
        EasyMock.expect(task.getPrevious()).andReturn("prev");
        EasyMock.expect(task.getCreated()).andReturn(123L);
        EasyMock.expect(task.getConsuming()).andReturn(456L);
        EasyMock.expect(task.getInitial()).andReturn("init");
        EasyMock.expect(task.getTrace()).andReturn("trace");
        EasyMock.expect(task.getChat()).andReturn("chat");
        task.setWorkflow("W");
        task.setNotifier("N");
        EasyMock.replay(task);

        LLMQueryDelegate delegate = new LLMQueryDelegate(task, "W", "N");
        Assertions.assertEquals("conv", delegate.getConversation());
        Assertions.assertEquals(5, delegate.getDeepness());
        Assertions.assertEquals("orig", delegate.getOriginal());
        Assertions.assertEquals("prev", delegate.getPrevious());
        Assertions.assertEquals(123L, delegate.getCreated());
        Assertions.assertEquals(456L, delegate.getConsuming());
        Assertions.assertEquals("init", delegate.getInitial());
        Assertions.assertEquals("trace", delegate.getTrace());
        Assertions.assertEquals("chat", delegate.getChat());

        EasyMock.verify(task);
    }

    @Test
    public void testIsClosed_delegatesToWorkTask() throws Exception {
        WorkflowTask task = EasyMock.createMock(WorkflowTask.class);
        task.setWorkflow("W");
        EasyMock.expectLastCall().once();
        task.setNotifier("N");
        EasyMock.expectLastCall().once();
        EasyMock.expect(task.isClosed()).andReturn(false).once();
        EasyMock.expect(task.isClosed()).andReturn(true).once();
        EasyMock.replay(task);

        LLMQueryDelegate delegate = new LLMQueryDelegate(task, "W", "N");
        Assertions.assertFalse(delegate.isClosed());
        Assertions.assertTrue(delegate.isClosed());

        EasyMock.verify(task);
    }

    @Test
    public void testClose_delegatesToWorkTask() throws Exception {
        WorkflowTask task = EasyMock.createMock(WorkflowTask.class);
        task.setWorkflow("W");
        EasyMock.expectLastCall().once();
        task.setNotifier("N");
        EasyMock.expectLastCall().once();
        task.close();
        EasyMock.expectLastCall().once();
        EasyMock.replay(task);

        LLMQueryDelegate delegate = new LLMQueryDelegate(task, "W", "N");
        delegate.close();

        EasyMock.verify(task);
    }

    @Test
    public void testIgnoreClosed_delegatesToWorkTask() throws Exception {
        WorkflowTask task = EasyMock.createMock(WorkflowTask.class);
        task.setWorkflow("W");
        EasyMock.expectLastCall().once();
        task.setNotifier("N");
        EasyMock.expectLastCall().once();
        task.ignoreClosed();
        EasyMock.expectLastCall().once();
        EasyMock.replay(task);

        LLMQueryDelegate delegate = new LLMQueryDelegate(task, "W", "N");
        delegate.ignoreClosed();

        EasyMock.verify(task);
    }

    @Test
    public void testIncrDeepness_delegatesToWorkTask() {
        WorkflowTask task = EasyMock.createMock(WorkflowTask.class);
        task.setWorkflow("W");
        EasyMock.expectLastCall().once();
        task.setNotifier("N");
        EasyMock.expectLastCall().once();
        EasyMock.expect(task.incrDeepness()).andReturn(task).anyTimes();
        EasyMock.replay(task);

        LLMQueryDelegate delegate = new LLMQueryDelegate(task, "W", "N");
        delegate.incrDeepness();

        EasyMock.verify(task);
    }
}
