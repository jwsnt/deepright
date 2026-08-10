package ai.open.right.netty;

import io.netty.channel.ChannelHandlerContext;
import io.netty.util.concurrent.Future;
import lombok.extern.slf4j.Slf4j;

/**
 * 写入事件后关闭连接
 *
 * @author shenjiawei
 */
@Slf4j
public class NettyCloser extends NettyAlarm {

    protected final ChannelHandlerContext context;

    public NettyCloser(ChannelHandlerContext context) {
        super();
        this.context = context;
    }

    @Override
    public void operationComplete(Future<Void> future) throws Exception {
        super.operationComplete(future);
        if (this.context.channel().isOpen()) {
            this.context.close().addListener(NettyAlarm.INSTANCE);
            if (log.isDebugEnabled()) {
                log.debug("Close channel={}", this.context.channel().remoteAddress());
            }
        }
    }
}
