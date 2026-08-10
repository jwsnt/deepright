package ai.open.right.workflow.flow.parallel.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.context.RedirectContext;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.SegmentUsage;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.flow.parallel.ParallelConfig;
import ai.open.right.workflow.flow.parallel.ParallelFlow;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;
import java.util.List;

public class ParallelServiceImplTest {

    @Test
    public void testGetParallelResponse() throws Exception {
        SyncWorkflowTask syncWorkflowTask1 = new SyncWorkflowTask(ObjectBuilder.buildLLMQuery(), null, 1000);
        Segment segment1 = Segment.build(ObjectBuilder.buildWorkflowTask(), Segment.SegmentConfig.builder().build());
        segment1.setUsage(new SegmentUsage());
        syncWorkflowTask1.writeBack(segment1);
        SyncWorkflowTask syncWorkflowTask2 = new SyncWorkflowTask(ObjectBuilder.buildLLMQuery(), null, 1000);
        Segment segment2 = Segment.build(ObjectBuilder.buildWorkflowTask(), Segment.SegmentConfig.builder().build());
        segment2.setUsage(new SegmentUsage());
        syncWorkflowTask2.writeBack(segment2);
        ParallelConfig parallelConfig = new ParallelConfig();
        ParallelServiceImpl parallelService = new ParallelServiceImpl();
        Assert.assertEquals("UNKNOWNUNKNOWN", parallelService.getParallelResponse(parallelConfig, Arrays.asList(syncWorkflowTask1, syncWorkflowTask2)).toString());
    }

    @Test(expected = WorkflowException.class)
    public void testGetParallelResponseWithException() throws Exception {
        SyncWorkflowTask syncWorkflowTask = new SyncWorkflowTask(ObjectBuilder.buildLLMQuery(), null, 1000);
        Segment segment = Segment.build(ObjectBuilder.buildWorkflowTask(), Segment.SegmentConfig.builder()
                .code(ProtocolCode.C500)
                .build());
        segment.setUsage(new SegmentUsage());
        syncWorkflowTask.writeBack(segment);
        ParallelConfig parallelConfig = new ParallelConfig();
        ParallelFlow parallelFlow = new ParallelFlow();
        parallelFlow.setStopOnFailed(true);
        parallelConfig.setParallelFlow(Arrays.asList(parallelFlow));
        ParallelServiceImpl parallelService = new ParallelServiceImpl();
        parallelService.getParallelResponse(parallelConfig, Arrays.asList(syncWorkflowTask));
        Assert.fail();
    }

    @Test(expected = IllegalArgumentException.class)
    public void testGetParallelResponseWithExceptionAndNotStopOnFailed() throws Exception {
        SyncWorkflowTask syncWorkflowTask = new SyncWorkflowTask(ObjectBuilder.buildLLMQuery(), null, 1000);
        Segment segment = Segment.build(ObjectBuilder.buildWorkflowTask(), Segment.SegmentConfig.builder()
                .code(ProtocolCode.C500)
                .build());
        segment.setUsage(new SegmentUsage());
        syncWorkflowTask.writeBack(segment);
        ParallelConfig parallelConfig = new ParallelConfig();
        ParallelFlow parallelFlow = new ParallelFlow();
        parallelFlow.setStopOnFailed(false);
        parallelConfig.setParallelFlow(Arrays.asList(parallelFlow));
        ParallelServiceImpl parallelService = new ParallelServiceImpl();
        parallelService.getParallelResponse(parallelConfig, Arrays.asList(syncWorkflowTask));
    }

    @Test
    public void testExecute() throws Exception {
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Segment segment1 = Segment.build(ObjectBuilder.buildWorkflowTask(), Segment.SegmentConfig.builder().build());
                segment1.setUsage(new SegmentUsage());
                notifierWriteBack.writeBack(segment1);
            }
        };
        ParallelConfig parallelConfig = new ParallelConfig();
        ParallelFlow parallelFlow1 = new ParallelFlow();
        parallelFlow1.setDynamic("WORKFLOW1");
        parallelFlow1.setStopOnFailed(true);
        ParallelFlow parallelFlow2 = new ParallelFlow();
        parallelFlow2.setDynamic("WORKFLOW2");
        parallelFlow2.setStopOnFailed(true);
        parallelConfig.setParallelFlow(Arrays.asList(parallelFlow1, parallelFlow2));
        ParallelServiceImpl parallelService = new ParallelServiceImpl();
        parallelService.setNotifierService(notifierManager);
        parallelService.setTimeout4Llm(1000);
        Assert.assertEquals("UNKNOWNUNKNOWN", parallelService.execute(parallelConfig, ObjectBuilder.buildWorkflowTask()));
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        ParallelServiceImpl.InitConfig service = new ParallelServiceImpl.InitConfig();
        service.setNotifierService(notifierManager);
        service.setTimeout4Llm(1000);
        ParallelServiceImpl empty = (ParallelServiceImpl) service.parallelService();
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(Integer.valueOf(1000), empty.getTimeout4Llm());
    }
}
