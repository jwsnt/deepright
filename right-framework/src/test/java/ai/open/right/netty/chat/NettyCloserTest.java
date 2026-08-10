package ai.open.right.netty.chat;

import ai.open.right.netty.NettyCloser;
import io.netty.channel.*;
import io.netty.util.concurrent.Future;
import org.easymock.EasyMock;
import org.junit.Test;

import java.net.SocketAddress;

public class NettyCloserTest {

    @Test
    public void testClose() throws Exception {
        ChannelHandlerContext channelHandlerContext = EasyMock.createMock(ChannelHandlerContext.class);
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channel.isOpen()).andReturn(true).anyTimes();
        EasyMock.expect(channelHandlerContext.channel()).andReturn(channel).anyTimes();
        ChannelFuture channelFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(channelHandlerContext.close()).andReturn(channelFuture).anyTimes();
        NettyCloser nettyCloser = new NettyCloser(channelHandlerContext);
        Future<Void> future = EasyMock.createMock(Future.class);
        EasyMock.expect(future.isSuccess()).andReturn(false).anyTimes();
        EasyMock.expect(future.cause()).andReturn(new Throwable()).anyTimes();
        EasyMock.replay(socketAddress, future, channelHandlerContext, channel);
        nettyCloser.operationComplete(future);
        EasyMock.verify(socketAddress, channel, future);
    }
}
