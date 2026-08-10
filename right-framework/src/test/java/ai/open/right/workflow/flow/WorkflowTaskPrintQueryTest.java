package ai.open.right.workflow.flow;

import ai.open.right.ObjectBuilder;
import ai.open.right.integration.RightConfig;
import ai.open.right.integration.RightTask;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.LLMQueryDelegate;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.SegmentDelegate;
import ai.open.right.workflow.notify.NothingWriteBack;
import ai.open.right.workflow.notify.impl.LocalhostNotifier;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;

/**
 * 覆盖各 {@link WorkflowTask} 实现类及 {@link LLMQuery} / {@link ai.open.right.workflow.flow.llm.Message} 上的 {@code printQuery()}（返回 this、且不改 query）。
 */
public class WorkflowTaskPrintQueryTest {

    @Test
    public void nettyRequest_printQuery_returnsThis() {
        NettyRequest req = (NettyRequest) ObjectBuilder.buildWorkflowTask();
        req.setQuery("q-netty");
        Assert.assertSame(req, req.printQuery());
        Assert.assertEquals("q-netty", req.getQuery());
    }

    @Test
    public void llmQueryDelegate_printQuery_returnsThis() {
        LLMQuery q = ObjectBuilder.buildLLMQuery();
        q.setQuery("q-llm");
        Assert.assertTrue(q instanceof LLMQueryDelegate);
        Assert.assertSame(q, q.printQuery());
        Assert.assertEquals("q-llm", q.getQuery());
    }

    @Test
    public void workflowTaskWrap_printQuery_returnsThis() {
        NettyRequest inner = (NettyRequest) ObjectBuilder.buildWorkflowTask();
        inner.setQuery("q-wrap");
        WorkflowTaskWrap wrap = new WorkflowTaskWrap(inner);
        Assert.assertSame(wrap, wrap.printQuery());
        Assert.assertEquals("q-wrap", wrap.getQuery());
    }

    @Test
    public void syncWorkflowTask_printQuery_returnsThis() {
        NothingWriteBack nwb = new NothingWriteBack();
        RightTask rightTask = new RightTask(
                RightConfig.builder()
                        .query("q-sync")
                        .biz("B")
                        .trace("T")
                        .chat("C")
                        .timeout(10000)
                        .conversation("CO")
                        .upstream("U")
                        .notifier("N")
                        .protocol("P")
                        .metadata(new HashMap<>())
                        .workflow("W")
                        .build()
                        .init(),
                nwb);
        rightTask.init();
        SyncWorkflowTask task = new SyncWorkflowTask(rightTask, null, 1000);
        Assert.assertSame(task, task.printQuery());
        Assert.assertEquals("q-sync", task.getQuery());
    }

    @Test
    public void rightTask_printQuery_returnsThis() {
        NothingWriteBack nwb = new NothingWriteBack();
        RightTask task = new RightTask(
                RightConfig.builder()
                        .query("q-right")
                        .biz("B")
                        .trace("T")
                        .chat("C")
                        .timeout(10000)
                        .conversation("CO")
                        .upstream("U")
                        .notifier("N")
                        .protocol("P")
                        .metadata(new HashMap<>())
                        .workflow("W")
                        .build()
                        .init(),
                nwb);
        task.init();
        Assert.assertSame(task, task.printQuery());
        Assert.assertEquals("q-right", task.getQuery());
    }

    @Test
    public void localhostWorkflowTaskImpl_printQuery_returnsThis() {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        LocalhostNotifier.WorkflowTaskImpl task = new LocalhostNotifier.WorkflowTaskImpl(segment, new NothingWriteBack());
        task.setQuery("q-impl");
        Assert.assertSame(task, task.printQuery());
        Assert.assertEquals("q-impl", task.getQuery());
    }

    @Test
    public void messageDelegate_printQuery_returnsThis() {
        LLMQuery q = ObjectBuilder.buildLLMQuery();
        q.setQuery("q-message");
        MessageDelegate message = new MessageDelegate(q);
        Assert.assertSame(message, message.printQuery());
        Assert.assertEquals("q-message", message.getQuery());
    }
}
