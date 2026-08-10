package ai.open.right.netty.a2a;

import ai.open.right.utils.DumpUtils;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.PooledByteBufAllocator;
import io.netty.util.ReferenceCountUtil;
import io.netty.handler.codec.http.DefaultHttpHeaders;
import io.netty.handler.codec.http.FullHttpRequest;
import io.netty.handler.codec.http.HttpMethod;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.apache.commons.io.FileUtils;
import org.springframework.util.ResourceUtils;

import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.stream.Stream;

public class NettyInputProxyTest {

    /**
     * POST 解析失败时构造内 catch 会 close()，抵消 retain，引用计数应回到构造前（用 mock 跟踪 refCnt，避免不同 Netty 版本下流关闭行为差异）。
     */
    @Test
    public void testConstructor_releasesWhenPostBodyJsonInvalid() throws Exception {
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf.writeBytes("{not-json".getBytes(StandardCharsets.UTF_8));
        AtomicInteger rc = new AtomicInteger(1);
        FullHttpRequest req = EasyMock.createMock(FullHttpRequest.class);
        EasyMock.expect(req.retain()).andAnswer(() -> {
            rc.incrementAndGet();
            return req;
        }).anyTimes();
        EasyMock.expect(req.method()).andReturn(HttpMethod.POST).anyTimes();
        EasyMock.expect(req.content()).andReturn(buf).anyTimes();
        EasyMock.expect(req.refCnt()).andAnswer(rc::get).anyTimes();
        EasyMock.expect(req.release()).andAnswer(() -> {
            rc.decrementAndGet();
            return true;
        }).anyTimes();
        EasyMock.replay(req);
        try {
            new NettyInputProxy(req);
            Assert.fail("expected parse failure");
        } catch (Exception ignored) {
        }
        Assert.assertEquals(1, rc.get());
        EasyMock.verify(req);
        while (buf.refCnt() > 0) {
            ReferenceCountUtil.release(buf);
        }
    }

    @Test
    public void test() throws Exception {
        String json = IOUtils.toString(ResourceUtils.getURL("classpath:MCP_Prompt_list_response.json").openStream(), StandardCharsets.UTF_8);
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf = buf.writeBytes(json.getBytes(StandardCharsets.UTF_8));
        FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
        EasyMock.expect(fullHttpRequest.retain()).andReturn(fullHttpRequest).anyTimes();
        EasyMock.expect(fullHttpRequest.method()).andReturn(HttpMethod.POST).anyTimes();
        EasyMock.expect(fullHttpRequest.content()).andReturn(buf).anyTimes();
        DefaultHttpHeaders headers = new DefaultHttpHeaders();
        headers.add("HELLO", "WORLD");
        EasyMock.expect(fullHttpRequest.headers()).andReturn(headers).anyTimes();
        EasyMock.expect(fullHttpRequest.refCnt()).andReturn(1).anyTimes();
        EasyMock.expect(fullHttpRequest.release()).andReturn(true).anyTimes();
        EasyMock.replay(fullHttpRequest);
        NettyInputProxy proxy = new NettyInputProxy(fullHttpRequest);
        Assert.assertNotNull(proxy.getContent());
        Assert.assertEquals(fullHttpRequest, proxy.getRequest());
        Assert.assertEquals("WORLD", proxy.initHeaders().get("HELLO"));
        Assert.assertSame(proxy.getContent(), proxy.initContent());
        proxy.close();
        EasyMock.verify(fullHttpRequest);
    }

    @Test
    public void test2() throws Exception {
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf = buf.writeBytes("{}".getBytes(StandardCharsets.UTF_8));
        FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
        EasyMock.expect(fullHttpRequest.retain()).andReturn(fullHttpRequest).anyTimes();
        EasyMock.expect(fullHttpRequest.method()).andReturn(HttpMethod.POST).anyTimes();
        EasyMock.expect(fullHttpRequest.content()).andReturn(buf).anyTimes();
        EasyMock.expect(fullHttpRequest.refCnt()).andReturn(1).anyTimes();
        EasyMock.expect(fullHttpRequest.release()).andReturn(true).anyTimes();
        EasyMock.replay(fullHttpRequest);
        NettyInputProxy proxy = new NettyInputProxy(fullHttpRequest);
        Assert.assertNotNull(proxy.getContent());
        Assert.assertNotNull(proxy.initContent());
        Assert.assertTrue(proxy.getContent().isEmpty());
        proxy.close();
        EasyMock.verify(fullHttpRequest);
    }

    /**
     * 覆盖非 POST 请求时 content = null 的分支（else 分支）
     */
    @Test
    public void testNonPostContentNull() throws Exception {
        FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
        EasyMock.expect(fullHttpRequest.retain()).andReturn(fullHttpRequest).anyTimes();
        EasyMock.expect(fullHttpRequest.method()).andReturn(HttpMethod.GET).anyTimes();
        EasyMock.expect(fullHttpRequest.refCnt()).andReturn(1).anyTimes();
        EasyMock.expect(fullHttpRequest.release()).andReturn(true).anyTimes();
        EasyMock.replay(fullHttpRequest);
        NettyInputProxy proxy = new NettyInputProxy(fullHttpRequest);
        Assert.assertNull(proxy.getContent());
        Assert.assertNull(proxy.initContent());
        proxy.close();
        EasyMock.verify(fullHttpRequest);
    }

    @Test
    public void testHarness_writesRequestDumpWhenPostAndHarnessDirSet() throws Exception {
        Path dir = Files.createTempDirectory("a2a-harness-dump");
        try {
            String json = "{\"harnessKey\":42}";
            ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
            buf.writeBytes(json.getBytes(StandardCharsets.UTF_8));
            FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
            EasyMock.expect(fullHttpRequest.retain()).andReturn(fullHttpRequest).anyTimes();
            EasyMock.expect(fullHttpRequest.method()).andReturn(HttpMethod.POST).anyTimes();
            EasyMock.expect(fullHttpRequest.content()).andReturn(buf).anyTimes();
            EasyMock.expect(fullHttpRequest.refCnt()).andReturn(1).anyTimes();
            EasyMock.expect(fullHttpRequest.release()).andReturn(true).anyTimes();
            EasyMock.replay(fullHttpRequest);
            NettyInputProxy proxy = new NettyInputProxy(fullHttpRequest, dir.toAbsolutePath().toString());
            Assert.assertNotNull(proxy.getContent());
            Path dumped;
            try (Stream<Path> list = Files.list(dir)) {
                dumped = list.filter(p -> p.getFileName().toString().startsWith(DumpUtils.DUMP_PREFIX + "_REQUEST_A2A_")).findFirst()
                        .orElseThrow(() -> new AssertionError("expected REQUEST_A2A dump file"));
            }
            String written = new String(Files.readAllBytes(dumped), StandardCharsets.UTF_8);
            Assert.assertTrue(written.contains("harnessKey"));
            proxy.close();
            EasyMock.verify(fullHttpRequest);
        } finally {
            FileUtils.deleteDirectory(dir.toFile());
        }
    }

    @Test
    public void testHarness_skipsDumpWhenHarnessNullOrBlank() throws Exception {
        for (String autodumpDir : new String[]{null, ""}) {
            ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
            buf.writeBytes("{}".getBytes(StandardCharsets.UTF_8));
            FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
            EasyMock.expect(fullHttpRequest.retain()).andReturn(fullHttpRequest).anyTimes();
            EasyMock.expect(fullHttpRequest.method()).andReturn(HttpMethod.POST).anyTimes();
            EasyMock.expect(fullHttpRequest.content()).andReturn(buf).anyTimes();
            EasyMock.expect(fullHttpRequest.refCnt()).andReturn(1).anyTimes();
            EasyMock.expect(fullHttpRequest.release()).andReturn(true).anyTimes();
            EasyMock.replay(fullHttpRequest);
            NettyInputProxy proxy = new NettyInputProxy(fullHttpRequest, autodumpDir);
            Assert.assertNotNull(proxy.getContent());
            proxy.close();
            EasyMock.verify(fullHttpRequest);
        }
    }

    /**
     * 非 POST 不解析 body，也不走 autodump，即使配置了目录也不会在别处创建文件（目录保持空）。
     */
    @Test
    public void testHarness_nonPostDoesNotWriteDump() throws Exception {
        Path dir = Files.createTempDirectory("a2a-harness-get");
        try {
            FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
            EasyMock.expect(fullHttpRequest.retain()).andReturn(fullHttpRequest).anyTimes();
            EasyMock.expect(fullHttpRequest.method()).andReturn(HttpMethod.GET).anyTimes();
            EasyMock.expect(fullHttpRequest.refCnt()).andReturn(1).anyTimes();
            EasyMock.expect(fullHttpRequest.release()).andReturn(true).anyTimes();
            EasyMock.replay(fullHttpRequest);
            new NettyInputProxy(fullHttpRequest, dir.toAbsolutePath().toString()).close();
            try (Stream<Path> list = Files.list(dir)) {
                Assert.assertFalse(list.findAny().isPresent());
            }
            EasyMock.verify(fullHttpRequest);
        } finally {
            FileUtils.deleteDirectory(dir.toFile());
        }
    }
}
