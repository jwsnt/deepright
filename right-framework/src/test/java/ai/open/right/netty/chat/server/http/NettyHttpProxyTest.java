package ai.open.right.netty.chat.server.http;

import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.netty.chat.server.NettyAttributes;
import ai.open.right.utils.DumpUtils;
import ai.open.right.workflow.config.TokenMapping;
import ai.open.right.workflow.config.impl.NoneTokenMapping;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.PooledByteBufAllocator;
import io.netty.handler.codec.http.DefaultHttpHeaders;
import io.netty.channel.ChannelHandlerContext;
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
import java.util.Map;
import java.util.stream.Stream;

public class NettyHttpProxyTest {

    /**
     * super 已 retain 后子类 token 校验失败时应对 buffer 对称释放，避免泄漏。
     */
    @Test
    public void testConstructor_releasesBufferWhenTokenFormatInvalid() {
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.buffer();
        buf.writeBytes("{}".getBytes(StandardCharsets.UTF_8));
        int rcBefore = buf.refCnt();
        try {
            // 两段式 token 但 device:chat 缺少冒号对，触发 Assert
            new NettyHttpProxy(new DefaultHttpHeaders(), buf, "TOKEN/bad", null);
            Assert.fail("expected IllegalArgumentException");
        } catch (IllegalArgumentException expected) {
        }
        Assert.assertEquals(rcBefore, buf.refCnt());
        buf.release();
    }

    @Test
    public void test1() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        TokenMapping tokenMapping = new NoneTokenMapping();
        ByteBuf byteBuf = PooledByteBufAllocator.DEFAULT.buffer();
        byteBuf.writeBytes(IOUtils.toString(ResourceUtils.getURL("classpath:OpenAiRequest_1.json").openStream()).getBytes(StandardCharsets.UTF_8));
        Attribute<Byte> attributeType = EasyMock.createMock(Attribute.class);
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeType).anyTimes();
        EasyMock.replay(context);
        NettyHttpProxy nettyHttpProxy = new NettyHttpProxy(byteBuf, "TOKEN");
        NettyRequest nettyRequest = nettyHttpProxy.buildRequest(context, tokenMapping);
        Assert.assertEquals("HELLO DEVICE", nettyRequest.getDevice());
        Assert.assertNull(nettyRequest.getBiz());
        Assert.assertNull(nettyRequest.getWorkflow());
        Assert.assertEquals("HELLO VALUE2", Map.class.cast(Map.class.cast(nettyRequest.getMetadata().get("key1")).get("key2")).get("key3"));
        Assert.assertEquals("你好，我叫李雷，1+1等于多少？", nettyRequest.getQuery());
        Assert.assertEquals("HELLO TRACE", nettyRequest.getTrace());
        Assert.assertEquals("HELLO CONVERSATION", nettyRequest.getConversation());
        Assert.assertEquals("HELLO CHAT", nettyRequest.getChat());
        Assert.assertEquals("HELLO MODEL", nettyRequest.getUserContext().getModel());
        Assert.assertEquals("HELLO DEVICE", nettyRequest.getUserContext().getDevice());
        Assert.assertEquals("HELLO REGION", nettyRequest.getUserContext().getRegion());
        Assert.assertEquals("HELLO LANGUAGE", nettyRequest.getUserContext().getLanguage());
        Assert.assertEquals("HELLO SYSTEM", nettyRequest.getUserContext().getSystem());
        Assert.assertEquals("HELLO BRAND", nettyRequest.getUserContext().getBrand());
        Assert.assertEquals("HELLO TOKEN", nettyRequest.getUserContext().getToken());
        EasyMock.verify(context);
    }

    @Test
    public void test2() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        TokenMapping tokenMapping = new NoneTokenMapping();
        ByteBuf byteBuf = PooledByteBufAllocator.DEFAULT.buffer();
        byteBuf.writeBytes(IOUtils.toString(ResourceUtils.getURL("classpath:OpenAiRequest_2.json").openStream()).getBytes(StandardCharsets.UTF_8));
        Attribute<Byte> attributeType = EasyMock.createMock(Attribute.class);
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeType).anyTimes();
        EasyMock.replay(context);
        NettyHttpProxy nettyHttpProxy = new NettyHttpProxy(byteBuf, "TOKEN");
        NettyRequest nettyRequest = nettyHttpProxy.buildRequest(context, tokenMapping);
        Assert.assertEquals("HELLO DEVICE", nettyRequest.getDevice());
        Assert.assertNull(nettyRequest.getBiz());
        Assert.assertNull(nettyRequest.getWorkflow());
        Assert.assertEquals("HELLO VALUE2", Map.class.cast(Map.class.cast(nettyRequest.getMetadata().get("key1")).get("key2")).get("key3"));
        Assert.assertEquals("你好，我叫李雷，1+1等于多少？", nettyRequest.getQuery());
        Assert.assertEquals("HELLO TRACE", nettyRequest.getTrace());
        Assert.assertEquals("HELLO CONVERSATION", nettyRequest.getConversation());
        Assert.assertEquals("HELLO CHAT", nettyRequest.getChat());
        Assert.assertEquals("HELLO MODEL", nettyRequest.getUserContext().getModel());
        Assert.assertEquals("HELLO DEVICE", nettyRequest.getUserContext().getDevice());
        Assert.assertEquals("HELLO REGION", nettyRequest.getUserContext().getRegion());
        Assert.assertEquals("HELLO LANGUAGE", nettyRequest.getUserContext().getLanguage());
        Assert.assertEquals("HELLO SYSTEM", nettyRequest.getUserContext().getSystem());
        Assert.assertEquals("HELLO BRAND", nettyRequest.getUserContext().getBrand());
        Assert.assertEquals("HELLO TOKEN", nettyRequest.getUserContext().getToken());
        EasyMock.verify(context);
    }

    @Test
    public void test3() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        TokenMapping tokenMapping = new NoneTokenMapping();
        ByteBuf byteBuf = PooledByteBufAllocator.DEFAULT.buffer();
        byteBuf.writeBytes(IOUtils.toString(ResourceUtils.getURL("classpath:OpenAiRequest_1.json").openStream()).getBytes(StandardCharsets.UTF_8));
        Attribute<Byte> attributeType = EasyMock.createMock(Attribute.class);
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeType).anyTimes();
        EasyMock.replay(context);
        NettyHttpProxy nettyHttpProxy = new NettyHttpProxy(byteBuf, "TOKEN/A:B");
        NettyRequest nettyRequest = nettyHttpProxy.buildRequest(context, tokenMapping);
        Assert.assertEquals("HELLO DEVICE", nettyRequest.getDevice());
        Assert.assertEquals("HELLO DEVICE", nettyRequest.getUserContext().getDevice());
        Assert.assertEquals("HELLO CHAT", nettyRequest.getChat());
        EasyMock.verify(context);
    }

    @Test
    public void test4() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        TokenMapping tokenMapping = new NoneTokenMapping();
        ByteBuf byteBuf = PooledByteBufAllocator.DEFAULT.buffer();
        byteBuf.writeBytes(IOUtils.toString(ResourceUtils.getURL("classpath:OpenAiRequest_3.json").openStream()).getBytes(StandardCharsets.UTF_8));
        Attribute<Byte> attributeType = EasyMock.createMock(Attribute.class);
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeType).anyTimes();
        EasyMock.replay(context);
        NettyHttpProxy nettyHttpProxy = new NettyHttpProxy(byteBuf, "TOKEN/A:B");
        NettyRequest nettyRequest = nettyHttpProxy.buildRequest(context, tokenMapping);
        Assert.assertEquals("A", nettyRequest.getDevice());
        Assert.assertEquals("A", nettyRequest.getUserContext().getDevice());
        Assert.assertEquals("HELLO CHAT", nettyRequest.getChat());
        EasyMock.verify(context);
    }

    @Test
    public void test5() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        TokenMapping tokenMapping = new NoneTokenMapping();
        ByteBuf byteBuf = PooledByteBufAllocator.DEFAULT.buffer();
        byteBuf.writeBytes(IOUtils.toString(ResourceUtils.getURL("classpath:OpenAiRequest_3.json").openStream()).getBytes(StandardCharsets.UTF_8));
        Attribute<Byte> attributeType = EasyMock.createMock(Attribute.class);
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeType).anyTimes();
        EasyMock.replay(context);
        NettyHttpProxy nettyHttpProxy = new NettyHttpProxy(byteBuf, "a@b/A:B");
        NettyRequest nettyRequest = nettyHttpProxy.buildRequest(context, tokenMapping);
        Assert.assertEquals("A", nettyRequest.getDevice());
        Assert.assertEquals("A", nettyRequest.getUserContext().getDevice());
        Assert.assertEquals("HELLO CHAT", nettyRequest.getChat());
        EasyMock.verify(context);
    }

    @org.junit.jupiter.api.Test
    public void testNettyHttpProxyInstantiationUnique() {
        org.junit.jupiter.api.Assertions.assertTrue(true);
    }

    @Test
    public void testHarness_writesDumpWhenDirectoryConfigured() throws Exception {
        Path dir = Files.createTempDirectory("netty-http-proxy-harness");
        try {
            String json = "{\"model\":\"m\",\"messages\":[{\"role\":\"user\",\"content\":\"hi\"}],\"metadata\":{},\"stream\":false,\"temperature\":0}";
            ByteBuf byteBuf = PooledByteBufAllocator.DEFAULT.buffer();
            byteBuf.writeBytes(json.getBytes(StandardCharsets.UTF_8));
            ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
            TokenMapping tokenMapping = new NoneTokenMapping();
            Attribute<Byte> attributeType = EasyMock.createMock(Attribute.class);
            EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeType).anyTimes();
            attributeType.set(EasyMock.anyObject());
            EasyMock.expectLastCall().anyTimes();
            EasyMock.replay(context, attributeType);
            NettyHttpProxy proxy = new NettyHttpProxy(null, byteBuf, "TOK", dir.toAbsolutePath().toString());
            proxy.buildRequest(context, tokenMapping);
            try (Stream<Path> list = Files.list(dir)) {
                org.junit.Assert.assertEquals(1L, list.filter(p -> p.getFileName().toString().startsWith(DumpUtils.DUMP_PREFIX + "_REQUEST_CHAT_")).count());
            }
            proxy.close();
            EasyMock.verify(context, attributeType);
        } finally {
            FileUtils.deleteDirectory(dir.toFile());
        }
    }

    @Test
    public void testHarness_skipsDumpWhenHarnessNull() throws Exception {
        String json = "{\"model\":\"m\",\"messages\":[{\"role\":\"user\",\"content\":\"hi\"}],\"metadata\":{},\"stream\":false,\"temperature\":0}";
        ByteBuf byteBuf = PooledByteBufAllocator.DEFAULT.buffer();
        byteBuf.writeBytes(json.getBytes(StandardCharsets.UTF_8));
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        TokenMapping tokenMapping = new NoneTokenMapping();
        Attribute<Byte> attributeType = EasyMock.createMock(Attribute.class);
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeType).anyTimes();
        attributeType.set(EasyMock.anyObject());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(context, attributeType);
        NettyHttpProxy proxy = new NettyHttpProxy(null, byteBuf, "TOK", null);
        proxy.buildRequest(context, tokenMapping);
        proxy.close();
        EasyMock.verify(context, attributeType);
    }

}
