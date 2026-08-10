package ai.open.right.netty;

import io.netty.channel.ChannelHandlerContext;

// Handler异常回调
public interface NettyCaptor {

    public void exceptionCaught(ChannelHandlerContext ctx, Throwable cause) throws Exception;
}
