package ai.open.right.workflow.flow;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.integration.RightConfig;
import ai.open.right.integration.RightTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.NothingWriteBack;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;

/**
 * WorkflowTaskWrap 全面单测：构造、getWorkTask、以及所有接口方法的委托。
 */
public class WorkflowTaskWrapTest {

    @Test
    public void constructor_holdsDelegate() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);
        Assert.assertSame(task, wrap.getWorkTask());
    }

    @Test
    public void constructor_defaultCloseableIsTrue() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        Assert.assertTrue(new WorkflowTaskWrap(task).getCloseable());
        Assert.assertTrue(new WorkflowTaskWrap(task, true).getCloseable());
    }

    @Test
    public void constructor_closeableFalse_storesFalse() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        Assert.assertFalse(new WorkflowTaskWrap(task, false).getCloseable());
    }

    /**
     * closeable=false：close 不委托，底层保持未关闭
     */
    @Test
    public void closeableFalse_close_doesNotCloseUnderlying() throws Exception {
        NothingWriteBack nwb = new NothingWriteBack();
        WorkflowTask task = new RightTask(
                RightConfig.builder().query("Q").biz("B").trace("T").chat("C").timeout(10000)
                        .conversation("C").upstream("U").notifier("N").protocol("P").metadata(new HashMap<String, Object>()).workflow("W").build().init(),
                nwb);
        ((RightTask) task).init();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task, false);
        wrap.close();
        Assert.assertFalse(nwb.isClosed());
        Assert.assertFalse(task.isClosed());
    }

    /**
     * closeable=false：底层已关闭时 isClosed 仍为 false（包装层不反映底层关闭状态）
     */
    @Test
    public void closeableFalse_isClosed_alwaysFalseEvenWhenUnderlyingClosed() throws Exception {
        NothingWriteBack nwb = new NothingWriteBack();
        WorkflowTask task = new RightTask(
                RightConfig.builder().query("Q").biz("B").trace("T").chat("C").timeout(10000)
                        .conversation("C").upstream("U").notifier("N").protocol("P").metadata(new HashMap<String, Object>()).workflow("W").build().init(),
                nwb);
        ((RightTask) task).init();
        task.close();
        Assert.assertTrue(task.isClosed());
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task, false);
        Assert.assertFalse(wrap.isClosed());
    }

    /**
     * closeable=false：checkClosed 不委托，底层已关闭也不抛异常
     */
    @Test
    public void closeableFalse_checkClosed_skipsDelegateWhenUnderlyingClosed() throws Exception {
        NothingWriteBack nwb = new NothingWriteBack();
        WorkflowTask task = new RightTask(
                RightConfig.builder().query("Q").biz("B").trace("T").chat("C").timeout(10000)
                        .conversation("C").upstream("U").notifier("N").protocol("P").metadata(new HashMap<String, Object>()).workflow("W").build().init(),
                nwb);
        ((RightTask) task).init();
        task.close();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task, false);
        wrap.checkClosed();
    }

    /**
     * closeable=true：底层已关闭时 checkClosed 委托并抛出
     */
    @Test(expected = WorkflowException.class)
    public void closeableTrue_checkClosed_delegatesWhenUnderlyingClosed() throws Exception {
        NothingWriteBack nwb = new NothingWriteBack();
        WorkflowTask task = new RightTask(
                RightConfig.builder().query("Q").biz("B").trace("T").chat("C").timeout(10000)
                        .conversation("C").upstream("U").notifier("N").protocol("P").metadata(new HashMap<String, Object>()).workflow("W").build().init(),
                nwb);
        ((RightTask) task).init();
        task.close();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task, true);
        wrap.checkClosed();
    }

    @Test
    public void workflowTask_getters_delegate() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);
        Assert.assertEquals(task.getMediaContext(), wrap.getMediaContext());
        Assert.assertEquals(task.getMetadata(), wrap.getMetadata());
        Assert.assertSame(task.getUserContext(), wrap.getUserContext());
        Assert.assertEquals(task.getHistories(), wrap.getHistories());
        Assert.assertEquals(task.getConversation(), wrap.getConversation());
        Assert.assertEquals(task.getNotifier(), wrap.getNotifier());
        Assert.assertEquals(task.getProtocol(), wrap.getProtocol());
        Assert.assertEquals(task.getWorkflow(), wrap.getWorkflow());
        Assert.assertEquals(task.getUpstream(), wrap.getUpstream());
        Assert.assertEquals(task.getCreated(), wrap.getCreated());
        Assert.assertEquals(task.getConsuming(), wrap.getConsuming());
        // getCreated 为 wrap 自身创建时间，不委托给 task
        Assert.assertNotNull(wrap.getCreated());
        Assert.assertEquals(Long.class, wrap.getCreated().getClass());
        Assert.assertTrue("getCreated should be around current time", Math.abs(wrap.getCreated() - System.currentTimeMillis()) < 2000L);
        Assert.assertEquals(task.getTrace(), wrap.getTrace());
        Assert.assertEquals(task.getQuery(), wrap.getQuery());
        Assert.assertEquals(task.getChat(), wrap.getChat());
        Assert.assertEquals(task.getBiz(), wrap.getBiz());
    }

    @Test
    public void workflowTask_setters_delegate() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);
        List<MediaContext> mediaContext = new ArrayList<>();
        wrap.setMediaContext(mediaContext);
        Assert.assertSame(mediaContext, task.getMediaContext());
        MediaContext ctx = new MediaContext();
        wrap.addMediaContext(ctx);
        Assert.assertTrue(task.getMediaContext().contains(ctx));
        ai.open.right.context.UserContext uc = ai.open.right.context.UserContext.builder().build();
        wrap.setUserContext(uc);
        Assert.assertSame(uc, task.getUserContext());
        List<History> histories = new ArrayList<>();
        wrap.addHistories(histories);
        Assert.assertNotSame(histories, task.getHistories());
        wrap.setWorkflow("WF");
        Assert.assertEquals("WF", task.getWorkflow());
        wrap.setNotifier("NF");
        Assert.assertEquals("NF", task.getNotifier());
        wrap.setUpstream("UP");
        Assert.assertEquals("UP", task.getUpstream());
        wrap.setProtocol("PR");
        Assert.assertEquals("PR", task.getProtocol());
        wrap.setQuery("QR");
        Assert.assertEquals("QR", task.getQuery());
        wrap.setChat("CH");
        Assert.assertEquals("CH", task.getChat());
        wrap.setBiz("BZ");
        Assert.assertEquals("BZ", task.getBiz());
    }

    @Test
    public void metadata_delegate() throws Exception {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);
        Assert.assertNull(wrap.getMetadata("K", String.class));
        wrap.putMetadata("K", "V");
        Assert.assertEquals("V", wrap.getMetadata("K", String.class));
        Assert.assertEquals("V", wrap.delMetadata("K", String.class));
        Assert.assertNull(wrap.getMetadata("K", String.class));
        wrap.putMetadata("K2", 100);
        Assert.assertTrue(wrap.containMetadata("K2"));
        wrap.delMetadata("K2");
        Assert.assertFalse(wrap.containMetadata("K2"));
    }

    @Test
    public void setProviderAndToken_writesIntoWrappedTaskMetadata() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);

        wrap.setProviderAndToken("provider-x", "token-y");

        Assert.assertEquals("provider-x", wrap.getMetadata().get(ProviderRequestService.KEY_PROVIDER));
        Assert.assertEquals("provider-x", task.getMetadata().get(ProviderRequestService.KEY_PROVIDER));
        Assert.assertEquals("token-y",
                wrap.getMetadata().get(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN));
        Assert.assertEquals("token-y",
                task.getMetadata().get(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN));
    }

    @Test
    public void containHistories_and_isFromFunCall_delegate() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);
        Assert.assertEquals(task.containHistories(), wrap.containHistories());
        Assert.assertEquals(task.isFromFunCall(), wrap.isFromFunCall());
    }

    @Test
    public void isFromFunMerge_delegatesToWorkTask() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);
        Assert.assertEquals(task.isFromFunMerge(), wrap.isFromFunMerge());
        Assert.assertFalse(wrap.isFromFunMerge());
        task.putMetadata(ProviderRequestService.KEY_FUN_MERGE, new Object());
        Assert.assertTrue(task.isFromFunMerge());
        Assert.assertTrue(wrap.isFromFunMerge());
    }

    @Test
    public void notifierTrack_delegate() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);
        Assert.assertFalse(wrap.containFunCallTrack());
        Assert.assertNull(wrap.getFunCallTrack());
        wrap.beginFunCallTrack("TRACK");
        Assert.assertTrue(wrap.containFunCallTrack());
        Assert.assertEquals("TRACK", wrap.getFunCallTrack());
        wrap.closeFunCallTrack();
        Assert.assertFalse(wrap.containFunCallTrack());
        Assert.assertNull(wrap.getFunCallTrack());
        wrap.beginFunCallTrack();
        Assert.assertTrue(wrap.containFunCallTrack());
        Assert.assertNotNull(wrap.getFunCallTrack());
        Assert.assertFalse(wrap.containChatTrack());
        wrap.beginChatTrack();
        Assert.assertTrue(wrap.containChatTrack());
    }

    @Test
    public void notifierWriteBack_delegate() throws Exception {
        NothingWriteBack nwb = new NothingWriteBack();
        WorkflowTask task = new RightTask(
                RightConfig.builder().query("Q").biz("B").trace("T").chat("C").timeout(10000)
                        .conversation("C").upstream("U").notifier("N").protocol("P").metadata(new HashMap<String, Object>()).workflow("W").build().init(),
                nwb);
        ((RightTask) task).init();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);
        Assert.assertEquals(task.getTakeover(), wrap.getTakeover());
        wrap.setTakeover("TK");
        Assert.assertEquals("TK", task.getTakeover());
        Segment segment = ObjectBuilder.buildSegment();
        wrap.writeSource(segment);
        wrap.writeBack(segment);
        wrap.checkClosed();
    }

    @Test
    public void notifierWriteBack_isClosed_and_close_delegate() throws Exception {
        NothingWriteBack nwb = new NothingWriteBack();
        WorkflowTask task = new RightTask(
                RightConfig.builder().query("Q").biz("B").trace("T").chat("C").timeout(10000)
                        .conversation("C").upstream("U").notifier("N").protocol("P").metadata(new HashMap<String, Object>()).workflow("W").build().init(),
                nwb);
        ((RightTask) task).init();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);
        Assert.assertFalse(wrap.isClosed());
        Assert.assertEquals(task.isClosed(), wrap.isClosed());
        wrap.close();
        Assert.assertTrue(nwb.isClosed());
        Assert.assertTrue(wrap.isClosed());
    }

    @Test
    public void notifierWriteBack_ignoreClosed_delegatesToInnerTask() throws Exception {
        NothingWriteBack nwb = new NothingWriteBack();
        WorkflowTask task = new RightTask(
                RightConfig.builder().query("Q").biz("B").trace("T").chat("C").timeout(10000)
                        .conversation("C").upstream("U").notifier("N").protocol("P").metadata(new HashMap<String, Object>()).workflow("W").build().init(),
                nwb);
        ((RightTask) task).init();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);
        Assert.assertFalse(nwb.getIgnoreClosed());
        wrap.ignoreClosed();
        Assert.assertTrue(nwb.getIgnoreClosed());
    }

    @Test
    public void redirectContext_delegate() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);
        Assert.assertEquals(task.getDeepness(), wrap.getDeepness());
        Assert.assertEquals(task.getOriginal(), wrap.getOriginal());
        Assert.assertEquals(task.getPrevious(), wrap.getPrevious());
        Assert.assertEquals(task.getInitial(), wrap.getInitial());
    }

    @Test
    public void setDeepness_delegatesToWorkTask() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);
        wrap.setDeepness(2);
        Assert.assertEquals("setDeepness should delegate to workTask", Integer.valueOf(2), task.getDeepness());
        wrap.setDeepness(5);
        Assert.assertEquals(Integer.valueOf(5), wrap.getDeepness());
    }

    @Test
    public void setHistories_delegatesToWorkTask() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);
        List<History> histories = new ArrayList<>();
        histories.add(new History());
        wrap.setHistories(histories);
        Assert.assertEquals(histories, wrap.getHistories());
        Assert.assertEquals(histories, task.getHistories());
    }

    @Test
    public void isEntry_delegatesToWorkTask() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);
        Assert.assertEquals(task.isEntry(), wrap.isEntry());
    }

    @Test
    public void incrDeepness_delegatesToWorkTask() {
        RightTask task = new RightTask(
                RightConfig.builder().query("Q").biz("B").trace("T").chat("C").timeout(10000)
                        .conversation("C").upstream("U").notifier("N").protocol("P").metadata(new HashMap<String, Object>()).workflow("W").build().init(),
                new NothingWriteBack());
        task.init();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);
        Assert.assertEquals(null, wrap.getDeepness());
        wrap.incrDeepness();
        Assert.assertEquals(Integer.valueOf(1), wrap.getDeepness());
        wrap.incrDeepness();
        Assert.assertEquals(Integer.valueOf(2), task.getDeepness());
    }

    @Test
    public void workflowObject_delegate() throws Exception {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);
        LLMConfig config = new LLMConfig();
        config.setProvider("P");
        wrap.setObjectQuery(config);
        LLMConfig got = wrap.getObjectQuery(LLMConfig.class);
        Assert.assertEquals("P", got.getProvider());
    }

    @Test
    public void dimension_delegate() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);
        Assert.assertEquals(task.getDevice(), wrap.getDevice());
        Assert.assertEquals(task.getDimension(), wrap.getDimension());
    }

    @Test
    public void wrap_identity_multipleCalls() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);
        Assert.assertSame(wrap.getWorkTask(), wrap.getWorkTask());
        Assert.assertEquals(wrap.getQuery(), wrap.getQuery());
        Assert.assertEquals(wrap.getBiz(), wrap.getBiz());
    }

    /**
     * markQuery / resetQuery 委托给内部 workTask
     */
    @Test
    public void markQueryAndResetQuery_delegateToWorkTask() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        task.setQuery("original");
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);
        wrap.markQuery();
        wrap.setQuery("changed");
        Assert.assertEquals("changed", wrap.getQuery());
        wrap.resetQuery();
        Assert.assertEquals("original", wrap.getQuery());
    }

    @Test
    public void emptyQuery_delegatesToWorkTask() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(task);

        Assert.assertSame(wrap, wrap.emptyQuery());
        Assert.assertNull(task.getQuery());
    }
}
