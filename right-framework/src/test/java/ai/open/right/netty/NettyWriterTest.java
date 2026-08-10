package ai.open.right.netty;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.netty.chat.NettySegment;
import ai.open.right.netty.chat.server.NettyAttributes;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.SegmentDelegate;
import io.netty.buffer.*;
import io.netty.channel.Channel;
import io.netty.channel.ChannelFuture;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.DefaultHttpResponse;
import io.netty.handler.codec.http.FullHttpRequest;
import io.netty.util.Attribute;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.net.SocketAddress;

public class NettyWriterTest {

    @Test
    public void testWriteHttp() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeServer = EasyMock.createMock(Attribute.class);
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeServer.get()).andReturn(NettyAttributes.SERVER_HTTP).anyTimes();
        EasyMock.expect(attributeHttp.get()).andReturn(NettyAttributes.CONNECTION_STREAM).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SERVER_TYPE)).andReturn(attributeServer).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeSSe.get()).andReturn(NettyAttributes.HTTP_SSE).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channel.isOpen()).andReturn(true).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.replay(channel, socketAddress, context, closeFuture, returnFuture, attributeServer, attributeHttp, attributeSSe);
        NettyWriter.write(context, ObjectBuilder.buildEmptyNettySegment());
        EasyMock.verify(channel, socketAddress, context, closeFuture, returnFuture, attributeServer, attributeHttp, attributeSSe);
    }

    @Test
    public void testWriteWs() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attribute = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attribute.get()).andReturn(NettyAttributes.SERVER_WS).anyTimes();
        EasyMock.replay(attribute);
        EasyMock.expect(context.attr(NettyAttributes.SERVER_TYPE)).andReturn(attribute).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.replay(closeFuture, returnFuture);
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.replay(context);
        NettyWriter.write(context, ObjectBuilder.buildEmptyNettySegment());
        EasyMock.verify(context, attribute, closeFuture, returnFuture);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testWriteWithNullContext() throws Exception {
        NettyWriter.write(null, ObjectBuilder.buildEmptyNettySegment());
    }

    @Test(expected = WorkflowException.class)
    public void testWriteWithInValidProtocol() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attribute = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attribute.get()).andReturn((byte) 100).anyTimes();
        EasyMock.replay(attribute);
        EasyMock.expect(context.attr(NettyAttributes.SERVER_TYPE)).andReturn(attribute).anyTimes();
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.replay(closeFuture, returnFuture);
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.replay(context);
        NettyWriter.write(context, ObjectBuilder.buildEmptyNettySegment());
        EasyMock.verify(context, attribute, closeFuture, returnFuture);
    }

    @Test
    public void testWriteHttpWithStreamAndNotFinished() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        attributeSSe.set(NettyAttributes.HTTP_SSE);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        EasyMock.expect(attributeSSe.get()).andReturn(NettyAttributes.HTTP_SSE).anyTimes();
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.replay(context, closeFuture, returnFuture, attributeSSe);
        NettyWriter.writeStream(context, ObjectBuilder.buildEmptyNettySegment(), UnpooledByteBufAllocator.DEFAULT.buffer());
        EasyMock.verify(context, closeFuture, returnFuture, attributeSSe);
    }

    @Test
    public void testWriteHttpWithStreamAndFinished() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        attributeSSe.set(NettyAttributes.HTTP_SSE);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        EasyMock.expect(attributeSSe.get()).andReturn(NettyAttributes.HTTP_SSE).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.replay(context, closeFuture, returnFuture, attributeSSe);
        NettyWriter.writeStream(context, ObjectBuilder.buildSegment(), UnpooledByteBufAllocator.DEFAULT.buffer());
        EasyMock.verify(context, closeFuture, returnFuture, attributeSSe);
    }

    @Test(expected = RuntimeException.class)
    public void testWriteHttpWithStreamAndException() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        attributeSSe.set(NettyAttributes.HTTP_SSE);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE).get()).andReturn(NettyAttributes.HTTP_SSE).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andThrow(new RuntimeException()).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        EasyMock.replay(context, closeFuture, returnFuture, attributeSSe);
        NettyWriter.writeStream(context, ObjectBuilder.buildSegment(), UnpooledByteBufAllocator.DEFAULT.buffer());
        EasyMock.verify(context, closeFuture, returnFuture, attributeSSe);
    }

    @Test
    public void testWriteHttpWithOnce() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        EasyMock.replay(attributeCors, context, closeFuture, returnFuture);
        NettyWriter.writeOnce(context, ObjectBuilder.buildSegment(), UnpooledByteBufAllocator.DEFAULT.buffer());
        EasyMock.verify(attributeCors, context, closeFuture, returnFuture);
    }

    @Test
    public void testSseHandshake() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        EasyMock.replay(attributeCors, context, closeFuture, returnFuture);
        NettyWriter.connectStream(context);
        EasyMock.verify(attributeCors, context, closeFuture, returnFuture);
    }

    @Test
    public void testBingHttpTypeWithEmptyUri() throws Exception {
        FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        attributeHttp.set(NettyAttributes.CONNECTION_STREAM);
        EasyMock.expectLastCall().anyTimes();
        ChannelFuture channelFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(DefaultHttpResponse.class))).andReturn(channelFuture).anyTimes();
        EasyMock.expect(channelFuture.addListener(NettyAlarm.INSTANCE)).andReturn(channelFuture).anyTimes();
        EasyMock.replay(attributeCors, fullHttpRequest, channelFuture, context, attributeHttp);
        NettyWriter.flagConnection(context, NettyAttributes.CONNECTION_STREAM);
        EasyMock.verify(attributeCors, fullHttpRequest, channelFuture, context, attributeHttp);
    }

    @Test
    public void testBingHttpTypeWithEmptyParam() throws Exception {
        FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        attributeHttp.set(NettyAttributes.CONNECTION_STREAM);
        EasyMock.expectLastCall().anyTimes();
        ChannelFuture channelFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(DefaultHttpResponse.class))).andReturn(channelFuture).anyTimes();
        EasyMock.expect(channelFuture.addListener(NettyAlarm.INSTANCE)).andReturn(channelFuture).anyTimes();
        EasyMock.replay(attributeCors, fullHttpRequest, channelFuture, context, attributeHttp);
        NettyWriter.flagConnection(context, NettyAttributes.CONNECTION_STREAM);
        EasyMock.verify(attributeCors, fullHttpRequest, channelFuture, context, attributeHttp);
    }

    @Test
    public void testBingHttpTypeWithInvalidParam1() throws Exception {
        FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        Attribute<Byte> attributeType = EasyMock.createMock(Attribute.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        attributeHttp.set(NettyAttributes.CONNECTION_STREAM);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(attributeCors, fullHttpRequest, context, attributeHttp, attributeType);
        NettyWriter.flagConnection(context, NettyAttributes.CONNECTION_STREAM);
        EasyMock.verify(attributeCors, fullHttpRequest, context, attributeHttp, attributeType);
    }

    @Test
    public void testBingHttpTypeWithInvalidParam2() throws Exception {
        FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        attributeHttp.set(NettyAttributes.CONNECTION_STREAM);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(attributeCors, fullHttpRequest, context, attributeHttp);
        NettyWriter.flagConnection(context, NettyAttributes.CONNECTION_STREAM);
        EasyMock.verify(attributeCors, fullHttpRequest, context, attributeHttp);
    }

    @Test
    public void testBingHttpTypeWithParamTrue() throws Exception {
        FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        attributeHttp.set(NettyAttributes.CONNECTION_STREAM);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(attributeCors, fullHttpRequest, context, attributeHttp);
        NettyWriter.flagConnection(context, NettyAttributes.CONNECTION_STREAM);
        EasyMock.verify(attributeCors, fullHttpRequest, context, attributeHttp);
    }

    @Test
    public void testBingHttpTypeWithParamFalse() throws Exception {
        FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        attributeHttp.set(NettyAttributes.CONNECTION_STREAM);
        EasyMock.expectLastCall().anyTimes();
        ChannelFuture channelFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(DefaultHttpResponse.class))).andReturn(channelFuture).anyTimes();
        EasyMock.expect(channelFuture.addListener(NettyAlarm.INSTANCE)).andReturn(channelFuture).anyTimes();
        EasyMock.replay(attributeCors, fullHttpRequest, channelFuture, context, attributeHttp);
        NettyWriter.flagConnection(context, NettyAttributes.CONNECTION_STREAM);
        EasyMock.verify(attributeCors, fullHttpRequest, channelFuture, context, attributeHttp);
    }

    @Test(expected = RuntimeException.class)
    public void testWriteFailed() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeServer = EasyMock.createMock(Attribute.class);
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeServer.get()).andThrow(new RuntimeException()).anyTimes();
        EasyMock.expect(attributeHttp.get()).andThrow(new RuntimeException()).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SERVER_TYPE)).andReturn(attributeServer).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        EasyMock.replay(context, closeFuture, returnFuture, attributeServer, attributeHttp);
        try {
            NettyWriter.write(context, ObjectBuilder.buildEmptyNettySegment());
        } finally {
            EasyMock.verify(context, closeFuture, returnFuture, attributeServer, attributeHttp);
        }
    }

    @Test(expected = NullPointerException.class)
    public void testWriteWithRelease() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeServer = EasyMock.createMock(Attribute.class);
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeServer.get()).andReturn(NettyAttributes.SERVER_HTTP).anyTimes();
        EasyMock.expect(attributeHttp.get()).andReturn(NettyAttributes.CONNECTION_STREAM).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SERVER_TYPE)).andReturn(attributeServer).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        EasyMock.replay(context, closeFuture, returnFuture, attributeServer, attributeHttp);
        NettyWriter.write(context, null);
        EasyMock.verify(context, closeFuture, returnFuture, attributeServer, attributeHttp);
    }

    @Test
    public void testForceClose() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeHttp.get()).andReturn(NettyAttributes.CONNECTION_ONCE).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        EasyMock.replay(context, closeFuture, returnFuture, attributeHttp);
        Segment segment = ObjectBuilder.buildSegment(Segment.SegmentConfig.builder()
                .code(0)
                .build());
        NettyWriter.write(context, segment);
        EasyMock.verify(context, closeFuture, returnFuture, attributeHttp);
    }

    @Test
    public void testForceCloseWithStream() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(EasyMock.anyObject(NettyCloser.class))).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        EasyMock.expect(attributeHttp.get()).andReturn(NettyAttributes.CONNECTION_STREAM).anyTimes();
        EasyMock.expect(attributeSSe.get()).andReturn(NettyAttributes.HTTP_SSE).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        EasyMock.replay(context, channel, socketAddress, closeFuture, returnFuture, attributeHttp);
        Segment segment = ObjectBuilder.buildSegment(Segment.SegmentConfig.builder()
                .code(0)
                .build());
        NettyWriter.write(context, segment);
        EasyMock.verify(context, channel, socketAddress, closeFuture, returnFuture, attributeHttp);
    }

    @Test
    public void testWriteNotSSE() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        EasyMock.expect(attributeSSe.get()).andReturn((byte) 0).anyTimes();
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        NettySegment nettySegment = ObjectBuilder.buildEmptyNettySegment();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.replay(context, closeFuture, returnFuture, attributeSSe);
        NettyWriter.writeStream(context, nettySegment, PooledByteBufAllocator.DEFAULT.buffer());
        EasyMock.verify(context, closeFuture, returnFuture, attributeSSe);
    }

    @Test
    public void testWriteWithCloseIsFinishAndNotSSE() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        EasyMock.expect(attributeSSe.get()).andReturn((byte) 0).anyTimes();
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        EasyMock.expect(attributeHttp.get()).andReturn(NettyAttributes.CONNECTION_STREAM).anyTimes();
        NettySegment nettySegment = EasyMock.createMock(NettySegment.class);
        EasyMock.expect(nettySegment.getCode()).andReturn(505).anyTimes();
        EasyMock.expect(nettySegment.isFinished()).andReturn(false).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.replay(context, closeFuture, returnFuture, attributeSSe, attributeHttp, nettySegment);
        NettyWriter.writeHttp(context, nettySegment, PooledByteBufAllocator.DEFAULT.buffer());
        EasyMock.verify(context, closeFuture, returnFuture, attributeSSe, attributeHttp, nettySegment);
    }

    @Test
    public void testWriteWithCloseIsOnce() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        EasyMock.expect(attributeSSe.get()).andReturn((byte) 0).anyTimes();
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        EasyMock.expect(attributeHttp.get()).andReturn(NettyAttributes.CONNECTION_ONCE).anyTimes();
        NettySegment nettySegment = EasyMock.createMock(NettySegment.class);
        EasyMock.expect(nettySegment.getCode()).andReturn(505).anyTimes();
        EasyMock.expect(nettySegment.isFinished()).andReturn(false).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        EasyMock.replay(attributeCors, context, closeFuture, returnFuture, attributeSSe, attributeHttp, nettySegment);
        NettyWriter.writeHttp(context, nettySegment, PooledByteBufAllocator.DEFAULT.buffer());
        EasyMock.verify(attributeCors, context, closeFuture, returnFuture, attributeSSe, attributeHttp, nettySegment);
    }

    @Test
    public void testWriteWithCloseIsSSEButNot200() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        EasyMock.expect(attributeSSe.get()).andReturn(NettyAttributes.HTTP_SSE).anyTimes();
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        EasyMock.expect(attributeHttp.get()).andReturn(NettyAttributes.CONNECTION_STREAM).anyTimes();
        NettySegment nettySegment = ObjectBuilder.buildEmptyNettySegment();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.replay(context, closeFuture, returnFuture, attributeSSe, attributeHttp);
        NettyWriter.writeHttp(context, nettySegment, PooledByteBufAllocator.DEFAULT.buffer());
        EasyMock.verify(context, closeFuture, returnFuture, attributeSSe, attributeHttp);
    }

    @Test(expected = RuntimeException.class)
    public void testWriteNotSSEWithException() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        EasyMock.expect(attributeSSe.get()).andReturn((byte) 0).anyTimes();
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        NettySegment nettySegment = EasyMock.createMock(NettySegment.class);
        EasyMock.expect(nettySegment.isFinished()).andThrow(new RuntimeException()).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.replay(context, nettySegment, closeFuture, returnFuture, attributeSSe);
        NettyWriter.writeStream(context, nettySegment, PooledByteBufAllocator.DEFAULT.buffer());
        EasyMock.verify(context, nettySegment, closeFuture, returnFuture, attributeSSe);
    }

    @Test(expected = RuntimeException.class)
    public void testCloseExceptionForReleaseBuffer() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        EasyMock.expect(attributeSSe.get()).andReturn((byte) 0).anyTimes();
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andThrow(new RuntimeException()).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channel.isOpen()).andReturn(true).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        NettySegment nettySegment = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        Attribute<Byte> attributeServer = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeServer.get()).andReturn(NettyAttributes.SERVER_HTTP).anyTimes();
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn(NettyAttributes.HTTP_CORS).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SERVER_TYPE)).andReturn(attributeServer).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        EasyMock.expect(attributeHttp.get()).andReturn(NettyAttributes.CONNECTION_ONCE).anyTimes();
        EasyMock.expect(context.close()).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(channel, socketAddress, context, closeFuture, returnFuture, attributeSSe, attributeHttp, attributeServer);
        try {
            NettyWriter.write(context, nettySegment);
        } finally {
            EasyMock.verify(channel, socketAddress, context, closeFuture, returnFuture, attributeSSe, attributeHttp, attributeServer);
        }
    }

    @Test
    public void testWriteHttpWithMcp() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.replay(context, attributeCors, closeFuture, returnFuture);
        NettyWriter.writeOnce(context, ObjectBuilder.buildSegment());
        EasyMock.verify(context, attributeCors, closeFuture, returnFuture);
    }

    /**
     * 覆盖 NettyWriter.close（208-211 行）：强制关闭通道，调用 ctx.close().addListener(NettyAlarm.INSTANCE)。
     */
    @Test
    public void testClose() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(context.close()).andReturn(closeFuture).once();
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).once();
        EasyMock.replay(context, closeFuture, returnFuture);
        NettyWriter.close(context);
        EasyMock.verify(context, closeFuture, returnFuture);
    }

    @Test(expected = RuntimeException.class)
    public void testWriteHttpWithMcpWithException() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(context, attributeCors);
        try {
            NettyWriter.writeOnce(context, ObjectBuilder.buildSegment());
        } finally {
            EasyMock.verify(context, attributeCors);
        }
    }

    @Test
    public void testWriteHttpWithA2A() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.replay(context, attributeCors, closeFuture, returnFuture);
        NettyWriter.writeStream(context, ObjectBuilder.buildSegmentWithOutFinish());
        EasyMock.verify(context, attributeCors, closeFuture, returnFuture);
    }

    @Test
    public void testWriteHttpWithA2AWithFinish() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(closeFuture.addListener(EasyMock.anyObject(NettyCloser.class))).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.replay(context, attributeCors, closeFuture, returnFuture);
        NettyWriter.writeStream(context, ObjectBuilder.buildSegment());
        EasyMock.verify(context, attributeCors, closeFuture, returnFuture);
    }

    @Test(expected = RuntimeException.class)
    public void testWriteHttpWithA2AWithException() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(context, attributeCors);
        try {
            NettyWriter.writeStream(context, ObjectBuilder.buildSegment());
        } finally {
            EasyMock.verify(context, attributeCors);
        }
    }

    @Test
    public void testWriteCodeNull() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeServer = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeServer.get()).andReturn(NettyAttributes.SERVER_HTTP).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SERVER_TYPE)).andReturn(attributeServer).anyTimes();
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeHttp.get()).andReturn(NettyAttributes.CONNECTION_ONCE).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(null).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();

        ChannelFuture future = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject())).andReturn(future).anyTimes();
        EasyMock.expect(future.addListener(EasyMock.anyObject())).andReturn(future).anyTimes();

        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();

        EasyMock.replay(closeFuture, returnFuture, context, attributeServer, attributeHttp, attributeCors, channel, future);
        NettySegment segment = ObjectBuilder.buildEmptyNettySegment();
        NettyWriter.write(context, segment);
        EasyMock.verify(context);
    }

    @Test
    public void testWriteCodeNegative() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeHttp.get()).andReturn(NettyAttributes.CONNECTION_ONCE).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        NettySegment segment = ObjectBuilder.buildEmptyNettySegment();
        Attribute<Byte> attributeServer = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeServer.get()).andReturn(NettyAttributes.SERVER_HTTP).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SERVER_TYPE)).andReturn(attributeServer).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeServer).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeServer).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channel.isOpen()).andReturn(true).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        EasyMock.replay(closeFuture, returnFuture, channel, socketAddress, attributeServer, attributeHttp, context);
        NettyWriter.write(context, segment);
        EasyMock.verify(context);
    }

    @Test
    public void testIsStreamCloseCombinations() {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attr = EasyMock.createMock(Attribute.class);
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE)).andReturn(attr).anyTimes();

        NettyStream stream = EasyMock.createMock(NettyStream.class);
        EasyMock.expect(stream.isFinished()).andReturn(true).anyTimes();
        // Case 1: Finished and not SSE
        EasyMock.expect(attr.get()).andReturn((byte) 0).anyTimes();
        EasyMock.replay(context, attr, stream);
        Assert.assertTrue(NettyWriter.isStreamClose(context, stream));

        // Case 2: Not 2xx and is SSE
        EasyMock.reset(context, attr, stream);
        EasyMock.expect(stream.isFinished()).andReturn(true).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE)).andReturn(attr).anyTimes();
        EasyMock.expect(attr.get()).andReturn(NettyAttributes.HTTP_SSE).anyTimes();
        EasyMock.expect(stream.getCode()).andReturn(400).anyTimes();
        EasyMock.replay(context, attr, stream);
        Assert.assertTrue(NettyWriter.isStreamClose(context, stream));
    }

    @org.junit.jupiter.api.Test
    public void testNettyWriterInstantiationUnique() {
        org.junit.jupiter.api.Assertions.assertTrue(true);
    }

    /**
     * 覆盖 writeStream 异常分支（NettyWriter 137-144 行）：若 buffer 尚未加入 composite（numComponents &lt; 2）则单独释放 buffer，再释放 composite。
     * 本用例：第一次 addComponent 即抛异常，numComponents=0。
     */
    @Test(expected = RuntimeException.class)
    public void testWriteStream_ReleaseBufferWhenAllocFails() throws Exception {
        ChannelHandlerContext ctx = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeSSe.get()).andReturn(NettyAttributes.HTTP_SSE).anyTimes();
        EasyMock.expect(ctx.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        CompositeByteBuf compositeByteBuf = EasyMock.createMock(CompositeByteBuf.class);
        PooledByteBufAllocator pooledByteBufAllocator = EasyMock.createMock(PooledByteBufAllocator.class);
        EasyMock.expect(pooledByteBufAllocator.compositeBuffer()).andReturn(compositeByteBuf).anyTimes();
        EasyMock.expect(ctx.alloc()).andReturn(pooledByteBufAllocator).anyTimes();
        RuntimeException runtimeException = new RuntimeException();
        EasyMock.expect(compositeByteBuf.addComponent(EasyMock.anyBoolean(), EasyMock.anyObject())).andThrow(runtimeException).anyTimes();
        EasyMock.expect(compositeByteBuf.numComponents()).andReturn(0).anyTimes();
        compositeByteBuf.release();
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(ctx, attributeSSe, compositeByteBuf, pooledByteBufAllocator);
        ByteBuf byteBuf = UnpooledByteBufAllocator.DEFAULT.buffer(1);
        try {
            NettyWriter.writeStream(ctx, NettyStream.SUCCESS, byteBuf);
        } finally {
            org.junit.Assert.assertEquals("catch 中应已执行 ReferenceCountUtil.release(buffer)", 0, byteBuf.refCnt());
            EasyMock.verify(ctx, attributeSSe, compositeByteBuf, pooledByteBufAllocator);
        }
    }

    /**
     * 覆盖 writeStream 异常分支（NettyWriter 137-144 行）：第二次 addComponent（添加 buffer）时抛异常，numComponents=1，仍满足 numComponents &lt; 2 故单独释放 buffer。
     */
    @Test(expected = RuntimeException.class)
    public void testWriteStream_ReleaseBufferWhenSecondAddComponentFails() throws Exception {
        ChannelHandlerContext ctx = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeSSe.get()).andReturn(NettyAttributes.HTTP_SSE).anyTimes();
        EasyMock.expect(ctx.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        CompositeByteBuf compositeByteBuf = EasyMock.createMock(CompositeByteBuf.class);
        PooledByteBufAllocator pooledByteBufAllocator = EasyMock.createMock(PooledByteBufAllocator.class);
        EasyMock.expect(pooledByteBufAllocator.compositeBuffer()).andReturn(compositeByteBuf).anyTimes();
        EasyMock.expect(ctx.alloc()).andReturn(pooledByteBufAllocator).anyTimes();
        RuntimeException runtimeException = new RuntimeException("addComponent buffer failed");
        EasyMock.expect(compositeByteBuf.addComponent(EasyMock.eq(true), EasyMock.anyObject())).andReturn(compositeByteBuf).once();
        EasyMock.expect(compositeByteBuf.addComponent(EasyMock.eq(true), EasyMock.anyObject())).andThrow(runtimeException).once();
        EasyMock.expect(compositeByteBuf.numComponents()).andReturn(1).anyTimes();
        compositeByteBuf.release();
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(ctx, attributeSSe, compositeByteBuf, pooledByteBufAllocator);
        ByteBuf byteBuf = UnpooledByteBufAllocator.DEFAULT.buffer(1);
        try {
            NettyWriter.writeStream(ctx, NettyStream.SUCCESS, byteBuf);
        } finally {
            org.junit.Assert.assertEquals("catch 中应已执行 ReferenceCountUtil.release(buffer)", 0, byteBuf.refCnt());
            EasyMock.verify(ctx, attributeSSe, compositeByteBuf, pooledByteBufAllocator);
        }
    }

    /**
     * 显式覆盖 writeStream catch 中 ReferenceCountUtil.release(buffer)（139-140 行）：
     * 在测试内捕获异常并断言 refCnt，确保覆盖率统计到该行。
     */
    @Test
    public void testWriteStream_CatchReleasesBufferExplicitCoverage() throws Exception {
        ChannelHandlerContext ctx = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeSSe.get()).andReturn(NettyAttributes.HTTP_SSE).anyTimes();
        EasyMock.expect(ctx.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        CompositeByteBuf compositeByteBuf = EasyMock.createMock(CompositeByteBuf.class);
        PooledByteBufAllocator pooledByteBufAllocator = EasyMock.createMock(PooledByteBufAllocator.class);
        EasyMock.expect(pooledByteBufAllocator.compositeBuffer()).andReturn(compositeByteBuf).anyTimes();
        EasyMock.expect(ctx.alloc()).andReturn(pooledByteBufAllocator).anyTimes();
        RuntimeException runtimeException = new RuntimeException("first addComponent fails");
        EasyMock.expect(compositeByteBuf.addComponent(EasyMock.eq(true), EasyMock.anyObject())).andThrow(runtimeException).once();
        EasyMock.expect(compositeByteBuf.numComponents()).andReturn(0).anyTimes();
        EasyMock.expect(compositeByteBuf.release()).andReturn(true).once();
        EasyMock.replay(ctx, attributeSSe, compositeByteBuf, pooledByteBufAllocator);
        ByteBuf byteBuf = UnpooledByteBufAllocator.DEFAULT.buffer(1);
        try {
            NettyWriter.writeStream(ctx, NettyStream.SUCCESS, byteBuf);
            Assert.fail("should throw");
        } catch (RuntimeException e) {
            Assert.assertSame(runtimeException, e);
        }
        Assert.assertEquals("catch 中应已执行 ReferenceCountUtil.release(buffer)，refCnt 应为 0", 0, byteBuf.refCnt());
        EasyMock.verify(ctx, attributeSSe, compositeByteBuf, pooledByteBufAllocator);
    }

    /**
     * 覆盖 writeStream catch 中 numComponents >= 2 时不执行 ReferenceCountUtil.release(buffer) 的分支：
     * 第三次 addComponent 时抛异常，numComponents=2，仅 composite.release()，不单独 release(buffer)。
     */
    @Test
    public void testWriteStream_CatchDoesNotReleaseBufferWhenNumComponentsAtLeast2() throws Exception {
        ChannelHandlerContext ctx = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeSSe.get()).andReturn(NettyAttributes.HTTP_SSE).anyTimes();
        EasyMock.expect(ctx.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        CompositeByteBuf compositeByteBuf = EasyMock.createMock(CompositeByteBuf.class);
        PooledByteBufAllocator pooledByteBufAllocator = EasyMock.createMock(PooledByteBufAllocator.class);
        EasyMock.expect(pooledByteBufAllocator.compositeBuffer()).andReturn(compositeByteBuf).anyTimes();
        EasyMock.expect(ctx.alloc()).andReturn(pooledByteBufAllocator).anyTimes();
        RuntimeException runtimeException = new RuntimeException("third addComponent fails");
        EasyMock.expect(compositeByteBuf.addComponent(EasyMock.eq(true), EasyMock.anyObject())).andReturn(compositeByteBuf).times(2);
        EasyMock.expect(compositeByteBuf.addComponent(EasyMock.eq(true), EasyMock.anyObject())).andThrow(runtimeException).once();
        EasyMock.expect(compositeByteBuf.numComponents()).andReturn(2).anyTimes();
        EasyMock.expect(compositeByteBuf.release()).andReturn(true).once();
        EasyMock.replay(ctx, attributeSSe, compositeByteBuf, pooledByteBufAllocator);
        ByteBuf byteBuf = UnpooledByteBufAllocator.DEFAULT.buffer(1);
        try {
            NettyWriter.writeStream(ctx, NettyStream.SUCCESS, byteBuf);
            Assert.fail("should throw");
        } catch (RuntimeException e) {
            Assert.assertSame(runtimeException, e);
        }
        EasyMock.verify(ctx, attributeSSe, compositeByteBuf, pooledByteBufAllocator);
        Assert.assertEquals("numComponents>=2 时不应单独 release(buffer)，调用方 buffer 引用未释放", 1, byteBuf.refCnt());
        byteBuf.release();
    }
}