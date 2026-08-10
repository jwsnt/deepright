package ai.open.right.netty.mcp.server;

import ai.open.right.netty.NettyHandler;
import ai.open.right.netty.mcp.NettyInputProxy;
import ai.open.right.netty.mcp.distribute.NettyDistributor;
import io.netty.channel.ChannelHandler.Sharable;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.FullHttpRequest;
import io.netty.util.ReferenceCountUtil;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

/**
 * @author shenjiawei
 */
@Sharable
@Slf4j
@Setter
@Getter
public class NettyMcpHandler extends NettyHandler {

    protected NettyDistributor distributor;

    @Override
    public void channelRead(ChannelHandlerContext ctx, Object msg) throws Exception {
        try {
            // 由distributor触发资源释放
            this.distributor.distribute(ctx, this.buildInputProxy(msg), this);
            if (log.isDebugEnabled()) {
                log.debug("Channel read success={}", ctx.channel().remoteAddress());
            }
        } catch (Exception e) {
            // ChannelRead抛出的异常会传给Pipeline下一个Handler，不会触发本Handler.exceptionCaught，显式调用
            this.exceptionCaught(ctx, e);
        } finally {
            // BuildInputProxy内会Retain，此处释放调用方引用，避免泄漏
            ReferenceCountUtil.release(msg);
        }
    }

    protected NettyInputProxy buildInputProxy(Object msg) throws Exception {
        return new NettyInputProxy(FullHttpRequest.class.cast(msg));
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected NettyDistributor distributor;

        @Bean
        @ConditionalOnMissingBean(value = NettyMcpHandler.class)
        public NettyMcpHandler nettyMcpHandler() throws Exception {
            NettyMcpHandler nettyHandler = new NettyMcpHandler();
            BeanUtils.copyProperties(this, nettyHandler);
            log.info("NettyChatHandler inited");
            return nettyHandler;
        }
    }
}
