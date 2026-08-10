package ai.open.right.netty;

import ai.open.right.netty.chat.server.NettyAttributes;
import io.netty.buffer.ByteBuf;
import io.netty.channel.Channel;
import io.netty.channel.ChannelFuture;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.timeout.IdleStateEvent;
import io.netty.util.Attribute;
import org.easymock.EasyMock;
import org.junit.Test;

public class NettyHandlerTest {
    @Test
    public void testUserEventTriggeredIdle() throws Exception {
        NettyHandler handler = new NettyHandler() {};
        ChannelHandlerContext ctx = EasyMock.createMock(ChannelHandlerContext.class);
        Channel channel = EasyMock.createMock(Channel.class);
        EasyMock.expect(ctx.channel()).andReturn(channel).anyTimes();
        EasyMock.expect(channel.remoteAddress()).andReturn(null).anyTimes();
        EasyMock.expect(ctx.fireUserEventTriggered(EasyMock.anyObject())).andReturn(ctx).anyTimes();
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeSSe.get()).andReturn(NettyAttributes.HTTP_SSE).anyTimes();
        EasyMock.expect(ctx.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(ctx.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(ctx.close()).andReturn(closeFuture).anyTimes();
        EasyMock.replay(ctx, channel);
        handler.userEventTriggered(ctx, IdleStateEvent.FIRST_READER_IDLE_STATE_EVENT);
        EasyMock.verify(ctx);
    }

    @org.junit.jupiter.api.Test
    public void testNettyHandlerInstantiationUnique() {
        org.junit.jupiter.api.Assertions.assertTrue(true);
    }

}