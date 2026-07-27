package ai.deepright.module;

import ai.open.right.WorkflowException;
import ai.open.right.netty.NettyAlarm;
import ai.open.right.netty.NettyCloser;
import ai.open.right.netty.NettyWriter;
import ai.open.right.netty.chat.server.http.NettyHttpHandler;
import ai.open.right.workflow.flow.file.impl.SysStore;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.Unpooled;
import io.netty.channel.ChannelFutureListener;
import io.netty.channel.ChannelHandlerContext;
import io.netty.channel.DefaultFileRegion;
import io.netty.handler.codec.http.*;
import io.netty.handler.stream.ChunkedStream;
import io.netty.handler.timeout.IdleState;
import io.netty.handler.timeout.IdleStateEvent;
import io.netty.util.CharsetUtil;
import io.netty.util.ReferenceCountUtil;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.io.InputStream;
import java.io.RandomAccessFile;
import java.util.List;

@Slf4j
@Getter
@Setter
public class HttpHandler extends NettyHttpHandler {

    protected static final ByteBuf SSE_HEARTBEAT_BUF = Unpooled.copiedBuffer(": keepalive\n\n", CharsetUtil.US_ASCII);

    protected HttpProtocol httpProtocol;

    protected SysStore sysStore;

    protected String health;

    protected String icon;

    @Override
    public void channelRead(ChannelHandlerContext ctx, Object msg) throws Exception {
        if (FullHttpRequest.class.isAssignableFrom(msg.getClass())) {
            FullHttpRequest request = FullHttpRequest.class.cast(msg);
            if (!this.doHttpFunction(ctx, request)) {
                // 无法处理请求则提交Super
                request.headers().set("uri", request.uri());
                super.channelRead(ctx, msg);
            }
        } else {
            super.channelRead(ctx, msg);
        }
    }

    @Override
    public void userEventTriggered(ChannelHandlerContext ctx, Object evt) {
        if (evt instanceof IdleStateEvent) {
            IdleStateEvent event = IdleStateEvent.class.cast(evt);
            if (NettyWriter.isSse(ctx) && IdleState.WRITER_IDLE.equals(event.state())) {
                // SSE 注释心跳
                log.info("SSE heartbeat sent, remote={}", ctx.channel().remoteAddress());
                ctx.writeAndFlush(new DefaultHttpContent(HttpHandler.SSE_HEARTBEAT_BUF.retainedDuplicate())).addListener(NettyAlarm.INSTANCE);
            } else {
                if (log.isInfoEnabled()) {
                    log.info("Channel will be closed by idle={}", ctx.channel().remoteAddress());
                }
                ctx.close().addListener(NettyAlarm.INSTANCE);
            }
        }
        ctx.fireUserEventTriggered(evt);
    }

    protected Boolean doHttpFunction(ChannelHandlerContext ctx, FullHttpRequest request) throws Exception {
        boolean done = false;
        try {
            QueryStringDecoder query = new QueryStringDecoder(request.uri());
            if (this.isDownload(query.uri())) {
                // 文件下载
                this.doDownload(ctx, query);
                done = true;
            } else if (this.isHealth(query.uri())) {
                // 健康检查
                this.doHealth(ctx, query);
                done = true;
            } else if (this.isIcon(query.uri())) {
                // 健康检查
                this.doEmpty(ctx);
                done = true;
            }
            return done;
        } catch (Exception e) {
            if (request.refCnt() > 0) {
                // 异常释放所有引用，仅释放自己（不要多次释放source）
                ReferenceCountUtil.release(request);
            }
            throw e;
        } finally {
            // 释放原始计数
            if (done) {
                ReferenceCountUtil.release(request);
            }
        }
    }

    // 如果为单机模式则从磁盘读取
    protected void doDownload(ChannelHandlerContext ctx, QueryStringDecoder query) throws Exception {
        List<String> params = List.class.cast(MapUtils.getObject(query.parameters(), "name"));
        WorkflowException.checkCondition(CollectionUtils.isEmpty(params), "The http server download param can not be empty");
        if (this.sysStore.supportNetwork()) {
            // 优先网络
            this.doStreamCopy(ctx, params.getFirst());
        } else {
            this.doFileCopy(ctx, params.getFirst());
        }
    }

    protected void doStreamCopy(ChannelHandlerContext ctx, InputStream stream) throws Exception {
        HttpResponse response = new DefaultHttpResponse(HttpVersion.HTTP_1_1, HttpResponseStatus.OK);
        response.headers().set(HttpHeaderNames.CONTENT_TYPE, "application/octet-stream");
        HttpUtil.setTransferEncodingChunked(response, true);
        ctx.write(response);
        ctx.writeAndFlush(new HttpChunkedInput(new ChunkedStream(stream))).addListener(NettyCloser.INSTANCE);
    }

    protected void doStreamCopy(ChannelHandlerContext ctx, String name) throws Exception {
        // 由ChunkHandler负责关闭流
        InputStream stream = this.sysStore.stream(name);
        if (stream != null) {
            try {
                this.doStreamCopy(ctx, stream);
            } catch (Exception e) {
                stream.close();
                WorkflowException.dolog(e);
                throw e;
            }
        } else {
            this.doEmpty(ctx);
        }
    }

    protected void doFileCopy(ChannelHandlerContext ctx, RandomAccessFile random) throws Exception {
        HttpResponse response = new DefaultHttpResponse(HttpVersion.HTTP_1_1, HttpResponseStatus.OK);
        HttpUtil.setContentLength(response, random.length());
        response.headers().set(HttpHeaderNames.CONTENT_TYPE, "application/octet-stream");
        ctx.write(response);
        // 由Netty.DefaultFileRegion在传输结束并释放时关闭RandomAccessFile，不能同步Try关闭
        ctx.write(new DefaultFileRegion(random.getChannel(), 0, random.length()), ctx.newProgressivePromise());
        ctx.writeAndFlush(LastHttpContent.EMPTY_LAST_CONTENT).addListener(NettyCloser.INSTANCE);
    }

    protected void doFileCopy(ChannelHandlerContext ctx, String name) throws Exception {
        RandomAccessFile random = this.sysStore.access(name);
        if (random != null) {
            try {
                this.doFileCopy(ctx, random);
            } catch (Exception e) {
                WorkflowException.dolog(e);
                random.close();
                throw e;
            }
        } else {
            this.doEmpty(ctx);
        }
    }

    protected void doHealth(ChannelHandlerContext ctx, QueryStringDecoder query) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("The http server health check url={}", query.uri());
        }
        DefaultFullHttpResponse response = new DefaultFullHttpResponse(HttpVersion.HTTP_1_1, HttpResponseStatus.OK, Unpooled.copiedBuffer("OK", CharsetUtil.UTF_8));
        ctx.writeAndFlush(response).addListener(ChannelFutureListener.CLOSE);
    }

    @Override
// 从Header获取Token
    protected String token(FullHttpRequest fullHttpRequest) throws Exception {
        if (StringUtils.startsWithIgnoreCase(fullHttpRequest.uri(), HttpDistributor.MAIN)) {
            return super.token(fullHttpRequest);
        } else {
            return "";
        }
    }

    protected void doEmpty(ChannelHandlerContext ctx) throws Exception {
        FullHttpResponse response = new DefaultFullHttpResponse(HttpVersion.HTTP_1_1, HttpResponseStatus.NOT_FOUND, Unpooled.copiedBuffer(new byte[]{}));
        ctx.writeAndFlush(response).addListener(ChannelFutureListener.CLOSE);
    }

    protected Boolean isDownload(String url) throws Exception {
        return StringUtils.startsWithIgnoreCase(url, this.httpProtocol.data());
    }

    protected Boolean isHealth(String url) throws Exception {
        return StringUtils.equalsIgnoreCase(this.health, url);
    }

    protected Boolean isIcon(String url) throws Exception {
        return StringUtils.equalsIgnoreCase(this.icon, url);
    }

    @ConditionalOnProperty(name = "chat.http.enable", havingValue = "true", matchIfMissing = true)
    @Configuration
    @Setter
    @Getter
    public static class HandlerConfig extends InitConfig {

        @Autowired
        protected HttpProtocol httpProtocol;

        @Autowired
        protected SysStore sysStore;

        @Value("${chat.http.health:/health}")
        protected String health;

        @Value("${chat.http.icon:/favicon.ico}")
        protected String icon;

        @Bean
        @ConditionalOnMissingBean(value = NettyHttpHandler.class)
        public NettyHttpHandler nettyHttpHandler() throws Exception {
            HttpHandler httpService = new HttpHandler();
            BeanUtils.copyProperties(this, httpService);
            log.info("HttpHandler inited");
            return httpService;
        }
    }
}
