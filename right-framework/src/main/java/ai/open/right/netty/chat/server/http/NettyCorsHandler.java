package ai.open.right.netty.chat.server.http;

import ai.open.right.netty.NettyAlarm;
import ai.open.right.netty.NettyWriter;
import io.netty.channel.ChannelHandler.Sharable;
import io.netty.channel.ChannelHandlerContext;
import io.netty.channel.ChannelInboundHandlerAdapter;
import io.netty.handler.codec.http.*;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Sharable
@Slf4j
@Setter
@Getter
// Http跨域Handler
public class NettyCorsHandler extends ChannelInboundHandlerAdapter {

    @Override
    public void channelRead(ChannelHandlerContext ctx, Object msg) throws Exception {
        if (msg instanceof HttpRequest) {
            HttpRequest request = HttpRequest.class.cast(msg);
            if (request.method().name().equals(HttpMethod.OPTIONS.name())) {
                DefaultFullHttpResponse defaultHttpResponse = new DefaultFullHttpResponse(HttpVersion.HTTP_1_1, HttpResponseStatus.NO_CONTENT);
                // 在通道绑定Cors类型的标记
                NettyWriter.flagCors(ctx);
                // 写入跨域Headers
                NettyWriter.addCorsHeaders(ctx, defaultHttpResponse.headers());
                if (log.isDebugEnabled()) {
                    log.debug("Cors response headers={}", defaultHttpResponse.headers());
                }
                ctx.writeAndFlush(defaultHttpResponse).addListener(NettyAlarm.INSTANCE);
                return;
            }
        }
        ctx.fireChannelRead(msg);
    }

    @ConditionalOnProperty(name = "chat.http.cors.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Bean
        public NettyCorsHandler nettyCorsHandler() throws Exception {
            NettyCorsHandler nettyCorsHandler = new NettyCorsHandler();
            log.info("NettyCorsHandler inited");
            return nettyCorsHandler;
        }
    }
}
