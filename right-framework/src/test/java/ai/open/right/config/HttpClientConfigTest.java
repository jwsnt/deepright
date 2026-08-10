package ai.open.right.config;

import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.junit.Assert;
import org.junit.Test;
import org.junit.jupiter.api.Assertions;

public class HttpClientConfigTest {

    public static final Integer NUMBER1024 = 1024;

    @Test
    public void testResource() throws Exception {
        HttpClientConfig config = new HttpClientConfig();
        config.setKeepalive(1000);
        config.setThreads(1024);
        Assert.assertEquals(Integer.valueOf(1000), config.getKeepalive());
        Assert.assertEquals(Integer.valueOf(1024), config.getThreads());
        config.setConnect4resource(HttpClientConfigTest.NUMBER1024);
        config.setRequest4resource(HttpClientConfigTest.NUMBER1024);
        config.setSocket4resource(HttpClientConfigTest.NUMBER1024);
        config.setSelectInterval(HttpClientConfigTest.NUMBER1024);
        config.setBufferRecv(HttpClientConfigTest.NUMBER1024);
        config.setBufferSend(HttpClientConfigTest.NUMBER1024);
        config.setRouter(HttpClientConfigTest.NUMBER1024);
        config.setTotal(HttpClientConfigTest.NUMBER1024);
        config.setExtreme(false);
        CloseableHttpAsyncClient client = config.resource();
        config.destroy();
        Assert.assertFalse(client.isRunning());
    }

    @Test
    public void testStream() throws Exception {
        HttpClientConfig config = new HttpClientConfig();
        config.setSelectInterval(HttpClientConfigTest.NUMBER1024);
        config.setConnect4stream(HttpClientConfigTest.NUMBER1024);
        config.setRequest4stream(HttpClientConfigTest.NUMBER1024);
        config.setSocket4stream(HttpClientConfigTest.NUMBER1024);
        config.setBufferRecv(HttpClientConfigTest.NUMBER1024);
        config.setBufferSend(HttpClientConfigTest.NUMBER1024);
        config.setRouter(HttpClientConfigTest.NUMBER1024);
        config.setTotal(HttpClientConfigTest.NUMBER1024);
        config.setExtreme(false);
        CloseableHttpAsyncClient client = config.stream();
        config.destroy();
        Assert.assertFalse(client.isRunning());
    }

    @Test
    public void testTools() throws Exception {
        HttpClientConfig config = new HttpClientConfig();
        config.setSelectInterval(HttpClientConfigTest.NUMBER1024);
        config.setSelectInterval(HttpClientConfigTest.NUMBER1024);
        config.setConnect4tools(HttpClientConfigTest.NUMBER1024);
        config.setRequest4tools(HttpClientConfigTest.NUMBER1024);
        config.setSocket4tools(HttpClientConfigTest.NUMBER1024);
        config.setBufferRecv(HttpClientConfigTest.NUMBER1024);
        config.setBufferSend(HttpClientConfigTest.NUMBER1024);
        config.setRouter(HttpClientConfigTest.NUMBER1024);
        config.setTotal(HttpClientConfigTest.NUMBER1024);
        config.setExtreme(false);
        CloseableHttpAsyncClient client = config.tools();
        config.destroy();
        Assert.assertFalse(client.isRunning());
    }

    @Test
    public void testOnce() throws Exception {
        HttpClientConfig config = new HttpClientConfig();
        config.setSelectInterval(HttpClientConfigTest.NUMBER1024);
        config.setConnect4once(HttpClientConfigTest.NUMBER1024);
        config.setRequest4once(HttpClientConfigTest.NUMBER1024);
        config.setSocket4once(HttpClientConfigTest.NUMBER1024);
        config.setBufferRecv(HttpClientConfigTest.NUMBER1024);
        config.setBufferSend(HttpClientConfigTest.NUMBER1024);
        config.setRouter(HttpClientConfigTest.NUMBER1024);
        config.setTotal(HttpClientConfigTest.NUMBER1024);
        config.setExtreme(false);
        CloseableHttpAsyncClient client = config.once();
        config.destroy();
        Assert.assertFalse(client.isRunning());
    }

    @Test
    public void testOther() throws Exception {
        HttpClientConfig config = new HttpClientConfig();
        config.setSelectInterval(HttpClientConfigTest.NUMBER1024);
        config.setConnect4other(HttpClientConfigTest.NUMBER1024);
        config.setRequest4other(HttpClientConfigTest.NUMBER1024);
        config.setSocket4other(HttpClientConfigTest.NUMBER1024);
        config.setBufferRecv(HttpClientConfigTest.NUMBER1024);
        config.setBufferSend(HttpClientConfigTest.NUMBER1024);
        config.setRouter(HttpClientConfigTest.NUMBER1024);
        config.setTotal(HttpClientConfigTest.NUMBER1024);
        config.setExtreme(false);
        CloseableHttpAsyncClient client = config.other();
        config.destroy();
        Assert.assertFalse(client.isRunning());
    }

    /**
     * extreme.enable=true 时 resource/stream/tools/other 均返回与 once 同一实例
     */
    @Test
    public void testExtremeReusesOnceForResourceStreamToolsOther() throws Exception {
        HttpClientConfig config = new HttpClientConfig();
        config.setKeepalive(1000);
        config.setSelectInterval(HttpClientConfigTest.NUMBER1024);
        config.setConnect4once(HttpClientConfigTest.NUMBER1024);
        config.setRequest4once(HttpClientConfigTest.NUMBER1024);
        config.setSocket4once(HttpClientConfigTest.NUMBER1024);
        config.setBufferRecv(HttpClientConfigTest.NUMBER1024);
        config.setBufferSend(HttpClientConfigTest.NUMBER1024);
        config.setRouter(HttpClientConfigTest.NUMBER1024);
        config.setTotal(HttpClientConfigTest.NUMBER1024);
        config.setExtreme(false);
        CloseableHttpAsyncClient onceClient = config.once();
        config.setExtreme(true);
        Assert.assertSame(onceClient, config.resource());
        Assert.assertSame(onceClient, config.stream());
        Assert.assertSame(onceClient, config.tools());
        Assert.assertSame(onceClient, config.other());
        config.destroy();
        Assert.assertFalse(onceClient.isRunning());
    }

    @Test
    public void testMonitor() throws Exception {
        HttpClientConfig config = new HttpClientConfig();
        config.setSelectInterval(HttpClientConfigTest.NUMBER1024);
        config.setConnect4resource(HttpClientConfigTest.NUMBER1024);
        config.setRequest4resource(HttpClientConfigTest.NUMBER1024);
        config.setSocket4resource(HttpClientConfigTest.NUMBER1024);
        config.setConnect4stream(HttpClientConfigTest.NUMBER1024);
        config.setRequest4stream(HttpClientConfigTest.NUMBER1024);
        config.setSocket4stream(HttpClientConfigTest.NUMBER1024);
        config.setConnect4tools(HttpClientConfigTest.NUMBER1024);
        config.setRequest4tools(HttpClientConfigTest.NUMBER1024);
        config.setSocket4tools(HttpClientConfigTest.NUMBER1024);
        config.setConnect4once(HttpClientConfigTest.NUMBER1024);
        config.setRequest4once(HttpClientConfigTest.NUMBER1024);
        config.setSocket4once(HttpClientConfigTest.NUMBER1024);
        config.setBufferRecv(HttpClientConfigTest.NUMBER1024);
        config.setBufferSend(HttpClientConfigTest.NUMBER1024);
        config.setRouter(HttpClientConfigTest.NUMBER1024);
        config.setTotal(HttpClientConfigTest.NUMBER1024);
        config.setExtreme(false);
        // 初始化各个客户端
        config.resource();
        config.stream();
        config.tools();
        config.once();

        // 调用 monitor 方法
        String monitorInfo = config.monitor();

        // 验证返回的字符串是否包含指定的子串
        Assert.assertTrue(monitorInfo.contains("The http Client(resource)="));
        Assert.assertTrue(monitorInfo.contains("The http Client(stream)="));
        Assert.assertTrue(monitorInfo.contains("The http Client(tools)="));
        Assert.assertTrue(monitorInfo.contains("The http Client(once)="));

        // 销毁客户端
        config.destroy();
    }

    /**
     * 新增 JUnit 5 测试方法，验证 monitor 输出内容
     */
    @org.junit.jupiter.api.Test
    public void testMonitorJUnit5() throws Exception {
        HttpClientConfig config = new HttpClientConfig();
        // 设置所有必要的超时和缓冲区字段，防止 NullPointerException
        config.setSelectInterval(HttpClientConfigTest.NUMBER1024);
        config.setConnect4resource(HttpClientConfigTest.NUMBER1024);
        config.setRequest4resource(HttpClientConfigTest.NUMBER1024);
        config.setSocket4resource(HttpClientConfigTest.NUMBER1024);
        config.setConnect4stream(HttpClientConfigTest.NUMBER1024);
        config.setRequest4stream(HttpClientConfigTest.NUMBER1024);
        config.setSocket4stream(HttpClientConfigTest.NUMBER1024);
        config.setConnect4tools(HttpClientConfigTest.NUMBER1024);
        config.setRequest4tools(HttpClientConfigTest.NUMBER1024);
        config.setSocket4tools(HttpClientConfigTest.NUMBER1024);
        config.setConnect4once(HttpClientConfigTest.NUMBER1024);
        config.setRequest4once(HttpClientConfigTest.NUMBER1024);
        config.setSocket4once(HttpClientConfigTest.NUMBER1024);
        config.setBufferRecv(HttpClientConfigTest.NUMBER1024);
        config.setBufferSend(HttpClientConfigTest.NUMBER1024);
        config.setRouter(HttpClientConfigTest.NUMBER1024);
        config.setTotal(HttpClientConfigTest.NUMBER1024);
        config.setSelectInterval(HttpClientConfigTest.NUMBER1024);
        config.setExtreme(false);
        // 初始化各个客户端
        config.resource();
        config.stream();
        config.tools();
        config.once();

        // 调用 monitor 方法
        String monitorInfo = config.monitor();
        Assertions.assertEquals(HttpClientConfigTest.NUMBER1024, config.getSelectInterval());
        // 验证返回的字符串是否包含指定的子串
        Assertions.assertTrue(monitorInfo.contains("The http Client(resource)="));
        Assertions.assertTrue(monitorInfo.contains("The http Client(stream)="));
        Assertions.assertTrue(monitorInfo.contains("The http Client(tools)="));
        Assertions.assertTrue(monitorInfo.contains("The http Client(once)="));

        // 销毁客户端
        config.destroy();
    }

    @org.junit.jupiter.api.Test
    public void testHttpClientConfig() {
        HttpClientConfig config = new HttpClientConfig();
        org.junit.jupiter.api.Assertions.assertNotNull(config);
    }

    @org.junit.jupiter.api.Test
    public void testCustomKeepAliveStrategy() {
        HttpClientConfig.CustomKeepAliveStrategy strategy = new HttpClientConfig.CustomKeepAliveStrategy(5000);
        org.apache.http.HttpResponse response = org.easymock.EasyMock.createMock(org.apache.http.HttpResponse.class);
        org.apache.http.protocol.HttpContext context = org.easymock.EasyMock.createMock(org.apache.http.protocol.HttpContext.class);
        // 模拟 headerIterator 调用，返回一个空的迭代器
        org.easymock.EasyMock.expect(response.headerIterator("Keep-Alive")).andReturn(new org.apache.http.message.BasicHeaderIterator(new org.apache.http.Header[0], null)).anyTimes();
        org.easymock.EasyMock.replay(response, context);
        long duration = strategy.getKeepAliveDuration(response, context);
        org.junit.jupiter.api.Assertions.assertEquals(5000, duration);
    }

}

