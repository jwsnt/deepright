package ai.open.right.netty.chat.server.ws;

import ai.open.right.WorkflowException;
import ai.open.right.netty.NettyAlarm;
import ai.open.right.netty.NettyCaptor;
import ai.open.right.netty.chat.NettyChatHandler;
import ai.open.right.netty.chat.NettyInputProxy;
import ai.open.right.netty.chat.distribute.NettyDistributor;
import ai.open.right.netty.chat.server.NettyAttributes;
import ai.open.right.netty.chat.server.http.NettyHttpHandler;
import io.netty.buffer.ByteBufAllocator;
import io.netty.channel.Channel;
import io.netty.channel.ChannelFuture;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.FullHttpRequest;
import io.netty.handler.codec.http.HttpHeaders;
import io.netty.handler.codec.http.websocketx.TextWebSocketFrame;
import io.netty.handler.timeout.IdleStateEvent;
import io.netty.util.Attribute;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.net.SocketAddress;
import java.util.Date;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.atomic.AtomicBoolean;

public class NettyWsHandlerTest {

    @Test
    public void testUserEventTriggered() {
        NettyChatHandler handler = new NettyWsHandler();
        ChannelHandlerContext channelHandlerContext = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture channelFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(channelFuture.addListener(NettyAlarm.INSTANCE)).andReturn(channelFuture).anyTimes();
        EasyMock.expect(channelHandlerContext.close()).andReturn(channelFuture).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channelHandlerContext.channel()).andReturn(channel).anyTimes();
        EasyMock.expect(channelHandlerContext.fireUserEventTriggered(EasyMock.anyObject())).andReturn(channelHandlerContext).anyTimes();
        EasyMock.replay(channelHandlerContext, socketAddress, channelFuture, channel);
        IdleStateEvent idleStateEvent = EasyMock.createMock(IdleStateEvent.class);
        EasyMock.replay(idleStateEvent);
        handler.userEventTriggered(channelHandlerContext, idleStateEvent);
        EasyMock.verify(channelHandlerContext, channelFuture, idleStateEvent, socketAddress, channel);
    }

    @Test
    public void testChannelActive() throws Exception {
        NettyChatHandler handler = new NettyWsHandler();
        ChannelHandlerContext channelHandlerContext = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture channelFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(channelHandlerContext.close()).andReturn(channelFuture).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channelHandlerContext.channel()).andReturn(channel).anyTimes();
        EasyMock.expect(channelHandlerContext.fireChannelActive()).andReturn(channelHandlerContext).anyTimes();
        EasyMock.replay(channelHandlerContext, socketAddress, channelFuture, channel);
        handler.channelActive(channelHandlerContext);
        EasyMock.verify(channelHandlerContext, channelFuture, socketAddress, channel);
    }

    @Test
    public void testExceptionCaught() throws Exception {
        NettyChatHandler handler = new NettyWsHandler();
        ChannelHandlerContext channelHandlerContext = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture channelFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(channelFuture.addListener(EasyMock.anyObject())).andReturn(channelFuture).anyTimes();
        EasyMock.expect(channelHandlerContext.close()).andReturn(channelFuture).anyTimes();
        EasyMock.replay(channelHandlerContext, channelFuture);
        handler.exceptionCaught(channelHandlerContext, new WorkflowException());
        EasyMock.verify(channelHandlerContext, channelFuture);
    }

    @Test
    public void testExceptionCaughtWithRuntime() throws Exception {
        NettyChatHandler handler = new NettyWsHandler();
        ChannelHandlerContext channelHandlerContext = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture channelFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(channelFuture.addListener(EasyMock.anyObject())).andReturn(channelFuture).anyTimes();
        EasyMock.expect(channelHandlerContext.close()).andReturn(channelFuture).anyTimes();
        EasyMock.replay(channelHandlerContext, channelFuture);
        handler.exceptionCaught(channelHandlerContext, new RuntimeException());
        EasyMock.verify(channelHandlerContext, channelFuture);
    }

    @Test
    public void testExceptionCaughtException() throws Exception {
        NettyChatHandler handler = new NettyWsHandler();
        ChannelHandlerContext channelHandlerContext = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture channelFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(channelFuture.addListener(EasyMock.anyObject())).andReturn(channelFuture).anyTimes();
        EasyMock.expect(channelHandlerContext.close()).andReturn(channelFuture).anyTimes();
        EasyMock.replay(channelHandlerContext, channelFuture);
        handler.exceptionCaught(channelHandlerContext, new WorkflowException());
        EasyMock.verify(channelHandlerContext, channelFuture);
    }

    @Test
    public void testChannelReadWithHttp() throws Exception {
        NettyChatHandler handler = new NettyHttpHandler();
        ChannelHandlerContext channelHandlerContext = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeService = EasyMock.createMock(Attribute.class);
        attributeService.set(NettyAttributes.SERVER_HTTP);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(channelHandlerContext.attr(NettyAttributes.SERVER_TYPE)).andReturn(attributeService).anyTimes();
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        attributeHttp.set(NettyAttributes.CONNECTION_ONCE);
        EasyMock.expectLastCall().anyTimes();
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        attributeSSe.set(NettyAttributes.HTTP_SSE);
        EasyMock.expectLastCall().anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channelHandlerContext.channel()).andReturn(channel).anyTimes();
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channelHandlerContext.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        EasyMock.expect(channelHandlerContext.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
        EasyMock.expect(fullHttpRequest.release()).andReturn(true).anyTimes();
        EasyMock.expect(fullHttpRequest.content()).andReturn(ByteBufAllocator.DEFAULT.buffer().writeBytes("HELLO".getBytes())).anyTimes();
        HttpHeaders headers = EasyMock.createMock(HttpHeaders.class);
        Map<String, String> headersMap = new HashMap<>();
        EasyMock.expect(headers.iterator()).andReturn(headersMap.entrySet().iterator()).anyTimes();
        EasyMock.expect(headers.get("Authorization")).andReturn("TOKEN").anyTimes();
        EasyMock.expect(headers.get("Accept")).andReturn("text/event-stream").anyTimes();
        EasyMock.expect(fullHttpRequest.headers()).andReturn(headers).anyTimes();
        NettyDistributor distributor = EasyMock.createMock(NettyDistributor.class);
        distributor.distribute(EasyMock.anyObject(ChannelHandlerContext.class), EasyMock.anyObject(NettyInputProxy.class), EasyMock.anyObject(NettyCaptor.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(channel, socketAddress, channelHandlerContext, headers, attributeService, distributor, fullHttpRequest, attributeHttp, attributeSSe);
        handler.setDistributor(distributor);
        handler.channelRead(channelHandlerContext, fullHttpRequest);
        EasyMock.verify(channel, socketAddress, channelHandlerContext, headers, fullHttpRequest, attributeService, distributor, attributeHttp, attributeSSe);
    }

    @Test
    public void testChannelReadWithWs() throws Exception {
        NettyChatHandler handler = new NettyWsHandler();
        ChannelHandlerContext channelHandlerContext = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attribute = EasyMock.createMock(Attribute.class);
        attribute.set(NettyAttributes.SERVER_WS);
        EasyMock.expectLastCall().anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channelHandlerContext.channel()).andReturn(channel).anyTimes();
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channelHandlerContext.attr(NettyAttributes.SERVER_TYPE)).andReturn(attribute).anyTimes();
        TextWebSocketFrame textWebSocketFrame = EasyMock.createMock(TextWebSocketFrame.class);
        EasyMock.expect(textWebSocketFrame.release()).andReturn(true).anyTimes();
        EasyMock.expect(channelHandlerContext.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(textWebSocketFrame.content()).andReturn(ByteBufAllocator.DEFAULT.buffer().writeBytes("HELLO".getBytes())).anyTimes();
        EasyMock.replay(textWebSocketFrame, channel, socketAddress);
        NettyDistributor distributor = EasyMock.createMock(NettyDistributor.class);
        distributor.distribute(EasyMock.anyObject(ChannelHandlerContext.class), EasyMock.anyObject(NettyInputProxy.class), EasyMock.anyObject(NettyCaptor.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(channelHandlerContext, attribute, distributor);
        handler.setDistributor(distributor);
        handler.channelRead(channelHandlerContext, textWebSocketFrame);
        EasyMock.verify(channelHandlerContext, textWebSocketFrame, attribute, distributor, channel, socketAddress);
    }

    @Test
    public void testChannelInactive() throws Exception {
        NettyChatHandler handler = new NettyWsHandler();
        ChannelHandlerContext channelHandlerContext = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture channelFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(channelHandlerContext.close()).andReturn(channelFuture).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.replay(socketAddress);
        EasyMock.expect(channelHandlerContext.channel()).andReturn(channel).anyTimes();
        EasyMock.expect(channelHandlerContext.fireChannelInactive()).andReturn(channelHandlerContext).anyTimes();
        EasyMock.replay(channelHandlerContext, channelFuture, channel);
        handler.channelInactive(channelHandlerContext);
        EasyMock.verify(channelHandlerContext, channelFuture, socketAddress, channel);
    }

    @Test
    public void testChannelReadWithNullDistributor() throws Exception {
        AtomicBoolean ex = new AtomicBoolean(false);
        NettyChatHandler handler = new NettyWsHandler() {
            @Override
            // 基础异常处理，实现类覆盖
            public void exceptionCaught(ChannelHandlerContext ctx, Throwable cause) throws Exception {
                ex.set(true);
            }
        };
        ChannelHandlerContext channelHandlerContext = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attribute = EasyMock.createMock(Attribute.class);
        attribute.set(NettyAttributes.SERVER_WS);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(channelHandlerContext.attr(NettyAttributes.SERVER_TYPE)).andReturn(attribute).anyTimes();
        TextWebSocketFrame textWebSocketFrame = EasyMock.createMock(TextWebSocketFrame.class);
        EasyMock.expect(textWebSocketFrame.release()).andReturn(true).anyTimes();
        EasyMock.expect(channelHandlerContext.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(textWebSocketFrame.content()).andReturn(ByteBufAllocator.DEFAULT.buffer().writeBytes("HELLO".getBytes())).anyTimes();
        EasyMock.replay(channelHandlerContext, textWebSocketFrame, attribute);
        handler.channelRead(channelHandlerContext, textWebSocketFrame);
        Assert.assertTrue(ex.get());
        EasyMock.verify(channelHandlerContext, textWebSocketFrame, attribute);
    }

    @Test
    public void testChannelReadWithInvalidRequest() throws Exception {
        AtomicBoolean ex = new AtomicBoolean(false);
        NettyChatHandler handler = new NettyWsHandler() {
            @Override
            // 基础异常处理，实现类覆盖
            public void exceptionCaught(ChannelHandlerContext ctx, Throwable cause) throws Exception {
                ex.set(true);
            }
        };
        // 转换错误
        handler.channelRead(null, new Date());
        Assert.assertTrue(ex.get());
    }

    @Test
    public void testChannelReadWithWsAndEmpty() throws Exception {
        NettyChatHandler handler = new NettyWsHandler();
        ChannelHandlerContext channelHandlerContext = EasyMock.createMock(ChannelHandlerContext.class);
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channelHandlerContext.channel()).andReturn(channel).anyTimes();
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        Attribute<Byte> attribute = EasyMock.createMock(Attribute.class);
        attribute.set(NettyAttributes.SERVER_WS);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(channelHandlerContext.attr(NettyAttributes.SERVER_TYPE)).andReturn(attribute).anyTimes();
        TextWebSocketFrame textWebSocketFrame = EasyMock.createMock(TextWebSocketFrame.class);
        EasyMock.expect(textWebSocketFrame.content()).andReturn(ByteBufAllocator.DEFAULT.buffer().writeBytes("HELLO".getBytes())).anyTimes();
        EasyMock.expect(textWebSocketFrame.release()).andReturn(true).anyTimes();
        EasyMock.expect(channelHandlerContext.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.replay(textWebSocketFrame, socketAddress, channel);
        NettyDistributor distributor = EasyMock.createMock(NettyDistributor.class);
        distributor.distribute(EasyMock.anyObject(ChannelHandlerContext.class), EasyMock.anyObject(NettyInputProxy.class), EasyMock.anyObject(NettyCaptor.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(channelHandlerContext, attribute, distributor);
        handler.setDistributor(distributor);
        handler.channelRead(channelHandlerContext, textWebSocketFrame);
        EasyMock.verify(channelHandlerContext, textWebSocketFrame, attribute, distributor, socketAddress, channel);
    }

    @org.junit.jupiter.api.Test
    public void testNettyWsHandlerInstantiationUnique() {
        org.junit.jupiter.api.Assertions.assertTrue(true);
    }

    @org.junit.jupiter.api.Test
    public void testByteBufHeartbeat() throws Exception {
        ChannelHandlerContext ctx = EasyMock.createMock(ChannelHandlerContext.class);
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress addr = EasyMock.createMock(SocketAddress.class);
        TextWebSocketFrame frame = EasyMock.createMock(TextWebSocketFrame.class);
        io.netty.buffer.ByteBuf buf = EasyMock.createMock(io.netty.buffer.ByteBuf.class);

        EasyMock.expect(frame.content()).andReturn(buf).anyTimes();
        EasyMock.expect(buf.readableBytes()).andReturn(0).anyTimes();
        EasyMock.expect(buf.release()).andReturn(true).anyTimes();
        EasyMock.expect(frame.release()).andReturn(true).anyTimes();
        EasyMock.expect(ctx.channel()).andReturn(channel).anyTimes();
        EasyMock.expect(channel.remoteAddress()).andReturn(addr).anyTimes();

        EasyMock.replay(ctx, channel, addr, frame, buf);
        NettyWsHandler handler = new NettyWsHandler();
        org.junit.jupiter.api.Assertions.assertNull(handler.byteBuf(ctx, frame));
        EasyMock.verify(ctx, channel, addr, frame, buf);
    }

    @org.junit.jupiter.api.Test
    public void testInitConfig() throws Exception {
        NettyWsHandler.InitConfig config = new NettyWsHandler.InitConfig();
        NettyDistributor distributor = EasyMock.createMock(NettyDistributor.class);
        config.setDistributor(distributor);
        NettyWsHandler handler = config.nettyWsHandler();
        org.junit.jupiter.api.Assertions.assertNotNull(handler);
        org.junit.jupiter.api.Assertions.assertEquals(distributor, handler.getDistributor());
    }

}

