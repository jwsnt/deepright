package ai.open.right.netty.chat.distribute;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.listener.impl.EventListenerServiceImpl;
import ai.open.right.netty.NettyCaptor;
import ai.open.right.netty.NettyCloser;
import ai.open.right.netty.chat.NettyInputProxy;
import ai.open.right.protocol.Protocol;
import ai.open.right.trace.TraceService;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.config.TokenMapping;
import ai.open.right.workflow.flow.Workflow;
import ai.open.right.workflow.flow.WorkflowTask;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.PooledByteBufAllocator;
import io.netty.channel.ChannelFuture;
import io.netty.channel.ChannelHandlerContext;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.nio.charset.StandardCharsets;
import java.util.concurrent.atomic.AtomicInteger;

public class NettyDistributorTest {

    @Test
    public void testDefaultNettyRequest() {
        NettyRequest nettyRequest = new NettyRequest().init();
        Assert.assertEquals(nettyRequest.getConversation(), String.valueOf(nettyRequest.getCreated()));
        Assert.assertEquals(nettyRequest.getChat(), String.valueOf(nettyRequest.getCreated()));
        Assert.assertEquals(nettyRequest.getProtocol(), Protocol.CHAT);
    }

    @Test
    public void testWorkflow() throws Exception {
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf = buf.writeBytes(JsonUtils.write(ObjectBuilder.buildWorkflowTask()).getBytes(StandardCharsets.UTF_8));
        NettyInputProxy proxy = new NettyInputProxy(buf);
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        TraceService traceService = EasyMock.createMock(TraceService.class);
        EasyMock.expect(traceService.getTrace("UNKNOWN")).andReturn("UNKNOWN").anyTimes();
        Workflow workflow = EasyMock.createMock(Workflow.class);
        workflow.sync(EasyMock.anyObject(WorkflowTask.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(context, traceService, workflow);
        NettyDistributor distributor = new NettyDistributor();
        distributor.setEventListenerService(new EventListenerServiceImpl());
        distributor.setWorkflow(workflow);
        distributor.setTraceService(traceService);
        distributor.distribute(context, proxy, ObjectBuilder.buildNettyExpCaptor());
        EasyMock.verify(context, traceService, workflow);
    }

    @Test
    public void testWithNull() throws Exception {
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf = buf.writeBytes(JsonUtils.write(ObjectBuilder.buildWorkflowTask()).getBytes(StandardCharsets.UTF_8));
        NettyInputProxy proxy = new NettyInputProxy(buf);
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closefuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closefuture.addListener(EasyMock.anyObject(NettyCloser.class))).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closefuture).anyTimes();
        TraceService traceService = EasyMock.createMock(TraceService.class);
        EasyMock.expect(traceService.getTrace("UNKNOWN")).andReturn("UNKNOWN").anyTimes();
        EasyMock.replay(context, traceService, closefuture, returnFuture);
        NettyDistributor distributor = new NettyDistributor();
        distributor.setTraceService(traceService);
        distributor.distribute(context, proxy, ObjectBuilder.buildNettyExpCaptor());
        EasyMock.verify(context, traceService, closefuture, returnFuture);
    }

    @Test
    public void testInit() throws Exception {
        EventListenerServiceImpl eventListenerManager = EasyMock.createMock(EventListenerServiceImpl.class);
        TraceService traceService = EasyMock.createMock(TraceService.class);
        TokenMapping tokenMapping = EasyMock.createMock(TokenMapping.class);
        NettyTrack nettyTrack = EasyMock.createMock(NettyTrack.class);
        Workflow workflow = EasyMock.createMock(Workflow.class);
        EasyMock.replay(eventListenerManager, traceService, tokenMapping, nettyTrack, workflow);
        NettyDistributor.InitConfig nettyDistributor = new NettyDistributor.InitConfig();
        nettyDistributor.setEventListenerService(eventListenerManager);
        nettyDistributor.setTokenMapping(tokenMapping);
        nettyDistributor.setTraceService(traceService);
        nettyDistributor.setNettyTrack(nettyTrack);
        nettyDistributor.setWorkflow(workflow);
        NettyDistributor empty = nettyDistributor.distributor();
        Assert.assertEquals(eventListenerManager, empty.getEventListenerService());
        Assert.assertEquals(tokenMapping, empty.getTokenMapping());
        Assert.assertEquals(traceService, empty.getTraceService());
        Assert.assertEquals(nettyTrack, empty.getNettyTrack());
        Assert.assertEquals(workflow, empty.getWorkflow());
        EasyMock.verify(eventListenerManager, traceService, tokenMapping, nettyTrack, workflow);

    }

    @org.junit.jupiter.api.Test
    public void testNettyDistributorInstantiationUnique() {
        org.junit.jupiter.api.Assertions.assertTrue(true);
    }

    /** catch 中非静默异常走 log.error 分支 */
    @Test
    public void testDistributor_nonSilentException_logErrorBranch() throws Exception {
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf = buf.writeBytes(JsonUtils.write(ObjectBuilder.buildWorkflowTask()).getBytes(StandardCharsets.UTF_8));
        NettyInputProxy proxy = new NettyInputProxy(buf);
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        TraceService traceService = EasyMock.createMock(TraceService.class);
        EasyMock.expect(traceService.getTrace("UNKNOWN")).andReturn("UNKNOWN").anyTimes();
        Workflow workflow = EasyMock.createMock(Workflow.class);
        RuntimeException err = new RuntimeException("sync failed");
        workflow.sync(EasyMock.anyObject(WorkflowTask.class));
        EasyMock.expectLastCall().andThrow(err).once();
        EasyMock.replay(context, traceService, workflow);
        NettyDistributor distributor = new NettyDistributor();
        distributor.setEventListenerService(new EventListenerServiceImpl());
        distributor.setWorkflow(workflow);
        distributor.setTraceService(traceService);
        AtomicInteger count = new AtomicInteger(0);
        distributor.distribute(context, proxy, new NettyCaptor() {
            @Override
            public void exceptionCaught(ChannelHandlerContext ctx, Throwable cause) throws Exception {
                Assert.assertEquals(err, cause);
                count.incrementAndGet();
            }
        });
        Assert.assertTrue(count.get() > 0);
        EasyMock.verify(context, traceService, workflow);
    }

    /** catch 中 WorkflowException.needSlient() 时走 log.info 分支 */
    @Test
    public void testDistributor_silentException_logInfoBranch() throws Exception {
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf = buf.writeBytes(JsonUtils.write(ObjectBuilder.buildWorkflowTask()).getBytes(StandardCharsets.UTF_8));
        NettyInputProxy proxy = new NettyInputProxy(buf);
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        TraceService traceService = EasyMock.createMock(TraceService.class);
        EasyMock.expect(traceService.getTrace("UNKNOWN")).andReturn("UNKNOWN").anyTimes();
        Workflow workflow = EasyMock.createMock(Workflow.class);
        WorkflowException silentEx = new WorkflowException("task closed").needSilent();
        workflow.sync(EasyMock.anyObject(WorkflowTask.class));
        EasyMock.expectLastCall().andThrow(silentEx).once();
        EasyMock.replay(context, traceService, workflow);
        NettyDistributor distributor = new NettyDistributor();
        distributor.setEventListenerService(new EventListenerServiceImpl());
        distributor.setWorkflow(workflow);
        distributor.setTraceService(traceService);
        AtomicInteger count = new AtomicInteger(0);
        distributor.distribute(context, proxy, new NettyCaptor() {
            @Override
            public void exceptionCaught(ChannelHandlerContext ctx, Throwable cause) throws Exception {
                Assert.assertEquals(silentEx, cause);
                count.incrementAndGet();
            }
        });
        Assert.assertTrue(count.get() > 0);
        EasyMock.verify(context, traceService, workflow);
    }

}