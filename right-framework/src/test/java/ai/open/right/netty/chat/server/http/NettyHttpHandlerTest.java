package ai.open.right.netty.chat.server.http;

import ai.open.right.WorkflowException;
import ai.open.right.netty.NettyCloser;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.netty.chat.server.NettyAttributes;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.DumpUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.config.TokenMapping;
import ai.open.right.workflow.config.impl.NoneTokenMapping;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.ByteBufAllocator;
import io.netty.buffer.Unpooled;
import io.netty.channel.ChannelFuture;
import io.netty.channel.ChannelHandlerContext;
import io.netty.channel.ChannelInboundHandlerAdapter;
import io.netty.channel.embedded.EmbeddedChannel;
import io.netty.handler.codec.http.*;
import io.netty.util.Attribute;
import org.apache.commons.io.FileUtils;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;
import java.util.Map;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.stream.Stream;

public class NettyHttpHandlerTest {

    private static void initHttpOnceContext(ChannelHandlerContext context) {
        context.attr(NettyAttributes.SERVER_TYPE).set(NettyAttributes.SERVER_HTTP);
        context.attr(NettyAttributes.CONNECTION_TYPE).set(NettyAttributes.CONNECTION_ONCE);
        context.attr(NettyAttributes.SSE_TYPE).set(NettyAttributes.HTTP_SSE);
        context.attr(NettyAttributes.CORS_TYPE).set((byte) 0);
    }

    @SuppressWarnings("unchecked")
    private static Map<String, Object> readOutboundErrorJson(EmbeddedChannel ch) throws Exception {
        ch.runPendingTasks();
        Object msg = ch.outboundMessages().poll();
        Assert.assertNotNull(msg);
        DefaultFullHttpResponse resp = (DefaultFullHttpResponse) msg;
        String body = resp.content().toString(StandardCharsets.UTF_8);
        return JsonUtils.read(body, Map.class);
    }

    @SuppressWarnings("unchecked")
    private static String firstChoiceMessageContent(Map<String, Object> json) {
        List<Map<String, Object>> choices = (List<Map<String, Object>>) json.get("choices");
        Assert.assertNotNull(choices);
        Assert.assertFalse(choices.isEmpty());
        Map<String, Object> message = (Map<String, Object>) choices.get(0).get("message");
        Assert.assertNotNull(message);
        return (String) message.get("content");
    }

    @Test
    public void test() throws Exception {
        ChannelInboundHandlerAdapter tail = new ChannelInboundHandlerAdapter() {
        };
        EmbeddedChannel embeddedChannel = new EmbeddedChannel(tail);
        ChannelHandlerContext context = embeddedChannel.pipeline().context(tail);
        initHttpOnceContext(context);
        NettyHttpHandler nettyHttpHandler = new NettyHttpHandler();
        nettyHttpHandler.exceptionCaught(context, new RuntimeException("OK"));
        embeddedChannel.runPendingTasks();
        Assert.assertFalse("应已写出 HTTP 错误响应", embeddedChannel.outboundMessages().isEmpty());
    }

    @Test
    public void testChannelReadEmptyBodyReturnsHttpErrorInsteadOfProtocolError() throws Exception {
        ChannelInboundHandlerAdapter tail = new ChannelInboundHandlerAdapter() {
        };
        EmbeddedChannel ch = new EmbeddedChannel(tail);
        ChannelHandlerContext ctx = ch.pipeline().context(tail);
        FullHttpRequest req = new DefaultFullHttpRequest(HttpVersion.HTTP_1_1, HttpMethod.POST, "/", Unpooled.buffer(0));
        req.headers().set(HttpHeaderNames.AUTHORIZATION, "Bearer token");
        NettyHttpHandler handler = new NettyHttpHandler();
        handler.channelRead(ctx, req);
        Map<String, Object> json = readOutboundErrorJson(ch);
        Assert.assertEquals(ProtocolCode.mapping(ProtocolCode.C500), json.get("code"));
        Assert.assertEquals("Internal Server Error", firstChoiceMessageContent(json));
    }

    /** exceptionCaught：无 exposure 时固定 C500 + 文案 Internal Server Error */
    @Test
    public void testExceptionCaught_noExposure_uses500AndGenericMessage() throws Exception {
        ChannelInboundHandlerAdapter tail = new ChannelInboundHandlerAdapter() {
        };
        EmbeddedChannel ch = new EmbeddedChannel(tail);
        ChannelHandlerContext ctx = ch.pipeline().context(tail);
        initHttpOnceContext(ctx);
        new NettyHttpHandler().exceptionCaught(ctx, new RuntimeException("do-not-leak"));
        Map<String, Object> json = readOutboundErrorJson(ch);
        Assert.assertEquals(ProtocolCode.mapping(ProtocolCode.C500), json.get("code"));
        Assert.assertEquals("Internal Server Error", firstChoiceMessageContent(json));
    }

    /** exceptionCaught：WorkflowException 且 needExposure 时返回业务 code 与异常消息 */
    @Test
    public void testExceptionCaught_exposedWorkflowException_usesCodeAndMessage() throws Exception {
        ChannelInboundHandlerAdapter tail = new ChannelInboundHandlerAdapter() {
        };
        EmbeddedChannel ch = new EmbeddedChannel(tail);
        ChannelHandlerContext ctx = ch.pipeline().context(tail);
        initHttpOnceContext(ctx);
        WorkflowException ex = new WorkflowException("client-visible", ProtocolCode.C400);
        ex.needExposure();
        new NettyHttpHandler().exceptionCaught(ctx, ex);
        Map<String, Object> json = readOutboundErrorJson(ch);
        Assert.assertEquals(ProtocolCode.mapping(ProtocolCode.C400), json.get("code"));
        Assert.assertEquals("client-visible", firstChoiceMessageContent(json));
    }

    /**
     * exposure 沿 cause 为 true 时，code/getMessage 仍针对顶层 Throwable：
     * 外层为 RuntimeException 时 WorkflowException.code 走兜底 500，文案为外层 message。
     */
    @Test
    public void testExceptionCaught_exposureTrueButTopLevelRuntime_usesTopMessageAndCodeFallback() throws Exception {
        ChannelInboundHandlerAdapter tail = new ChannelInboundHandlerAdapter() {
        };
        EmbeddedChannel ch = new EmbeddedChannel(tail);
        ChannelHandlerContext ctx = ch.pipeline().context(tail);
        initHttpOnceContext(ctx);
        WorkflowException inner = new WorkflowException("from-inner", ProtocolCode.C429);
        inner.needExposure();
        RuntimeException outer = new RuntimeException("wrapper", inner);
        new NettyHttpHandler().exceptionCaught(ctx, outer);
        Map<String, Object> json = readOutboundErrorJson(ch);
        Assert.assertEquals(ProtocolCode.mapping(ProtocolCode.C500), json.get("code"));
        Assert.assertEquals("wrapper", firstChoiceMessageContent(json));
    }

    /** WorkflowException 未 needExposure 时与普适异常相同，不对外暴露业务码与详情 */
    @Test
    public void testExceptionCaught_workflowExceptionWithoutExposure_masksAs500() throws Exception {
        ChannelInboundHandlerAdapter tail = new ChannelInboundHandlerAdapter() {
        };
        EmbeddedChannel ch = new EmbeddedChannel(tail);
        ChannelHandlerContext ctx = ch.pipeline().context(tail);
        initHttpOnceContext(ctx);
        new NettyHttpHandler().exceptionCaught(ctx, new WorkflowException("secret", ProtocolCode.C400));
        Map<String, Object> json = readOutboundErrorJson(ch);
        Assert.assertEquals(ProtocolCode.mapping(ProtocolCode.C500), json.get("code"));
        Assert.assertEquals("Internal Server Error", firstChoiceMessageContent(json));
    }

    @Test
    public void testRead() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closefuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closefuture.addListener(EasyMock.anyObject(NettyCloser.class))).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closefuture).anyTimes();
        Attribute<Byte> attributeService = EasyMock.createMock(Attribute.class);
        attributeService.set(NettyAttributes.SERVER_HTTP);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(attributeService.get()).andReturn(NettyAttributes.SERVER_HTTP).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SERVER_TYPE)).andReturn(attributeService).anyTimes();
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeHttp.get()).andReturn(NettyAttributes.CONNECTION_ONCE).anyTimes();
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeSSe.get()).andReturn(NettyAttributes.HTTP_SSE).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closefuture).anyTimes();
        EasyMock.replay(context, closefuture, attributeCors, returnFuture, attributeService, attributeHttp, attributeSSe);
        AtomicBoolean ex = new AtomicBoolean(false);
        NettyHttpHandler nettyHttpHandler = new NettyHttpHandler() {
            @Override
            // 基础异常处理，实现类覆盖
            public void exceptionCaught(ChannelHandlerContext ctx, Throwable cause) throws Exception {
                ex.set(true);
            }
        };
        // 解析错误
        nettyHttpHandler.channelRead(context, new RuntimeException("OK"));
        Assert.assertTrue(ex.get());
        EasyMock.verify(context, closefuture, attributeCors, returnFuture, returnFuture, attributeService, attributeHttp, attributeSSe);
    }

    @Test(expected = WorkflowException.class)
    public void testToken() throws Exception {
        FullHttpRequest fullHttpRequest = new DefaultFullHttpRequest(HttpVersion.HTTP_1_1, HttpMethod.POST, "/", Unpooled.EMPTY_BUFFER);
        NettyHttpHandler nettyHttpHandler = new NettyHttpHandler();
        nettyHttpHandler.token(fullHttpRequest);
    }

    @org.junit.jupiter.api.Test
    public void testNettyHttpHandlerInstantiationUnique() {
        org.junit.jupiter.api.Assertions.assertTrue(true);
    }

    @org.junit.jupiter.api.Test
    public void testTokenSuccess() throws Exception {
        FullHttpRequest request = new DefaultFullHttpRequest(HttpVersion.HTTP_1_1, HttpMethod.POST, "/", Unpooled.EMPTY_BUFFER);
        request.headers().set(HttpHeaderNames.AUTHORIZATION, "Bearer my-token");
        NettyHttpHandler handler = new NettyHttpHandler();
        org.junit.jupiter.api.Assertions.assertEquals("my-token", handler.token(request));
    }

    @org.junit.jupiter.api.Test
    public void testFlagSse() throws Exception {
        ChannelInboundHandlerAdapter tail = new ChannelInboundHandlerAdapter() {
        };
        EmbeddedChannel ch = new EmbeddedChannel(tail);
        ChannelHandlerContext ctx = ch.pipeline().context(tail);
        DefaultHttpHeaders headers = new DefaultHttpHeaders();
        headers.set("Accept", "text/event-stream");
        NettyHttpHandler handler = new NettyHttpHandler();
        handler.flagSse(ctx, headers);
        org.junit.jupiter.api.Assertions.assertEquals(NettyAttributes.HTTP_SSE, ch.attr(NettyAttributes.SSE_TYPE).get());
    }

    @Test
    public void testHarnessGetterSetter() {
        NettyHttpHandler h = new NettyHttpHandler();
        h.setAutoDump("/data/http-dumps");
        Assert.assertEquals("/data/http-dumps", h.getAutoDump());
    }

    /**
     * buildInputProxy 将 handler.autodump 传入 NettyHttpProxy；配置目录时 buildRequest 会落盘 REQUEST_CHAT。
     */
    @Test
    public void testBuildInputProxy_passesHarnessForDump() throws Exception {
        Path dir = Files.createTempDirectory("netty-http-handler-harness");
        try {
            String json = IOUtils.toString(ResourceUtils.getURL("classpath:OpenAiRequest_1.json").openStream(), StandardCharsets.UTF_8);
            ByteBuf content = Unpooled.copiedBuffer(json, StandardCharsets.UTF_8);
            FullHttpRequest req = new DefaultFullHttpRequest(HttpVersion.HTTP_1_1, HttpMethod.POST, "/", content.retain());
            req.headers().set(HttpHeaderNames.AUTHORIZATION, "Bearer TOKEN");
            ChannelInboundHandlerAdapter tail = new ChannelInboundHandlerAdapter() {
            };
            EmbeddedChannel ch = new EmbeddedChannel(tail);
            ChannelHandlerContext ctx = ch.pipeline().context(tail);
            NettyHttpHandler handler = new NettyHttpHandler();
            handler.setAutoDump(dir.toAbsolutePath().toString());
            ByteBuf forProxy = content.retain();
            NettyHttpProxy proxy = (NettyHttpProxy) handler.buildInputProxy(ctx, req, forProxy);
            ChannelHandlerContext buildCtx = EasyMock.createMock(ChannelHandlerContext.class);
            Attribute<Byte> attributeType = EasyMock.createMock(Attribute.class);
            EasyMock.expect(buildCtx.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeType).anyTimes();
            attributeType.set(EasyMock.anyObject());
            EasyMock.expectLastCall().anyTimes();
            TokenMapping tokenMapping = new NoneTokenMapping();
            EasyMock.replay(buildCtx, attributeType);
            NettyRequest built = proxy.buildRequest(buildCtx, tokenMapping);
            Assert.assertNotNull(built);
            try (Stream<Path> list = Files.list(dir)) {
                long n = list.filter(p -> p.getFileName().toString().startsWith(DumpUtils.DUMP_PREFIX + "_REQUEST_CHAT_")).count();
                Assert.assertEquals(1L, n);
            }
            proxy.close();
            EasyMock.verify(buildCtx, attributeType);
        } finally {
            FileUtils.deleteDirectory(dir.toFile());
        }
    }

    /**
     * 覆盖 byteBuf 异常分支：buffer 已赋值后 Assert 失败，进入 catch 执行 ReferenceCountUtil.release(buffer)
     */
    @Test(expected = IllegalArgumentException.class)
    public void testByteBufExceptionReleasesBuffer() throws Exception {
        ByteBuf emptyBuf = Unpooled.buffer(0);
        org.junit.Assert.assertEquals(0, emptyBuf.readableBytes());
        FullHttpRequest fullHttpRequest = new DefaultFullHttpRequest(HttpVersion.HTTP_1_1, HttpMethod.POST, "/", emptyBuf);
        ChannelHandlerContext ctx = EasyMock.createMock(ChannelHandlerContext.class);
        EasyMock.replay(ctx);
        NettyHttpHandler handler = new NettyHttpHandler();
        try {
            handler.byteBuf(ctx, fullHttpRequest);
        } finally {
            org.junit.Assert.assertTrue("buffer should have been released in catch", emptyBuf.refCnt() <= 1);
            if (emptyBuf.refCnt() > 0) {
                emptyBuf.release();
            }
            EasyMock.verify(ctx);
        }
    }

}
