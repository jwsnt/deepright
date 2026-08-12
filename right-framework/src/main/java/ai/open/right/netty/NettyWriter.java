package ai.open.right.netty;

import ai.open.right.WorkflowException;
import ai.open.right.netty.chat.NettySegment;
import ai.open.right.netty.chat.server.NettyAttributes;
import ai.open.right.netty.chat.server.http.NettyHttpResponse;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.JsonUtils;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.CompositeByteBuf;
import io.netty.buffer.Unpooled;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.*;
import io.netty.handler.codec.http.websocketx.TextWebSocketFrame;
import io.netty.util.ReferenceCountUtil;
import lombok.extern.slf4j.Slf4j;
import org.springframework.util.Assert;

import java.io.OutputStream;
import java.nio.charset.StandardCharsets;

@Slf4j
public class NettyWriter {

    private static final ByteBuf STREAM_DONE = Unpooled.unreleasableBuffer(Unpooled.copiedBuffer("[DONE]\n\n", StandardCharsets.UTF_8));

    private static final ByteBuf STREAM_START = Unpooled.unreleasableBuffer(Unpooled.copiedBuffer("data: ", StandardCharsets.UTF_8));

    private static final ByteBuf STREAM_CLOSE = Unpooled.unreleasableBuffer(Unpooled.copiedBuffer("\n\n", StandardCharsets.UTF_8));

    // 写入报文，并由调用的内部函数控制关闭通道
    public static void write(ChannelHandlerContext ctx, NettySegment segment) throws Exception {
        Assert.notNull(ctx, "Channel can not be empty");
        ByteBuf buffer = null;
        try {
            // 无法获取Code获取Code为负数的立即关闭
            if (segment.getCode() != null && segment.getCode() <= 0) {
                if (log.isDebugEnabled()) {
                    log.debug("Channel was forcefully closed={}", segment.getCode());
                }
                // 如果为Http Stream类型先写入[Done] \n\n
                if (NettyWriter.isStream(ctx)) {
                    if (log.isDebugEnabled()) {
                        log.debug("Channel write [DONE]={}", ctx.channel().remoteAddress());
                    }
                    buffer = ctx.alloc().directBuffer();
                    // ""发送，Null不发送
                    if (segment.getContent() != null) {
                        buffer.writeBytes(NettyWriter.STREAM_START.retainedDuplicate());
                        NettyWriter.writeHttpBuffer(ctx, segment, buffer);
                        buffer.writeBytes(NettyWriter.STREAM_CLOSE.retainedDuplicate());
                    }
                    buffer.writeBytes(NettyWriter.STREAM_START.retainedDuplicate());
                    buffer.writeBytes(NettyWriter.STREAM_DONE.retainedDuplicate());
                    // 写入后关闭
                    ctx.writeAndFlush(new DefaultHttpContent(buffer)).addListener(new NettyCloser(ctx));
                } else {
                    // WS/Http Once直接关闭
                    ctx.close().addListener(NettyAlarm.INSTANCE);
                }
                return;
            }
            if (NettyWriter.isHttpService(ctx)) {
                // 处理Http服务(Once/Stream)
                if (log.isDebugEnabled()) {
                    log.debug("Channel write http response :{}", ctx.channel().remoteAddress());
                }
                // 非Stream（At Once）时Content不能为Null
                WorkflowException.checkCondition(!NettyWriter.isStream(ctx) && segment.getContent() == null, "Channel write http response is null");
                // ""发送，Null不发送
                if (segment.getContent() != null) {
                    buffer = ctx.alloc().directBuffer();
                    NettyWriter.writeHttpBuffer(ctx, segment, buffer);
                    NettyWriter.writeHttp(ctx, segment, buffer);
                    // writeHttp 已将 buffer 所有权转移给 Netty，标记为 null 防止 catch 块误释放
                    buffer = null;
                }
            } else if (NettyWriter.isWsService(ctx)) {
                // 处理Ws服务
                if (log.isDebugEnabled()) {
                    log.debug("Channel write ws response :{}", ctx.channel().remoteAddress());
                }
                // ""发送，Null不发送
                if (segment.getContent() != null) {
                    buffer = ctx.alloc().directBuffer();
                    NettyWriter.writeWsBuffer(ctx, segment, buffer);
                    NettyWriter.writeWs(ctx, buffer);
                    // writeWs 已将 buffer 所有权转移给 Netty，标记为 null 防止 catch 块误释放
                    buffer = null;
                }
            } else {
                throw new WorkflowException("Can not support this network protocol", ProtocolCode.C400);
            }
        } catch (Exception e) {
            if (buffer != null && buffer.refCnt() > 0) {
                // 强制完全释放
                ReferenceCountUtil.release(buffer, buffer.refCnt());
            }
            throw e;
        } finally {
            // Stream响应，标记索引
            segment.mark();
        }
    }

    public static void writeHttpBuffer(ChannelHandlerContext ctx, NettySegment segment, ByteBuf byteBuf) throws Exception {
        try (OutputStream output = new NettyOutputBuffer(byteBuf)) {
            // 转为Open AI格式的Http Response
            JsonUtils.write(output, new NettyHttpResponse(segment, NettyWriter.isStream(ctx), NettyWriter.isSse(ctx)));
        }
    }

    public static void writeWsBuffer(ChannelHandlerContext ctx, NettySegment segment, ByteBuf byteBuf) throws Exception {
        try (OutputStream output = new NettyOutputBuffer(byteBuf)) {
            // 自定义格式的Segment
            JsonUtils.write(output, segment);
        }
    }

    // 写入HTTP报文（Once/Stream），并控制关闭通道
    // 注意：此方法会将byteBuf所有权转移给Netty，调用成功后，上层不应再释放byteBuf
    public static void writeHttp(ChannelHandlerContext ctx, NettySegment segment, ByteBuf byteBuf) throws Exception {
        // writeStream/writeOnce 会将 byteBuf 传递给 Netty，所有权即转移
        // 即使后续抛出异常，也不应该由调用者释放，而是由 Netty 负责
        if (NettyWriter.isStream(ctx)) {
            // Stream模式下，如果空报文不写入通道
            NettyWriter.writeStream(ctx, segment, byteBuf);
        } else {
            // Http Once 发送完成后关闭连接
            NettyWriter.writeOnce(ctx, segment, byteBuf);
        }
    }

    // 写入HTTP报文（Stream），并不关闭通道（由调用函数关闭）
    public static void writeStream(ChannelHandlerContext ctx, NettyStream stream, ByteBuf buffer) throws Exception {
        // 扩容，Stream格式、平移内容、写入头尾
        CompositeByteBuf composite = ctx.alloc().compositeBuffer();
        try {
            composite.addComponent(true, NettyWriter.STREAM_START.retainedDuplicate());
            composite.addComponent(true, buffer);
            composite.addComponent(true, NettyWriter.STREAM_CLOSE.retainedDuplicate());
            // 1，消息标记完成 且 非SSE通道（Stream正常关闭）
            // 2，非200 Code 且 为SEE通道（SSE异常关闭）
            if (NettyWriter.isStreamClose(ctx, stream)) {
                // 关闭条件：
                // 1，消息标记完成 且 非SSE通道（）
                // 2，非200 Code 且 为SEE通道（终止通道）
                composite.addComponent(true, NettyWriter.STREAM_START.retainedDuplicate());
                composite.addComponent(true, STREAM_DONE.retainedDuplicate());
            }
            // 一旦传递给DefaultHttpContent，buffer的所有权转移给Netty，Netty负责释放
            // 注意：此行之后buffer的所有权已转移，调用者catch块不应再释放

            if (NettyWriter.isStreamClose(ctx, stream)) {
                ctx.writeAndFlush(new DefaultHttpContent(composite)).addListener(NettyCloser.INSTANCE);
            } else {
                ctx.writeAndFlush(new DefaultHttpContent(composite)).addListener(NettyAlarm.INSTANCE);
            }
        } catch (Exception e) {
            // 异常时释放：若 buffer 尚未加入 composite（numComponents < 2）则单独释放，再释放 composite
            if (buffer != null && buffer.refCnt() > 0 && composite.numComponents() < 2) {
                ReferenceCountUtil.release(buffer);
            }
            composite.release();
            throw e;
        }
    }

    // 写入HTTP报文（Stream），并当NettyStream.isFinished=true时负责关闭通道
    public static void writeStream(ChannelHandlerContext ctx, NettyStream stream) throws Exception {
        ByteBuf buffer = null;
        try {
            // Stream格式
            buffer = ctx.alloc().directBuffer();
            buffer.writeBytes((NettyWriter.STREAM_START).retainedDuplicate());
            try (OutputStream output = new NettyOutputBuffer(buffer)) {
                JsonUtils.write(output, stream);
            }
            buffer.writeBytes(NettyWriter.STREAM_CLOSE.retainedDuplicate());
            // 消息标记完成
            if (stream.isFinished()) {
                buffer.writeBytes((NettyWriter.STREAM_START).retainedDuplicate());
                buffer.writeBytes(NettyWriter.STREAM_DONE.retainedDuplicate());
            }
            ctx.writeAndFlush(new DefaultHttpContent(buffer)).addListener(stream.isFinished() ? new NettyCloser(ctx) : NettyAlarm.INSTANCE);
        } catch (Exception e) {
            if (buffer != null && buffer.refCnt() > 0) {
                // 强制完全释放
                ReferenceCountUtil.release(buffer, buffer.refCnt());
            }
            throw e;
        }
    }

    // 写入HTTP报文（Once），并不关闭通道（由调用函数关闭）
    public static void writeOnce(ChannelHandlerContext ctx, NettyStream object, ByteBuf byteBuf) throws Exception {
        DefaultFullHttpResponse defaultFullHttpResponse = new DefaultFullHttpResponse(HttpVersion.HTTP_1_1, HttpResponseStatus.valueOf(object.getCode()), byteBuf);
        defaultFullHttpResponse.headers().set(HttpHeaderNames.CONTENT_TYPE, "application/json;charset=utf-8");
        defaultFullHttpResponse.headers().set(HttpHeaderNames.CONNECTION, HttpHeaderValues.KEEP_ALIVE);
        defaultFullHttpResponse.headers().set(HttpHeaderNames.CONTENT_LENGTH, byteBuf.readableBytes());
        defaultFullHttpResponse.headers().set(HttpHeaderNames.CACHE_CONTROL, "no-cache");
        // 写入跨域Header
        NettyWriter.addCorsHeaders(ctx, defaultFullHttpResponse.headers());
        ctx.writeAndFlush(defaultFullHttpResponse).addListener(NettyCloser.INSTANCE);
    }

    // 写入HTTP报文（MCP/A2A Once），并关闭通道
    public static void writeOnce(ChannelHandlerContext ctx, Object object) throws Exception {
        ByteBuf buffer = null;
        try {
            buffer = ctx.alloc().directBuffer();
            try (OutputStream output = new NettyOutputBuffer(buffer)) {
                JsonUtils.write(output, object);
            }
            NettyWriter.writeOnce(ctx, NettyStream.SUCCESS, buffer);
        } catch (Exception e) {
            if (buffer != null && buffer.refCnt() > 0) {
                // 强制完全释放
                ReferenceCountUtil.release(buffer, buffer.refCnt());
            }
            throw e;
        }
    }

    // 写入WS报文，并不关闭通道（由调用函数关闭）
    public static void writeWs(ChannelHandlerContext ctx, ByteBuf byteBuf) throws Exception {
        ctx.writeAndFlush(new TextWebSocketFrame(byteBuf)).addListener(NettyAlarm.INSTANCE);
    }

    // 强制关闭通道
    public static void close(ChannelHandlerContext ctx) throws Exception {
        ctx.close().addListener(NettyAlarm.INSTANCE);
    }

    // 写入跨域Headers
    public static void addCorsHeaders(ChannelHandlerContext ctx, HttpHeaders headers) throws Exception {
        if (NettyWriter.isCors(ctx)) {
            if (log.isDebugEnabled()) {
                log.debug("Write cors header={}", ctx.channel().remoteAddress());
            }
            headers.set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Cache-Control, Authorization");
            headers.set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS");
            headers.set("Access-Control-Allow-Origin", "*");
            headers.set("Access-Control-Max-Age", "3600");
        }
    }

    // 建立Stream连接
    public static void connectStream(ChannelHandlerContext ctx) throws Exception {
        DefaultHttpResponse defaultHttpResponse = new DefaultHttpResponse(HttpVersion.HTTP_1_1, HttpResponseStatus.OK);
        defaultHttpResponse.headers().set(HttpHeaderNames.CONTENT_TYPE, "text/event-stream;charset=utf-8");
        defaultHttpResponse.headers().set(HttpHeaderNames.CONNECTION, HttpHeaderValues.KEEP_ALIVE);
        defaultHttpResponse.headers().set(HttpHeaderNames.CACHE_CONTROL, "no-cache");
        // 写入跨域Header
        NettyWriter.addCorsHeaders(ctx, defaultHttpResponse.headers());
        ctx.writeAndFlush(defaultHttpResponse).addListener(NettyAlarm.INSTANCE);
    }

    // 在通道绑定通道类型的标记(Once/Stream)
    public static void flagConnection(ChannelHandlerContext ctx, Byte type) throws Exception {
        ctx.attr(NettyAttributes.CONNECTION_TYPE).set(type);
        if (NettyAttributes.CONNECTION_STREAM.equals(type)) {
            // 如果是Stream服务，建立Stream连接
            NettyWriter.connectStream(ctx);
        }
    }

    // 在通道绑定服务类型的标记(WS/HTTP)
    public static void flagServer(ChannelHandlerContext ctx, Byte type) throws Exception {
        ctx.attr(NettyAttributes.SERVER_TYPE).set(type);
    }

    // 在通道绑定Cors类型的标记
    public static void flagCors(ChannelHandlerContext ctx) throws Exception {
        ctx.attr(NettyAttributes.CORS_TYPE).set(NettyAttributes.HTTP_CORS);
    }

    // 在通道绑定SSE类型的标记
    public static void flagSse(ChannelHandlerContext ctx) throws Exception {
        ctx.attr(NettyAttributes.SSE_TYPE).set(NettyAttributes.HTTP_SSE);
    }

    // 1，消息标记完成 且 非SSE通道（Stream正常关闭）
    // 2，非200 Code 且 为SEE通道（SSE异常关闭）
    public static Boolean isStreamClose(ChannelHandlerContext ctx, NettyStream object) {
        return (object.isFinished() && !NettyWriter.isSse(ctx)) || (!ProtocolCode.range2xx(object.getCode()) && NettyWriter.isSse(ctx));
    }

    public static Boolean isHttpService(ChannelHandlerContext ctx) {
        return NettyAttributes.SERVER_HTTP.equals(ctx.attr(NettyAttributes.SERVER_TYPE).get());
    }

    public static Boolean isWsService(ChannelHandlerContext ctx) {
        return NettyAttributes.SERVER_WS.equals(ctx.attr(NettyAttributes.SERVER_TYPE).get());
    }

    public static Boolean isStream(ChannelHandlerContext ctx) {
        return NettyAttributes.CONNECTION_STREAM.equals(ctx.attr(NettyAttributes.CONNECTION_TYPE).get());
    }

    public static Boolean isCors(ChannelHandlerContext ctx) {
        return NettyAttributes.HTTP_CORS.equals(ctx.attr(NettyAttributes.CORS_TYPE).get());
    }

    public static Boolean isOnce(ChannelHandlerContext ctx) {
        return NettyAttributes.CONNECTION_ONCE.equals(ctx.attr(NettyAttributes.CONNECTION_TYPE).get());
    }

    public static Boolean isSse(ChannelHandlerContext ctx) {
        return NettyAttributes.HTTP_SSE.equals(ctx.attr(NettyAttributes.SSE_TYPE).get());
    }
}
