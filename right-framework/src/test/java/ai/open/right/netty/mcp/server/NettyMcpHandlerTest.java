package ai.open.right.netty.mcp.server;

import ai.open.right.netty.mcp.NettyInputProxy;
import ai.open.right.netty.mcp.distribute.NettyDistributor;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.PooledByteBufAllocator;
import io.netty.channel.Channel;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.FullHttpRequest;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.net.SocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.atomic.AtomicBoolean;

public class NettyMcpHandlerTest {

    @Test
    public void test() throws Exception {
        String json = IOUtils.toString(ResourceUtils.getURL("classpath:MCP_Prompt_list_response.json").openStream(), StandardCharsets.UTF_8);
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf = buf.writeBytes(json.getBytes(StandardCharsets.UTF_8));
        FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
        EasyMock.expect(fullHttpRequest.retain()).andReturn(fullHttpRequest).anyTimes();
        EasyMock.expect(fullHttpRequest.content()).andReturn(buf).anyTimes();
        EasyMock.expect(fullHttpRequest.refCnt()).andReturn(1).anyTimes();
        EasyMock.expect(fullHttpRequest.release()).andReturn(true).anyTimes();
        EasyMock.replay(fullHttpRequest);
        NettyInputProxy inputProxy = new NettyInputProxy(fullHttpRequest);
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        NettyDistributor distributor = EasyMock.createMock(NettyDistributor.class);
        Channel channel = EasyMock.createMock(io.netty.channel.Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        NettyMcpHandler nettyMcpHandler = new NettyMcpHandler() {
            @Override
            protected NettyInputProxy buildInputProxy(Object msg) {
                return inputProxy;
            }
        };
        distributor.distribute(context, inputProxy, nettyMcpHandler);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(context, channel, socketAddress, distributor);
        nettyMcpHandler.setDistributor(distributor);
        nettyMcpHandler.channelRead(context, fullHttpRequest);
        EasyMock.verify(context, channel, socketAddress, fullHttpRequest, distributor);
    }

    @Test
    public void testWithException() throws Exception {
        String json = IOUtils.toString(ResourceUtils.getURL("classpath:MCP_Prompt_list_response.json").openStream(), StandardCharsets.UTF_8);
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf = buf.writeBytes(json.getBytes(StandardCharsets.UTF_8));
        FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
        EasyMock.expect(fullHttpRequest.retain()).andReturn(fullHttpRequest).anyTimes();
        EasyMock.expect(fullHttpRequest.refCnt()).andReturn(1).anyTimes();
        EasyMock.expect(fullHttpRequest.release()).andReturn(true).anyTimes();
        EasyMock.expect(fullHttpRequest.content()).andReturn(buf).anyTimes();
        EasyMock.replay(fullHttpRequest);
        NettyInputProxy inputProxy = new NettyInputProxy(fullHttpRequest);
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        NettyDistributor distributor = EasyMock.createMock(NettyDistributor.class);
        Channel channel = EasyMock.createMock(io.netty.channel.Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        AtomicBoolean ex = new AtomicBoolean(false);
        NettyMcpHandler nettyMcpHandler = new NettyMcpHandler() {
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
        distributor.distribute(context, inputProxy, nettyMcpHandler);
        EasyMock.expectLastCall().andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(context, channel, socketAddress, distributor);
        nettyMcpHandler.setDistributor(distributor);
        nettyMcpHandler.channelRead(context, fullHttpRequest);
        Assert.assertTrue(ex.get());
        EasyMock.verify(context, channel, socketAddress, fullHttpRequest, distributor);
    }

    @Test
    public void testBuildInput() throws Exception {
        String json = IOUtils.toString(ResourceUtils.getURL("classpath:MCP_Prompt_list_response.json").openStream(), StandardCharsets.UTF_8);
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf = buf.writeBytes(json.getBytes(StandardCharsets.UTF_8));
        FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
        EasyMock.expect(fullHttpRequest.retain()).andReturn(fullHttpRequest).anyTimes();
        EasyMock.expect(fullHttpRequest.content()).andReturn(buf).anyTimes();
        EasyMock.replay(fullHttpRequest);
        NettyMcpHandler nettyMcpHandler = new NettyMcpHandler();
        NettyInputProxy nettyInputProxy = nettyMcpHandler.buildInputProxy(fullHttpRequest);
        Assert.assertEquals(fullHttpRequest, nettyInputProxy.getRequest());
        EasyMock.verify(fullHttpRequest);
    }

    @Test
    public void testSetGet() {
        NettyDistributor nettyDistributor = EasyMock.createMock(NettyDistributor.class);
        EasyMock.replay(nettyDistributor);
        NettyMcpHandler nettyChatHandler = new NettyMcpHandler();
        nettyChatHandler.setDistributor(nettyDistributor);
        Assert.assertEquals(nettyDistributor, nettyChatHandler.getDistributor());
        EasyMock.verify(nettyDistributor);
    }
}
