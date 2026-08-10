package ai.open.right.netty.chat.server.http;

import ai.open.right.WorkflowException;
import ai.open.right.netty.NettyCloser;
import ai.open.right.netty.NettyWriter;
import ai.open.right.netty.chat.NettyChatHandler;
import ai.open.right.netty.chat.NettyInputProxy;
import ai.open.right.netty.chat.distribute.NettyDistributor;
import ai.open.right.netty.chat.server.NettyAttributes;
import ai.open.right.protocol.ProtocolCode;
import io.netty.buffer.ByteBuf;
import io.netty.channel.ChannelHandler.Sharable;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.FullHttpRequest;
import io.netty.handler.codec.http.HttpHeaders;
import io.netty.util.ReferenceCountUtil;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.RegExUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

/**
 * @author shenjiawei
 */
@Sharable
@Slf4j
@Setter
@Getter
public class NettyHttpHandler extends NettyChatHandler {

    @Override
    protected NettyInputProxy buildInputProxy(ChannelHandlerContext ctx, Object source, ByteBuf byteBuf) throws Exception {
        FullHttpRequest fullHttpRequest = FullHttpRequest.class.cast(source);
        // 绑定通道模式(WS/HTTP)
        NettyWriter.flagServer(ctx, this.type());
        // 是否开启SSE通道
        this.flagSse(ctx, fullHttpRequest.headers());
        String token = this.token(fullHttpRequest);
        return new NettyHttpProxy(fullHttpRequest.headers(), byteBuf, token, this.autoDump);
    }

    @Override
    // Netty异常处理
    public void exceptionCaught(ChannelHandlerContext ctx, Throwable cause) throws Exception {
        // 写入500和错误内容后关闭通道
        if (log.isInfoEnabled()) {
            log.info(cause.getMessage(), cause);
        }
        boolean exposure = Exception.class.isAssignableFrom(cause.getClass()) && WorkflowException.exposure(Exception.class.cast(cause));
        NettyWriter.write(ctx, NettyErrorSegment.builder()
                .code(exposure ? WorkflowException.code(Exception.class.cast(cause)) : ProtocolCode.C500)
                .content(exposure ? cause.getMessage() : "Internal Server Error")
                .build());
        ctx.close().addListener(new NettyCloser(ctx));
    }

    // 在通道绑定Cors类型的标记
    protected void flagSse(ChannelHandlerContext ctx, HttpHeaders header) throws Exception {
        if (StringUtils.containsIgnoreCase(header.get("Accept"), "text/event-stream")) {
            NettyWriter.flagSse(ctx);
        }
    }

    @Override
    protected ByteBuf byteBuf(ChannelHandlerContext ctx, Object source) throws Exception {
        ByteBuf buffer = null;
        try {
            buffer = FullHttpRequest.class.cast(source).content().retain();
            // 请求不能为空（假定没有Get请求）
            Assert.isTrue(buffer.readableBytes() > 0, "Http request body can not be empty");
            // 增加引用计数
            return buffer;
        } catch (Exception e) {
            if (buffer != null && buffer.refCnt() > 0) {
                // 异常释放所有引用，仅释放自己（不要多次释放source）
                ReferenceCountUtil.release(buffer);
            }
            throw e;
        } finally {
            // 释放原始计数
            ReferenceCountUtil.release(source);
        }
    }

    // 从Header获取Token
    protected String token(FullHttpRequest fullHttpRequest) throws Exception {
        String token = StringUtils.trim(RegExUtils.replaceFirst(fullHttpRequest.headers().get("Authorization"), "Bearer", ""));
        if (StringUtils.isEmpty(token)) {
            throw new WorkflowException("Token can not be empty. Please check the Header `Authorization`", ProtocolCode.C401);
        }
        return token;
    }


    @Override
    protected Byte type() {
        return NettyAttributes.SERVER_HTTP;
    }

    @ConditionalOnProperty(name = "chat.http.enable", havingValue = "true", matchIfMissing = true)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected NettyDistributor distributor;

        @Value("${autodump.http:}")
        protected String autoDump;

        @Bean
        @ConditionalOnMissingBean(value = NettyHttpHandler.class)
        public NettyHttpHandler nettyHttpHandler() throws Exception {
            NettyHttpHandler nettyHttpHandler = new NettyHttpHandler();
            BeanUtils.copyProperties(this, nettyHttpHandler);
            log.info("NettyHttpHandler inited");
            return nettyHttpHandler;
        }
    }
}
