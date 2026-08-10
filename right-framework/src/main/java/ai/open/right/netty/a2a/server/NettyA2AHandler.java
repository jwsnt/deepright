package ai.open.right.netty.a2a.server;

import ai.open.right.netty.NettyHandler;
import ai.open.right.netty.a2a.NettyInputProxy;
import ai.open.right.netty.a2a.distribute.NettyDistributor;
import io.netty.channel.ChannelHandler.Sharable;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.FullHttpRequest;
import io.netty.util.ReferenceCountUtil;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
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
public class NettyA2AHandler extends NettyHandler {

    protected NettyDistributor distributor;

    protected String autoDump;

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
        return new NettyInputProxy(FullHttpRequest.class.cast(msg), this.autoDump);
    }
    @Configuration
    @ConditionalOnProperty(name = "a2a.enable", havingValue = "true", matchIfMissing = false)
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected NettyDistributor distributor;

        @Value("${autodump.http:}")
        protected String autoDump;

        @Bean
        @ConditionalOnMissingBean(value = NettyA2AHandler.class)
        public NettyA2AHandler nettyA2AHandler() throws Exception {
            NettyA2AHandler nettyHandler = new NettyA2AHandler();
            BeanUtils.copyProperties(this, nettyHandler);
            log.info("NettyA2AHandler inited");
            return nettyHandler;
        }
    }
}
