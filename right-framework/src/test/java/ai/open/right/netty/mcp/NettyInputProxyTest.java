package ai.open.right.netty.mcp;

import io.netty.buffer.ByteBuf;
import io.netty.buffer.PooledByteBufAllocator;
import io.netty.util.ReferenceCountUtil;
import io.netty.handler.codec.http.DefaultHttpHeaders;
import io.netty.handler.codec.http.FullHttpRequest;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.nio.charset.StandardCharsets;
import java.util.concurrent.atomic.AtomicInteger;

public class NettyInputProxyTest {

    /**
     * JSON 解析失败时构造内 catch 会 close()，抵消 retain，引用计数应回到构造前。
     */
    @Test
    public void testConstructor_releasesWhenBodyJsonInvalid() throws Exception {
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf.writeBytes("{not-json".getBytes(StandardCharsets.UTF_8));
        AtomicInteger rc = new AtomicInteger(1);
        FullHttpRequest req = EasyMock.createMock(FullHttpRequest.class);
        EasyMock.expect(req.retain()).andAnswer(() -> {
            rc.incrementAndGet();
            return req;
        }).anyTimes();
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
        EasyMock.expect(fullHttpRequest.content()).andReturn(buf).anyTimes();
        DefaultHttpHeaders headers = new DefaultHttpHeaders();
        headers.add("HELLO", "WORLD");
        EasyMock.expect(fullHttpRequest.headers()).andReturn(headers).anyTimes();
        EasyMock.expect(fullHttpRequest.refCnt()).andReturn(1).anyTimes();
        EasyMock.expect(fullHttpRequest.release()).andReturn(true).anyTimes();
        EasyMock.replay(fullHttpRequest);
        NettyInputProxy proxy = new NettyInputProxy(fullHttpRequest);
        Assert.assertNotNull(proxy.getContent());
        Assert.assertSame(proxy.getContent(), proxy.initContent());
        Assert.assertEquals(fullHttpRequest, proxy.getRequest());
        Assert.assertEquals("WORLD", proxy.initHeaders().get("HELLO"));
        proxy.close();
        EasyMock.verify(fullHttpRequest);
    }

    /**
     * 覆盖 protected final Map&lt;String, Object&gt; content：POST 解析后 content 非空且与 initContent 一致
     */
    @Test
    public void testContentFromJson() throws Exception {
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf = buf.writeBytes("{\"id\":\"1\",\"method\":\"test\"}".getBytes(StandardCharsets.UTF_8));
        FullHttpRequest fullHttpRequest = EasyMock.createMock(FullHttpRequest.class);
        EasyMock.expect(fullHttpRequest.retain()).andReturn(fullHttpRequest).anyTimes();
        EasyMock.expect(fullHttpRequest.content()).andReturn(buf).anyTimes();
        EasyMock.expect(fullHttpRequest.refCnt()).andReturn(1).anyTimes();
        EasyMock.expect(fullHttpRequest.release()).andReturn(true).anyTimes();
        EasyMock.replay(fullHttpRequest);
        NettyInputProxy proxy = new NettyInputProxy(fullHttpRequest);
        Assert.assertNotNull(proxy.getContent());
        Assert.assertEquals("1", proxy.getContent().get("id"));
        Assert.assertEquals("test", proxy.getContent().get("method"));
        Assert.assertSame(proxy.getContent(), proxy.initContent());
        proxy.close();
        EasyMock.verify(fullHttpRequest);
    }
}
