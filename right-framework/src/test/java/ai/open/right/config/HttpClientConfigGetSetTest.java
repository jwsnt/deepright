package ai.open.right.config;

import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.junit.Assert;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.InjectMocks;
import org.mockito.junit.MockitoJUnitRunner;

import static org.junit.Assert.assertEquals;

@RunWith(MockitoJUnitRunner.class)
public class HttpClientConfigGetSetTest {

    @InjectMocks
    private HttpClientConfig config;
    
    @Before
    public void setUp() {
        config.setBufferSend(Integer.valueOf(2048));
        config.setBufferRecv(Integer.valueOf(4096));
        config.setRequest4resource(Integer.valueOf(1000));
        config.setConnect4resource(Integer.valueOf(2000));
        config.setSocket4resource(Integer.valueOf(3000));
        config.setRequest4stream(Integer.valueOf(1100));
        config.setConnect4stream(Integer.valueOf(2100));
        config.setSocket4stream(Integer.valueOf(3100));
        config.setRequest4other(Integer.valueOf(1200));
        config.setConnect4other(Integer.valueOf(2200));
        config.setSocket4other(Integer.valueOf(3200));
        config.setRequest4tools(Integer.valueOf(1300));
        config.setConnect4tools(Integer.valueOf(2300));
        config.setSocket4tools(Integer.valueOf(3300));
        config.setRequest4once(Integer.valueOf(1400));
        config.setConnect4once(Integer.valueOf(2400));
        config.setSocket4once(Integer.valueOf(3400));
        config.setRouter(Integer.valueOf(50));
        config.setTotal(Integer.valueOf(150));
        Assert.assertNull(config.getResource());
        Assert.assertNull(config.getStream());
        Assert.assertNull(config.getTools());
        Assert.assertNull(config.getOther());
        Assert.assertNull(config.getOnce());
    }

    @Test
    public void testBufferSend() {
        assertEquals(Integer.valueOf(2048), config.getBufferSend());
        config.setBufferSend(Integer.valueOf(8192));
        assertEquals(Integer.valueOf(8192), config.getBufferSend());
    }

    @Test
    public void testBufferRecv() {
        assertEquals(Integer.valueOf(4096), config.getBufferRecv());
        config.setBufferRecv(Integer.valueOf(16384));
        assertEquals(Integer.valueOf(16384), config.getBufferRecv());
    }

    @Test
    public void testResourceTimeouts() {
        assertEquals(Integer.valueOf(1000), config.getRequest4resource());
        assertEquals(Integer.valueOf(2000), config.getConnect4resource());
        assertEquals(Integer.valueOf(3000), config.getSocket4resource());
        config.setRequest4resource(Integer.valueOf(1001));
        config.setConnect4resource(Integer.valueOf(2001));
        config.setSocket4resource(Integer.valueOf(3001));
        assertEquals(Integer.valueOf(1001), config.getRequest4resource());
        assertEquals(Integer.valueOf(2001), config.getConnect4resource());
        assertEquals(Integer.valueOf(3001), config.getSocket4resource());
    }

    @Test
    public void testStreamTimeouts() {
        assertEquals(Integer.valueOf(1100), config.getRequest4stream());
        assertEquals(Integer.valueOf(2100), config.getConnect4stream());
        assertEquals(Integer.valueOf(3100), config.getSocket4stream());
        config.setRequest4stream(Integer.valueOf(1101));
        config.setConnect4stream(Integer.valueOf(2101));
        config.setSocket4stream(Integer.valueOf(3101));
        assertEquals(Integer.valueOf(1101), config.getRequest4stream());
        assertEquals(Integer.valueOf(2101), config.getConnect4stream());
        assertEquals(Integer.valueOf(3101), config.getSocket4stream());
    }

    @Test
    public void testOtherTimeouts() {
        assertEquals(Integer.valueOf(1200), config.getRequest4other());
        assertEquals(Integer.valueOf(2200), config.getConnect4other());
        assertEquals(Integer.valueOf(3200), config.getSocket4other());
        config.setRequest4other(Integer.valueOf(1201));
        config.setConnect4other(Integer.valueOf(2201));
        config.setSocket4other(Integer.valueOf(3201));
        assertEquals(Integer.valueOf(1201), config.getRequest4other());
        assertEquals(Integer.valueOf(2201), config.getConnect4other());
        assertEquals(Integer.valueOf(3201), config.getSocket4other());
    }

    @Test
    public void testToolsTimeouts() {
        assertEquals(Integer.valueOf(1300), config.getRequest4tools());
        assertEquals(Integer.valueOf(2300), config.getConnect4tools());
        assertEquals(Integer.valueOf(3300), config.getSocket4tools());
        config.setRequest4tools(Integer.valueOf(1301));
        config.setConnect4tools(Integer.valueOf(2301));
        config.setSocket4tools(Integer.valueOf(3301));
        assertEquals(Integer.valueOf(1301), config.getRequest4tools());
        assertEquals(Integer.valueOf(2301), config.getConnect4tools());
        assertEquals(Integer.valueOf(3301), config.getSocket4tools());
    }

    @Test
    public void testOnceTimeouts() {
        assertEquals(Integer.valueOf(1400), config.getRequest4once());
        assertEquals(Integer.valueOf(2400), config.getConnect4once());
        assertEquals(Integer.valueOf(3400), config.getSocket4once());
        config.setRequest4once(Integer.valueOf(1401));
        config.setConnect4once(Integer.valueOf(2401));
        config.setSocket4once(Integer.valueOf(3401));
        assertEquals(Integer.valueOf(1401), config.getRequest4once());
        assertEquals(Integer.valueOf(2401), config.getConnect4once());
        assertEquals(Integer.valueOf(3401), config.getSocket4once());
    }

    @Test
    public void testRouterAndTotal() {
        assertEquals(Integer.valueOf(50), config.getRouter());
        assertEquals(Integer.valueOf(150), config.getTotal());
        config.setRouter(Integer.valueOf(60));
        config.setTotal(Integer.valueOf(160));
        assertEquals(Integer.valueOf(60), config.getRouter());
        assertEquals(Integer.valueOf(160), config.getTotal());
    }
}
