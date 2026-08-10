package ai.open.right.workflow.notify.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMQueryDelegate;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.NothingWriteBack;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;

public class EndpointNotifierTest {

    @Test
    public void testNotifier1() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMQueryDelegate delegate = new LLMQueryDelegate(workflowTask, "WR", "NO");
        EndpointNotifier endpointNotifier = new EndpointNotifier();
        endpointNotifier.notify(Segment.build(delegate, Segment.SegmentConfig.builder().build()), new NothingWriteBack() {

            @Override
            public void writeSource(Segment segment) throws Exception {
                this.writeBack(segment);
            }

            @Override
            public void writeBack(Segment segment) throws Exception {
                Assert.assertEquals(segment.getContent(), "UNKNOWN");
            }
        });
    }

    @Test
    public void testNotifier2() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMQueryDelegate delegate = new LLMQueryDelegate(workflowTask, "WR", "NO");
        EndpointNotifier endpointNotifier = new EndpointNotifier();
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        endpointNotifier.notify(Segment.build(delegate, Segment.SegmentConfig.builder().build()), new NothingWriteBack() {

            @Override
            public void writeSource(Segment segment) throws Exception {
                this.writeBack(segment);
            }

            @Override
            public void writeBack(Segment segment) throws Exception {
                Assert.assertEquals(segment.getContent(), "UNKNOWN");
            }
        }, new ArrayList<>());
    }

    @Test
    public void testNotifier3() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMQueryDelegate delegate = new LLMQueryDelegate(workflowTask, "WR", "NO");
        EndpointNotifier endpointNotifier = new EndpointNotifier();
        endpointNotifier.notify(Segment.build(delegate, Segment.SegmentConfig.builder().build()), ObjectBuilder.buildLLMQuery(), new NothingWriteBack() {

            @Override
            public void writeSource(Segment segment) throws Exception {
                this.writeBack(segment);
            }

            @Override
            public void writeBack(Segment segment) throws Exception {
                Assert.assertEquals(segment.getContent(), "UNKNOWN");
            }
        }, new ArrayList<>());
    }

    @Test
    public void testNotifier4() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMQueryDelegate delegate = new LLMQueryDelegate(workflowTask, "WR", "NO");
        EndpointNotifier endpointNotifier = new EndpointNotifier();
        endpointNotifier.notify(Segment.build(delegate, Segment.SegmentConfig.builder().build()), RedirectContext.EMPTY, new NothingWriteBack() {

            @Override
            public void writeSource(Segment segment) throws Exception {
                this.writeBack(segment);
            }

            @Override
            public void writeBack(Segment segment) throws Exception {
                Assert.assertEquals(segment.getContent(), "UNKNOWN");
            }
        }, new ArrayList<>());
    }

    @Test
    public void testNotifier5() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMQueryDelegate delegate = new LLMQueryDelegate(workflowTask, "WR", "NO");
        EndpointNotifier endpointNotifier = new EndpointNotifier();
        endpointNotifier.notify(Segment.build(delegate, Segment.SegmentConfig.builder().build()), RedirectContext.EMPTY, new NothingWriteBack() {

            @Override
            public void writeSource(Segment segment) throws Exception {
                this.writeBack(segment);
            }

            @Override
            public void writeBack(Segment segment) throws Exception {
                Assert.assertEquals(segment.getContent(), "UNKNOWN");
            }
        });
    }

    @Test
    public void testInit() throws Exception {
        EndpointNotifier endpointNotifier = new EndpointNotifier();
        EndpointNotifier empty = new EndpointNotifier.InitConfig().endpointNotifier();
        Assert.assertNotNull(empty);
    }
}
