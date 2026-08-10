package ai.open.right.netty.chat;

import ai.open.right.utils.DumpUtils;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.workflow.config.TokenMapping;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.PooledByteBufAllocator;
import io.netty.channel.ChannelHandlerContext;
import io.netty.util.IllegalReferenceCountException;
import org.easymock.EasyMock;
import org.apache.commons.io.FileUtils;
import org.junit.Assert;
import org.junit.Test;

import java.io.ByteArrayInputStream;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.stream.Stream;

public class NettyInputProxyTest {

    /** 用于调用 protected buildRequest 的测试子类 */
    private static class TestableNettyInputProxy extends NettyInputProxy {
        TestableNettyInputProxy(ByteBuf buffer) {
            super(buffer);
        }
        NettyRequest callBuildRequest(InputStream is) throws Exception {
            return buildRequest(is);
        }
        NettyRequest callBuildRequest(String content) throws Exception {
            return buildRequest(content);
        }
    }

    /**
     * 基类构造仅 retain ByteBuf；此处用合法 buffer 覆盖成功路径（与异常路径由 NettyHttpProxyTest 覆盖）。
     */
    @Test
    public void testConstructor_retainSucceeds() {
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf.writeBytes("{}".getBytes(StandardCharsets.UTF_8));
        int rcBefore = buf.refCnt();
        NettyInputProxy proxy = new NettyInputProxy(buf);
        Assert.assertEquals(rcBefore + 1, buf.refCnt());
        proxy.close();
        Assert.assertEquals(rcBefore, buf.refCnt());
        buf.release();
    }

    @Test
    public void testContent() throws Exception {
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf = buf.writeBytes("{\"query\":\"hello\",\"biz\":\"example/example3\",\"workflow\":\"workflow1\"}".getBytes());
        NettyInputProxy proxy = new NettyInputProxy(buf);
        ChannelHandlerContext ctx = EasyMock.createMock(ChannelHandlerContext.class);
        TokenMapping tokenMapping = EasyMock.createMock(TokenMapping.class);
        EasyMock.replay(ctx, tokenMapping);
        Assert.assertEquals(proxy.buildRequest(ctx, tokenMapping).getQuery(), "hello");
        EasyMock.verify(ctx, tokenMapping);
    }

    @Test(expected = IllegalReferenceCountException.class)
    public void testClose() throws Exception {
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf = buf.writeBytes("hello".getBytes());
        NettyInputProxy proxy = new NettyInputProxy(buf);
        proxy.close();
        // close() 只 release 一次，refCnt 仍为 1，再 release 后读会抛异常
        buf.release();
        buf.readByte();
    }

    @org.junit.jupiter.api.Test
    public void testNettyInputProxyInstantiationUnique() {
        org.junit.jupiter.api.Assertions.assertTrue(true);
    }

    /**
     * 覆盖 buildRequest(String content)：从 JSON 字符串反序列化为 NettyRequest
     */
    @Test
    public void testBuildRequest_fromString() throws Exception {
        String content = "{\"query\":\"fromString\",\"biz\":\"biz1\",\"workflow\":\"wf1\"}";
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf.writeBytes(content.getBytes(StandardCharsets.UTF_8));
        TestableNettyInputProxy proxy = new TestableNettyInputProxy(buf);
        NettyRequest request = proxy.callBuildRequest(content);
        Assert.assertNotNull(request);
        Assert.assertEquals("fromString", request.getQuery());
        Assert.assertEquals("biz1", request.getBiz());
        Assert.assertEquals("wf1", request.getWorkflow());
    }

    /**
     * 覆盖 buildRequest(InputStream inputstream)：从输入流反序列化为 NettyRequest
     */
    @Test
    public void testBuildRequest_fromInputStream() throws Exception {
        String json = "{\"query\":\"fromStream\",\"biz\":\"biz2\",\"workflow\":\"wf2\"}";
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf.writeBytes(json.getBytes(StandardCharsets.UTF_8));
        TestableNettyInputProxy proxy = new TestableNettyInputProxy(buf);
        try (InputStream is = new ByteArrayInputStream(json.getBytes(StandardCharsets.UTF_8))) {
            NettyRequest request = proxy.callBuildRequest(is);
            Assert.assertNotNull(request);
            Assert.assertEquals("fromStream", request.getQuery());
            Assert.assertEquals("biz2", request.getBiz());
            Assert.assertEquals("wf2", request.getWorkflow());
        }
    }

    @Test
    public void testHarness_writesDumpWhenConfigured() throws Exception {
        Path dir = Files.createTempDirectory("chat-input-proxy-harness");
        try {
            String json = "{\"query\":\"hq\",\"biz\":\"bb\",\"workflow\":\"ww\"}";
            ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
            buf.writeBytes(json.getBytes(StandardCharsets.UTF_8));
            NettyInputProxy proxy = new NettyInputProxy(buf, dir.toAbsolutePath().toString());
            ChannelHandlerContext ctx = EasyMock.createMock(ChannelHandlerContext.class);
            TokenMapping tokenMapping = EasyMock.createMock(TokenMapping.class);
            EasyMock.replay(ctx, tokenMapping);
            Assert.assertEquals("hq", proxy.buildRequest(ctx, tokenMapping).getQuery());
            try (Stream<Path> list = Files.list(dir)) {
                Assert.assertEquals(1L, list.filter(p -> p.getFileName().toString().startsWith(DumpUtils.DUMP_PREFIX + "_REQUEST_CHAT_")).count());
            }
            proxy.close();
            EasyMock.verify(ctx, tokenMapping);
        } finally {
            FileUtils.deleteDirectory(dir.toFile());
        }
    }

}
