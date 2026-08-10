package ai.open.right.netty.chat.server.ws;

import ai.open.right.netty.chat.NettyChatHandler;
import ai.open.right.netty.chat.distribute.NettyDistributor;
import ai.open.right.netty.chat.server.NettyAttributes;
import io.netty.buffer.ByteBuf;
import io.netty.channel.ChannelHandler.Sharable;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.websocketx.TextWebSocketFrame;
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
public class NettyWsHandler extends NettyChatHandler {

    @Override
    protected ByteBuf byteBuf(ChannelHandlerContext ctx, Object source) throws Exception {
        ByteBuf buffer = null;
        try {
            buffer = TextWebSocketFrame.class.cast(source).content();
            if (buffer.readableBytes() == 0) {
                // 空格保持连接（心跳）
                if (log.isDebugEnabled()) {
                    log.debug("Heartbeat message={}", ctx.channel().remoteAddress());
                }
                return null;
            }
            // 增加引用计数，供调用方持有
            return buffer.retain();
        } finally {
            // 释放原始计数
            ReferenceCountUtil.release(source);
        }
    }

    @Override
    protected Byte type() {
        return NettyAttributes.SERVER_WS;
    }

    @ConditionalOnProperty(name = "chat.ws.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected NettyDistributor distributor;

        @Value("${autodump.http:}")
        protected String autoDump;

        @Bean
        @ConditionalOnMissingBean(value = NettyWsHandler.class)
        public NettyWsHandler nettyWsHandler() throws Exception {
            NettyWsHandler nettyWsHandler = new NettyWsHandler();
            BeanUtils.copyProperties(this, nettyWsHandler);
            log.info("NettyWsHandler inited");
            return nettyWsHandler;
        }
    }
}
