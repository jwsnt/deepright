package ai.open.right.integration.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.UserContext;
import ai.open.right.integration.RightConfig;
import ai.open.right.trace.TraceService;
import ai.open.right.workflow.flow.Workflow;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.SegmentUsage;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.junit.jupiter.api.Assertions;

import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;

public class RightServiceImplTest {

    @Test
    public void test() throws Exception {
        Map<String, Object> metadata = new HashMap<>();
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig rightConfig = RightConfig.builder()
                .query("Query")
                .biz("Biz")
                .trace("Trace")
                .chat("Chat")
                .timeout(10000)
                .conversation("Conversation")
                .userContext(userContext)
                .upstream("Upstream")
                .notifier("Notifier")
                .protocol("Protocol")
                .metadata(metadata)
                .workflow("Workflow")
                .build();
        TraceService traceService = EasyMock.createMock(TraceService.class);
        EasyMock.expect(traceService.getTrace("Trace")).andReturn("Trace").anyTimes();
        EasyMock.replay(traceService);
        RightServiceImpl rightService = new RightServiceImpl();
        rightService.setTraceService(traceService);
        rightService.setWorkflow(new Workflow() {

            @Override
            public void async(WorkflowTask workTask) throws Exception {
                this.sync(workTask);
            }

            @Override
            public void sync(WorkflowTask workTask) throws Exception {
                Segment segment = ObjectBuilder.buildSegment();
                segment.setUsage(new SegmentUsage());
                workTask.writeBack(segment);
            }
        });
        Future<String> content = rightService.get(rightConfig);
        Assert.assertEquals("UNKNOWN", content.get());
        EasyMock.verify(traceService);
    }

    @Test
    public void testGetWithTimeout() throws Exception {
        Map<String, Object> metadata = new HashMap<>();
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig rightConfig = RightConfig.builder()
                .query("Query")
                .biz("Biz")
                .trace("Trace")
                .chat("Chat")
                .timeout(10000)
                .conversation("Conversation")
                .userContext(userContext)
                .upstream("Upstream")
                .notifier("Notifier")
                .protocol("Protocol")
                .metadata(metadata)
                .workflow("Workflow")
                .build();
        TraceService traceService = EasyMock.createMock(TraceService.class);
        EasyMock.expect(traceService.getTrace("Trace")).andReturn("Trace").anyTimes();
        EasyMock.replay(traceService);
        RightServiceImpl rightService = new RightServiceImpl();
        rightService.setTraceService(traceService);
        rightService.setWorkflow(new Workflow() {

            @Override
            public void async(WorkflowTask workTask) throws Exception {
                Segment segment = ObjectBuilder.buildSegment();
                segment.setUsage(new SegmentUsage());
                workTask.writeBack(segment);
            }

            @Override
            public void sync(WorkflowTask workTask) throws Exception {
                Segment segment = ObjectBuilder.buildSegment();
                segment.setUsage(new SegmentUsage());
                workTask.writeBack(segment);
            }

        });
        Future<String> content = rightService.get(rightConfig);
        Assert.assertEquals("UNKNOWN", content.get(100, TimeUnit.MILLISECONDS));
        EasyMock.verify(traceService);
    }

    @Test
    public void testGetWithCancel() throws Exception {
        Map<String, Object> metadata = new HashMap<>();
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig rightConfig = RightConfig.builder()
                .query("Query")
                .biz("Biz")
                .trace("Trace")
                .chat("Chat")
                .timeout(10000)
                .conversation("Conversation")
                .userContext(userContext)
                .upstream("Upstream")
                .notifier("Notifier")
                .protocol("Protocol")
                .metadata(metadata)
                .workflow("Workflow")
                .build();
        TraceService traceService = EasyMock.createMock(TraceService.class);
        EasyMock.expect(traceService.getTrace("Trace")).andReturn("Trace").anyTimes();
        EasyMock.replay(traceService);
        RightServiceImpl rightService = new RightServiceImpl();
        rightService.setTraceService(traceService);
        rightService.setWorkflow(new Workflow() {

            @Override
            public void async(WorkflowTask workTask) throws Exception {
                Segment segment = ObjectBuilder.buildSegment();
                segment.setUsage(new SegmentUsage());
                workTask.writeBack(segment);
            }

            @Override
            public void sync(WorkflowTask workTask) throws Exception {
                Segment segment = ObjectBuilder.buildSegment();
                segment.setUsage(new SegmentUsage());
                workTask.writeBack(segment);
            }
        });
        Future<String> content = rightService.get(rightConfig);
        Assert.assertFalse(content.cancel(true));
        EasyMock.verify(traceService);
    }

    @Test
    public void testGetWithIsCancel() throws Exception {
        Map<String, Object> metadata = new HashMap<>();
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig rightConfig = RightConfig.builder()
                .query("Query")
                .biz("Biz")
                .trace("Trace")
                .chat("Chat")
                .timeout(10000)
                .conversation("Conversation")
                .userContext(userContext)
                .upstream("Upstream")
                .notifier("Notifier")
                .protocol("Protocol")
                .metadata(metadata)
                .workflow("Workflow")
                .build();
        TraceService traceService = EasyMock.createMock(TraceService.class);
        EasyMock.expect(traceService.getTrace("Trace")).andReturn("Trace").anyTimes();
        EasyMock.replay(traceService);
        RightServiceImpl rightService = new RightServiceImpl();
        rightService.setTraceService(traceService);
        rightService.setWorkflow(new Workflow() {

            @Override
            public void async(WorkflowTask workTask) throws Exception {
                Segment segment = ObjectBuilder.buildSegment();
                segment.setUsage(new SegmentUsage());
                workTask.writeBack(segment);
            }

            @Override
            public void sync(WorkflowTask workTask) throws Exception {
                Segment segment = ObjectBuilder.buildSegment();
                segment.setUsage(new SegmentUsage());
                workTask.writeBack(segment);
            }
        });
        Future<String> content = rightService.get(rightConfig);
        Assert.assertFalse(content.isCancelled());
        EasyMock.verify(traceService);
    }

    @Test
    public void testGetWithIsDone() throws Exception {
        Map<String, Object> metadata = new HashMap<>();
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig rightConfig = RightConfig.builder()
                .query("Query")
                .biz("Biz")
                .trace("Trace")
                .chat("Chat")
                .timeout(10000)
                .conversation("Conversation")
                .userContext(userContext)
                .upstream("Upstream")
                .notifier("Notifier")
                .protocol("Protocol")
                .metadata(metadata)
                .workflow("Workflow")
                .build();
        TraceService traceService = EasyMock.createMock(TraceService.class);
        EasyMock.expect(traceService.getTrace("Trace")).andReturn("Trace").anyTimes();
        EasyMock.replay(traceService);
        RightServiceImpl rightService = new RightServiceImpl();
        rightService.setTraceService(traceService);
        rightService.setWorkflow(new Workflow() {

            @Override
            public void async(WorkflowTask workTask) throws Exception {
                Thread.sleep(100);
                Segment segment = ObjectBuilder.buildSegment();
                segment.setUsage(new SegmentUsage());
                workTask.writeBack(segment);
            }

            @Override
            public void sync(WorkflowTask workTask) throws Exception {
                Thread.sleep(100);
                Segment segment = ObjectBuilder.buildSegment();
                segment.setUsage(new SegmentUsage());
                workTask.writeBack(segment);
            }
        });
        Future<String> content = rightService.get(rightConfig);
        Assert.assertFalse(content.isDone());
        content.get();
        Assert.assertTrue(content.isDone());
        EasyMock.verify(traceService);
    }

    @Test
    public void testGetThrowsTimeout() throws Exception {
        Map<String, Object> metadata = new HashMap<>();
        UserContext userContext = ObjectBuilder.buildEmpty();
        RightConfig rightConfig = RightConfig.builder()
                .query("Query")
                .biz("Biz")
                .trace("Trace")
                .chat("Chat")
                .timeout(1)
                .conversation("Conversation")
                .userContext(userContext)
                .upstream("Upstream")
                .notifier("Notifier")
                .protocol("Protocol")
                .metadata(metadata)
                .workflow("Workflow")
                .build();
        TraceService traceService = EasyMock.createMock(TraceService.class);
        EasyMock.expect(traceService.getTrace("Trace")).andReturn("Trace").anyTimes();
        EasyMock.replay(traceService);
        RightServiceImpl rightService = new RightServiceImpl();
        rightService.setTraceService(traceService);
        rightService.setWorkflow(new Workflow() {

            @Override
            public void async(WorkflowTask workTask) throws Exception {
                Thread.sleep(100);
                Segment segment = ObjectBuilder.buildSegment();
                segment.setUsage(new SegmentUsage());
                workTask.writeBack(segment);
            }

            @Override
            public void sync(WorkflowTask workTask) throws Exception {
                Thread.sleep(100);
                Segment segment = ObjectBuilder.buildSegment();
                segment.setUsage(new SegmentUsage());
                workTask.writeBack(segment);
            }
        });
        Future<String> content = rightService.get(rightConfig);
        Assert.assertFalse(content.isDone());
        try {
            content.get();
        } finally {
            Assert.assertTrue(content.isDone());
            EasyMock.verify(traceService);
        }
    }

    @org.junit.jupiter.api.Test
    public void testInitConfig() throws Exception {
        RightServiceImpl.InitConfig initConfig = new RightServiceImpl.InitConfig();
        TraceService traceService = EasyMock.createMock(TraceService.class);
        Workflow workflow = EasyMock.createMock(Workflow.class);
        initConfig.setTraceService(traceService);
        initConfig.setWorkflow(workflow);
        initConfig.setTimeout(1000);
        ai.open.right.integration.RightService rightService = initConfig.rightService();
        Assertions.assertTrue(rightService instanceof RightServiceImpl);
        Assertions.assertEquals(traceService, ((RightServiceImpl) rightService).getTraceService());
        Assertions.assertEquals(workflow, ((RightServiceImpl) rightService).getWorkflow());
        Assertions.assertEquals(Integer.valueOf(1000), ((RightServiceImpl) rightService).getTimeout());
    }

    @org.junit.jupiter.api.Test
    public void testGetNullConfig() throws Exception {
        RightServiceImpl rightService = new RightServiceImpl();
        Assertions.assertThrows(Exception.class, () -> rightService.get(null));
    }

    @org.junit.jupiter.api.Test
    public void testRightFutureGetException() throws Exception {
        RightServiceImpl.RightFuture rightFuture = new RightServiceImpl.RightFuture(null, null, 1000, 1000);
        // 模拟 super.get() 抛出异常（非 2xx 状态码触发 WorkflowException）
        Segment segment = ObjectBuilder.buildSegment(500);
        segment.reset(true, 0);
        rightFuture.writeBack(segment);
        Assertions.assertThrows(java.util.concurrent.ExecutionException.class, () -> rightFuture.get());
    }

    @org.junit.jupiter.api.Test
    public void testRightFutureGetWithTimeoutLog() throws Exception {
        RightServiceImpl.RightFuture rightFuture = new RightServiceImpl.RightFuture(null, null, 1000, 1000);
        Segment segment = ObjectBuilder.buildSegment(200);
        segment.setContent("result");
        segment.reset(true, 0);
        rightFuture.writeBack(segment);
        String result = rightFuture.get(100, TimeUnit.MILLISECONDS);
        Assertions.assertEquals("result", result);
    }

    @org.junit.jupiter.api.Test
    public void testRightFutureIsDoneAfterGet() throws Exception {
        RightServiceImpl.RightFuture rightFuture = new RightServiceImpl.RightFuture(null, null, 1000, 1000);
        Segment segment = ObjectBuilder.buildSegment(200);
        segment.reset(true, 0);
        rightFuture.writeBack(segment);
        Assertions.assertFalse(rightFuture.isDone());
        rightFuture.get();
        Assertions.assertTrue(rightFuture.isDone());
    }

    @org.junit.jupiter.api.Test
    public void testRightAdditional() {
        RightServiceImpl service = new RightServiceImpl();
        org.junit.jupiter.api.Assertions.assertNotNull(service);
    }

    @org.junit.jupiter.api.Test
    public void testBuildRightFuture() {
        RightServiceImpl service = new RightServiceImpl();
        service.setTimeout(5000);
        RightConfig config = RightConfig.builder().build();
        RightServiceImpl.RightFuture future = service.buildRightFuture(config);
        org.junit.jupiter.api.Assertions.assertNotNull(future);
    }


    @org.junit.jupiter.api.Test
    public void testRightServiceImplMoreGettersSetters() {
        RightServiceImpl service = new RightServiceImpl();
        TraceService traceService = EasyMock.createMock(TraceService.class);
        Workflow workflow = EasyMock.createMock(Workflow.class);
        service.setTraceService(traceService);
        service.setWorkflow(workflow);
        service.setTimeout(100);

        Assert.assertEquals(traceService, service.getTraceService());
        Assert.assertEquals(workflow, service.getWorkflow());
        Assert.assertEquals(Integer.valueOf(100), service.getTimeout());
    }

    @org.junit.jupiter.api.Test
    public void testInitConfigMoreGettersSetters() {
        RightServiceImpl.InitConfig config = new RightServiceImpl.InitConfig();
        TraceService traceService = EasyMock.createMock(TraceService.class);
        Workflow workflow = EasyMock.createMock(Workflow.class);
        config.setTraceService(traceService);
        config.setWorkflow(workflow);
        config.setTimeout(200);

        Assert.assertEquals(traceService, config.getTraceService());
        Assert.assertEquals(workflow, config.getWorkflow());
        Assert.assertEquals(Integer.valueOf(200), config.getTimeout());
    }

    @org.junit.jupiter.api.Test
    public void testBuildRightTaskDirectly() {
        RightServiceImpl service = new RightServiceImpl();
        RightConfig config = RightConfig.builder().query("test").biz("test-biz").workflow("test-workflow").trace("test-trace").build();
        RightServiceImpl.RightFuture future = new RightServiceImpl.RightFuture(null, null, 1000, 1000);
        ai.open.right.integration.RightTask task = service.buildRightTask(config, future);
        Assert.assertNotNull(task);
    }

    @org.junit.jupiter.api.Test
    public void testRightServiceGetWorkflowException() throws Exception {
        RightServiceImpl service = new RightServiceImpl();
        TraceService traceService = EasyMock.createMock(TraceService.class);
        EasyMock.expect(traceService.getTrace(EasyMock.anyString())).andReturn("trace").anyTimes();
        Workflow workflow = EasyMock.createMock(Workflow.class);
        workflow.async(EasyMock.anyObject());
        EasyMock.expectLastCall().andThrow(new RuntimeException("workflow failure"));
        EasyMock.replay(traceService, workflow);

        service.setTraceService(traceService);
        service.setWorkflow(workflow);

        RightConfig config = RightConfig.builder().trace("t").build();
        Assertions.assertThrows(RuntimeException.class, () -> service.get(config));
    }
}
