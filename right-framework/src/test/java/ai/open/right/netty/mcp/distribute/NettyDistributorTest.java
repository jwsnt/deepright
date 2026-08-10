package ai.open.right.netty.mcp.distribute;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.netty.NettyCaptor;
import ai.open.right.netty.mcp.NettyInputProxy;
import ai.open.right.trace.TraceService;
import ai.open.right.workflow.flow.Workflow;
import ai.open.right.workflow.flow.WorkflowTask;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.PooledByteBufAllocator;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.FullHttpRequest;
import io.netty.handler.codec.http.HttpHeaders;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Map;
import java.util.concurrent.atomic.AtomicInteger;

public class NettyDistributorTest {

    @Test
    public void testDistributor() throws Exception {
        String json = IOUtils.toString(ResourceUtils.getURL("classpath:MCP_Prompt_list_response.json").openStream(), StandardCharsets.UTF_8);
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf = buf.writeBytes(json.getBytes(StandardCharsets.UTF_8));
        FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
        EasyMock.expect(fullHttpRequest.retain()).andReturn(fullHttpRequest).anyTimes();
        EasyMock.expect(fullHttpRequest.content()).andReturn(buf).anyTimes();
        EasyMock.expect(fullHttpRequest.refCnt()).andReturn(1).anyTimes();
        EasyMock.expect(fullHttpRequest.release()).andReturn(true).anyTimes();
        HttpHeaders headers = EasyMock.createMock(HttpHeaders.class);
        EasyMock.expect(headers.iterator()).andReturn(new ArrayList<Map.Entry<String, String>>().iterator()).anyTimes();
        EasyMock.expect(fullHttpRequest.headers()).andReturn(headers).anyTimes();
        EasyMock.replay(fullHttpRequest);
        NettyInputProxy proxy = new NettyInputProxy(fullHttpRequest);
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        TraceService traceService = EasyMock.createMock(TraceService.class);
        EasyMock.expect(traceService.getTrace(null)).andReturn("UNKNOWN").anyTimes();
        Workflow workflow = EasyMock.createMock(Workflow.class);
        workflow.async(EasyMock.anyObject(WorkflowTask.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(headers, context, traceService, workflow);
        NettyDistributor distributor = new NettyDistributor();
        distributor.setTraceService(traceService);
        distributor.distribute(context, proxy, ObjectBuilder.buildNettyExpCaptor());
        EasyMock.verify(headers, context, traceService, workflow, fullHttpRequest);
    }

    @Test
    public void testDistributorWithException() throws Exception {
        String json = IOUtils.toString(ResourceUtils.getURL("classpath:MCP_Prompt_list_response.json").openStream(), StandardCharsets.UTF_8);
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf = buf.writeBytes(json.getBytes(StandardCharsets.UTF_8));
        FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
        EasyMock.expect(fullHttpRequest.retain()).andReturn(fullHttpRequest).anyTimes();
        EasyMock.expect(fullHttpRequest.content()).andReturn(buf).anyTimes();
        EasyMock.expect(fullHttpRequest.refCnt()).andReturn(1).anyTimes();
        EasyMock.expect(fullHttpRequest.release()).andReturn(true).anyTimes();
        HttpHeaders headers = EasyMock.createMock(HttpHeaders.class);
        EasyMock.expect(headers.iterator()).andReturn(new ArrayList<Map.Entry<String, String>>().iterator()).anyTimes();
        EasyMock.expect(fullHttpRequest.headers()).andReturn(headers).anyTimes();
        EasyMock.replay(fullHttpRequest);
        NettyInputProxy proxy = new NettyInputProxy(fullHttpRequest);
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        TraceService traceService = EasyMock.createMock(TraceService.class);
        RuntimeException runtimeException = new RuntimeException();
        EasyMock.expect(traceService.getTrace(null)).andThrow(runtimeException).anyTimes();
        Workflow workflow = EasyMock.createMock(Workflow.class);
        workflow.async(EasyMock.anyObject(WorkflowTask.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(headers, context, traceService, workflow);
        NettyDistributor distributor = new NettyDistributor();
        distributor.setTraceService(traceService);
        try {
            AtomicInteger count = new AtomicInteger(0);
            distributor.distribute(context, proxy, new NettyCaptor() {
                @Override
                public void exceptionCaught(ChannelHandlerContext ctx, Throwable cause) throws Exception {
                    Assert.assertEquals(runtimeException, cause);
                    count.incrementAndGet();
                }
            });
            Assert.assertTrue(count.get() > 0);
        } finally {
            EasyMock.verify(headers, context, traceService, workflow, fullHttpRequest);
        }
    }

    /** catch 中 WorkflowException.needSlient() 时走 log.info 分支 */
    @Test
    public void testDistributor_silentException_logInfoBranch() throws Exception {
        String json = IOUtils.toString(ResourceUtils.getURL("classpath:MCP_Prompt_list_response.json").openStream(), StandardCharsets.UTF_8);
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf = buf.writeBytes(json.getBytes(StandardCharsets.UTF_8));
        FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
        EasyMock.expect(fullHttpRequest.retain()).andReturn(fullHttpRequest).anyTimes();
        EasyMock.expect(fullHttpRequest.content()).andReturn(buf).anyTimes();
        EasyMock.expect(fullHttpRequest.refCnt()).andReturn(1).anyTimes();
        EasyMock.expect(fullHttpRequest.release()).andReturn(true).anyTimes();
        HttpHeaders headers = EasyMock.createMock(HttpHeaders.class);
        EasyMock.expect(headers.iterator()).andReturn(new ArrayList<Map.Entry<String, String>>().iterator()).anyTimes();
        EasyMock.expect(fullHttpRequest.headers()).andReturn(headers).anyTimes();
        EasyMock.replay(fullHttpRequest);
        NettyInputProxy proxy = new NettyInputProxy(fullHttpRequest);
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        TraceService traceService = EasyMock.createMock(TraceService.class);
        WorkflowException silentEx = new WorkflowException("task closed").needSilent();
        EasyMock.expect(traceService.getTrace(null)).andThrow(silentEx).anyTimes();
        Workflow workflow = EasyMock.createMock(Workflow.class);
        workflow.async(EasyMock.anyObject(WorkflowTask.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(headers, context, traceService, workflow);
        NettyDistributor distributor = new NettyDistributor();
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
        EasyMock.verify(headers, context, traceService, workflow, fullHttpRequest);
    }
}
