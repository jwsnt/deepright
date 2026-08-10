package ai.open.right.netty;

import ai.open.right.WorkflowException;
import io.netty.channel.ChannelHandlerContext;
import io.netty.channel.ChannelInboundHandlerAdapter;
import io.netty.handler.timeout.IdleStateEvent;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;

/**
 * @author shenjiawei
 */
@Slf4j
@Setter
@Getter
abstract public class NettyHandler extends ChannelInboundHandlerAdapter implements NettyCaptor {

    // IDLE处理
    @Override
    public void userEventTriggered(ChannelHandlerContext ctx, Object evt) {
        if (evt instanceof IdleStateEvent) {
            if (log.isInfoEnabled()) {
                log.info("Channel will be closed by idle={}", ctx.channel().remoteAddress());
            }
            ctx.close().addListener(NettyAlarm.INSTANCE);
        }
        ctx.fireUserEventTriggered(evt);
    }

    @Override
    public void channelActive(ChannelHandlerContext ctx) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Channel active={}", ctx.channel().remoteAddress());
        }
        ctx.fireChannelActive();
    }

    @Override
    public void channelInactive(ChannelHandlerContext ctx) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Channel inactive={}", ctx.channel().remoteAddress());
        }
        ctx.fireChannelInactive();
    }

    @Override
    // 基础异常处理，实现类覆盖
    public void exceptionCaught(ChannelHandlerContext ctx, Throwable cause) throws Exception {
        if (WorkflowException.class.isAssignableFrom(cause.getClass())) {
            WorkflowException.class.cast(cause).dolog();
        } else {
            if (log.isDebugEnabled()) {
                log.debug(cause.getMessage(), cause);
            }
        }
        ctx.close().addListener(NettyAlarm.INSTANCE);
    }
}
