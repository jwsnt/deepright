package ai.open.right.workflow.notify.impl;
import org.easymock.EasyMock;
import ai.open.right.context.RedirectContext;
import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.SegmentDelegate;
import ai.open.right.workflow.flow.trigger.WorkflowTrigger;
import ai.open.right.workflow.notify.NothingWriteBack;
import ai.open.right.workflow.notify.NotifierWriteBack;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.Map;

public class SourceNotifierTest {

    @Test
    public void test1() throws Exception {
        SourceNotifier sourceNotifier = new SourceNotifier();
        SegmentDelegate _segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        _segment.setFinished(true);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        sourceNotifier.notify(_segment, workflowTask, new NothingWriteBack() {
            @Override
            public void writeSource(Segment segment) throws Exception {
                Assert.assertEquals(_segment, segment);
            }

            @Override
            public void writeBack(Segment segment) throws Exception {
                this.writeSource(segment);

            }
        }, new ArrayList<>());
    }

    @Test
    public void test2() throws Exception {
        SourceNotifier sourceNotifier = new SourceNotifier();
        SegmentDelegate _segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        _segment.setFinished(true);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        sourceNotifier.notify(_segment, workflowTask, new NothingWriteBack() {
            @Override
            public void writeSource(Segment segment) throws Exception {
                Assert.assertEquals(_segment, segment);
            }

            @Override
            public void writeBack(Segment segment) throws Exception {
                this.writeSource(segment);
            }
        });
    }

    @Test
    public void test3() throws Exception {
        SourceNotifier sourceNotifier = new SourceNotifier();
        SegmentDelegate _segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        _segment.setFinished(true);
        sourceNotifier.notify(_segment, new NothingWriteBack() {
            @Override
            public void writeSource(Segment segment) throws Exception {
                Assert.assertEquals(_segment, segment);
            }

            @Override
            public void writeBack(Segment segment) throws Exception {
                this.writeSource(segment);
            }
        }, new ArrayList<>());
    }

    @Test
    public void test4() throws Exception {
        SourceNotifier sourceNotifier = new SourceNotifier();
        SegmentDelegate _segment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        _segment.setFinished(true);
        sourceNotifier.notify(_segment, new NothingWriteBack() {
            @Override
            public void writeSource(Segment segment) throws Exception {
                Assert.assertEquals(_segment, segment);
            }

            @Override
            public void writeBack(Segment segment) throws Exception {
                this.writeSource(segment);
            }
        });
    }

    @Test
    public void testInit() throws Exception {
        SourceNotifier sourceNotifier = new SourceNotifier();
        SourceNotifier empty = new SourceNotifier.InitConfig().sourceNotifier();
        Assert.assertNotNull(empty);
    }
    @Test
    public void testNotifySilent() throws Exception {
        SourceNotifier notifier = new SourceNotifier();
        Segment segment = EasyMock.createMock(Segment.class);
        EasyMock.expect(segment.getSilent()).andReturn(true).anyTimes();
        EasyMock.replay(segment);
        notifier.notify(segment, (RedirectContext) null, null);
        EasyMock.verify(segment);
    }
}
