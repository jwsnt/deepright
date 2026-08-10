package ai.open.right.netty.a2a.server;

import ai.open.right.netty.NettyCaptor;
import ai.open.right.netty.a2a.NettyInputProxy;
import ai.open.right.netty.a2a.distribute.NettyDistributor;
import ai.open.right.utils.DumpUtils;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.PooledByteBufAllocator;
import io.netty.channel.ChannelHandlerContext;
import io.netty.channel.ChannelInboundHandlerAdapter;
import io.netty.channel.embedded.EmbeddedChannel;
import io.netty.handler.codec.http.FullHttpRequest;
import io.netty.handler.codec.http.HttpMethod;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.apache.commons.io.FileUtils;
import org.springframework.util.ResourceUtils;

import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.stream.Stream;

public class NettyA2AHandlerTest {

    @Test
    public void test() throws Exception {
        String json = IOUtils.toString(ResourceUtils.getURL("classpath:MCP_Prompt_list_response.json").openStream(), StandardCharsets.UTF_8);
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf = buf.writeBytes(json.getBytes(StandardCharsets.UTF_8));
        FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
        EasyMock.expect(fullHttpRequest.retain()).andReturn(fullHttpRequest).anyTimes();
        EasyMock.expect(fullHttpRequest.method()).andReturn(HttpMethod.POST).anyTimes();
        EasyMock.expect(fullHttpRequest.content()).andReturn(buf).anyTimes();
        EasyMock.expect(fullHttpRequest.refCnt()).andReturn(1).anyTimes();
        EasyMock.expect(fullHttpRequest.release()).andReturn(true).anyTimes();
        EasyMock.replay(fullHttpRequest);
        NettyInputProxy inputProxy = new NettyInputProxy(fullHttpRequest);
        ChannelInboundHandlerAdapter tail = new ChannelInboundHandlerAdapter() {
        };
        EmbeddedChannel embeddedChannel = new EmbeddedChannel(tail);
        ChannelHandlerContext context = embeddedChannel.pipeline().context(tail);
        NettyDistributor distributor = new NettyDistributor() {
            @Override
            public void distribute(io.netty.channel.ChannelHandlerContext ctx, NettyInputProxy proxy, NettyCaptor captor) throws Exception {
                // 不跑真实 Trace/Workflow，仅覆盖 channelRead 成功路径
            }
        };
        NettyA2AHandler nettyA2AHandler = new NettyA2AHandler() {
            @Override
            protected NettyInputProxy buildInputProxy(Object msg) {
                return inputProxy;
            }
        };
        nettyA2AHandler.setDistributor(distributor);
        nettyA2AHandler.channelRead(context, fullHttpRequest);
        EasyMock.verify(fullHttpRequest);
    }

    @Test
    public void testWithException() throws Exception {
        String json = IOUtils.toString(ResourceUtils.getURL("classpath:MCP_Prompt_list_response.json").openStream(), StandardCharsets.UTF_8);
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf = buf.writeBytes(json.getBytes(StandardCharsets.UTF_8));
        FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
        EasyMock.expect(fullHttpRequest.retain()).andReturn(fullHttpRequest).anyTimes();
        EasyMock.expect(fullHttpRequest.method()).andReturn(HttpMethod.POST).anyTimes();
        EasyMock.expect(fullHttpRequest.content()).andReturn(buf).anyTimes();
        EasyMock.expect(fullHttpRequest.refCnt()).andReturn(1).anyTimes();
        EasyMock.expect(fullHttpRequest.release()).andReturn(true).anyTimes();
        EasyMock.replay(fullHttpRequest);
        NettyInputProxy inputProxy = new NettyInputProxy(fullHttpRequest);
        ChannelInboundHandlerAdapter tail = new ChannelInboundHandlerAdapter() {
        };
        EmbeddedChannel embeddedChannel = new EmbeddedChannel(tail);
        ChannelHandlerContext context = embeddedChannel.pipeline().context(tail);
        NettyDistributor distributor = new NettyDistributor() {
            @Override
            public void distribute(io.netty.channel.ChannelHandlerContext ctx, NettyInputProxy proxy, NettyCaptor captor) throws Exception {
                throw new RuntimeException();
            }
        };
        AtomicBoolean ex = new AtomicBoolean(false);
        NettyA2AHandler nettyA2AHandler = new NettyA2AHandler() {
            @Override
            protected NettyInputProxy buildInputProxy(Object msg) {
                return inputProxy;
            }

            @Override
            // 基础异常处理，实现类覆盖
            public void exceptionCaught(ChannelHandlerContext ctx, Throwable cause) throws Exception {
                ex.set(true);
            }
        };
        nettyA2AHandler.setDistributor(distributor);
        nettyA2AHandler.channelRead(context, fullHttpRequest);
        Assert.assertTrue(ex.get());
        EasyMock.verify(fullHttpRequest);
    }

    @Test
    public void testBuildInput() throws Exception {
        String json = IOUtils.toString(ResourceUtils.getURL("classpath:A2A_MessageSend_request.json").openStream(), StandardCharsets.UTF_8);
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf = buf.writeBytes(json.getBytes(StandardCharsets.UTF_8));
        FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
        EasyMock.expect(fullHttpRequest.retain()).andReturn(fullHttpRequest).anyTimes();
        EasyMock.expect(fullHttpRequest.method()).andReturn(HttpMethod.POST).anyTimes();
        EasyMock.expect(fullHttpRequest.content()).andReturn(buf).anyTimes();
        EasyMock.replay(fullHttpRequest);
        NettyA2AHandler nettyA2AHandler = new NettyA2AHandler();
        NettyInputProxy nettyInputProxy = nettyA2AHandler.buildInputProxy(fullHttpRequest);
        Assert.assertEquals(fullHttpRequest, nettyInputProxy.getRequest());
        EasyMock.verify(fullHttpRequest);
    }

    @Test
    public void testSetGet() {
        NettyDistributor nettyDistributor = new NettyDistributor();
        NettyA2AHandler nettyA2AHandler = new NettyA2AHandler();
        nettyA2AHandler.setDistributor(nettyDistributor);
        Assert.assertEquals(nettyDistributor, nettyA2AHandler.getDistributor());
    }

    @Test
    public void testHarnessGetterSetter() {
        NettyA2AHandler handler = new NettyA2AHandler();
        handler.setAutoDump("/tmp/a2a-dumps");
        Assert.assertEquals("/tmp/a2a-dumps", handler.getAutoDump());
    }

    @Test
    public void testBuildInputProxy_passesHarnessForDump() throws Exception {
        Path dir = Files.createTempDirectory("a2a-handler-harness");
        try {
            String json = "{\"fromHandler\":true}";
            ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
            buf.writeBytes(json.getBytes(StandardCharsets.UTF_8));
            FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
            EasyMock.expect(fullHttpRequest.retain()).andReturn(fullHttpRequest).anyTimes();
            EasyMock.expect(fullHttpRequest.method()).andReturn(HttpMethod.POST).anyTimes();
            EasyMock.expect(fullHttpRequest.content()).andReturn(buf).anyTimes();
            EasyMock.expect(fullHttpRequest.refCnt()).andReturn(1).anyTimes();
            EasyMock.expect(fullHttpRequest.release()).andReturn(true).anyTimes();
            EasyMock.replay(fullHttpRequest);
            NettyA2AHandler handler = new NettyA2AHandler();
            handler.setAutoDump(dir.toAbsolutePath().toString());
            NettyInputProxy proxy = handler.buildInputProxy(fullHttpRequest);
            Assert.assertNotNull(proxy.getContent());
            try (Stream<Path> list = Files.list(dir)) {
                long n = list.filter(p -> p.getFileName().toString().startsWith(DumpUtils.DUMP_PREFIX + "_REQUEST_A2A_")).count();
                Assert.assertEquals(1L, n);
            }
            proxy.close();
            EasyMock.verify(fullHttpRequest);
        } finally {
            FileUtils.deleteDirectory(dir.toFile());
        }
    }
}
