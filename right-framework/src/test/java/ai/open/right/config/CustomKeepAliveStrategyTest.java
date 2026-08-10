package ai.open.right.config;

import org.apache.http.Header;
import org.apache.http.HttpResponse;
import org.apache.http.message.BasicHeader;
import org.apache.http.protocol.HttpContext;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.mockito.Mockito;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.mockito.Mockito.when;

public class CustomKeepAliveStrategyTest {

    private HttpClientConfig.CustomKeepAliveStrategy strategy;

    private final Integer MAX_KEEP_ALIVE = 5000; // 设定最大限制为 5 秒

    private HttpResponse response;
    private HttpContext context;

    @BeforeEach
    public void setUp() {
        strategy = new HttpClientConfig.CustomKeepAliveStrategy(MAX_KEEP_ALIVE);
        response = Mockito.mock(HttpResponse.class);
        context = Mockito.mock(HttpContext.class);
    }

    @Test
    @DisplayName("场景1：服务器没有返回 Keep-Alive 标头，应返回预设的最大超时时间")
    public void shouldReturnDefaultTimeoutWhenNoHeaderPresent() {
        // 模拟没有任何相关 Header
        when(response.headerIterator("Keep-Alive")).thenReturn(new org.apache.http.message.BasicHeaderIterator(new Header[]{}, null));

        Long duration = strategy.getKeepAliveDuration(response, context);

        assertEquals(MAX_KEEP_ALIVE, duration.intValue(), "当服务器未指定超时，应使用自定义默认值");
    }

    @Test
    @DisplayName("场景2：服务器返回的超时时间在合理范围内，应尊重服务器设置")
    public void shouldReturnServerTimeoutWhenWithinRange() {
        // 模拟服务器返回 Keep-Alive: timeout=2
        Header keepAliveHeader = new BasicHeader("Keep-Alive", "timeout=2");
        when(response.headerIterator("Keep-Alive")).thenReturn(new org.apache.http.message.BasicHeaderIterator(new Header[]{keepAliveHeader}, "Keep-Alive"));

        Long duration = strategy.getKeepAliveDuration(response, context);

        assertEquals(2000, duration.intValue(), "当服务器设置在限制范围内，应使用服务器的时间（2000ms）");
    }

    @Test
    @DisplayName("场景3：服务器返回的超时时间过长，应被强制修剪为最大限制")
    public void shouldReturnMaxTimeoutWhenServerTimeoutIsTooLong() {
        // 模拟服务器要求保持 60 秒连接 Keep-Alive: timeout=60
        Header keepAliveHeader = new BasicHeader("Keep-Alive", "timeout=60");
        when(response.headerIterator("Keep-Alive")).thenReturn(new org.apache.http.message.BasicHeaderIterator(new Header[]{keepAliveHeader}, "Keep-Alive"));

        Long duration = strategy.getKeepAliveDuration(response, context);

        assertEquals(MAX_KEEP_ALIVE, duration.intValue(), "服务器给的时间太长，应被强制限制在 MAX_KEEP_ALIVE");
    }
}