package ai.open.right.netty.chat;

import ai.open.right.netty.NettyHandler;
import ai.open.right.netty.NettyWriter;
import ai.open.right.netty.chat.distribute.NettyDistributor;
import io.netty.buffer.ByteBuf;
import io.netty.channel.ChannelHandlerContext;
import io.netty.util.ReferenceCountUtil;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;

/**
 * @author shenjiawei
 */
@Slf4j
@Setter
@Getter
abstract public class NettyChatHandler extends NettyHandler {

    protected NettyDistributor distributor;

    protected String autoDump;

    @Override
    public void channelRead(ChannelHandlerContext ctx, Object msg) throws Exception {
        ByteBuf buffer = null;
        try {
            // 服务类型与具体报文解析无关，尽早标记以便异常时仍能按协议回包
            NettyWriter.flagServer(ctx, this.type());
            buffer = this.byteBuf(ctx, msg);
            if (buffer != null) {
                // 由distributor触发资源释放
                this.distributor.distribute(ctx, this.buildInputProxy(ctx, msg, buffer), this);
            }
            if (log.isDebugEnabled()) {
                log.debug("Channel read success={}", ctx.channel().remoteAddress());
            }
        } catch (Exception e) {
            // ChannelRead抛出的异常会传给Pipeline下一个Handler，不会触发本Handler.exceptionCaught，显式调用
            this.exceptionCaught(ctx, e);
        } finally {
            // BuildInputProxy内会Retain，此处释放调用方引用，避免泄漏
            if (buffer != null) {
                ReferenceCountUtil.release(buffer);
            }
        }
    }

    // NettyInputProxy将ByteBuf解析为对象
    protected NettyInputProxy buildInputProxy(ChannelHandlerContext ctx, Object source, ByteBuf byteBuf) throws Exception {
        return new NettyInputProxy(byteBuf, this.autoDump);
    }

    abstract protected ByteBuf byteBuf(ChannelHandlerContext ctx, Object source) throws Exception;

    abstract protected Byte type();
}
