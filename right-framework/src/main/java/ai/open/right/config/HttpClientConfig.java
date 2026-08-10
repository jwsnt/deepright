package ai.open.right.config;

import jakarta.annotation.PreDestroy;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.reflect.FieldUtils;
import org.apache.http.HttpRequest;
import org.apache.http.HttpResponse;
import org.apache.http.HttpStatus;
import org.apache.http.ProtocolException;
import org.apache.http.client.config.RequestConfig;
import org.apache.http.conn.ConnectionKeepAliveStrategy;
import org.apache.http.impl.DefaultConnectionReuseStrategy;
import org.apache.http.impl.client.DefaultConnectionKeepAliveStrategy;
import org.apache.http.impl.client.DefaultRedirectStrategy;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.apache.http.impl.nio.client.HttpAsyncClientBuilder;
import org.apache.http.impl.nio.conn.PoolingNHttpClientConnectionManager;
import org.apache.http.impl.nio.reactor.IOReactorConfig;
import org.apache.http.protocol.HttpContext;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.DependsOn;
import org.springframework.context.annotation.PropertySource;
import org.springframework.scheduling.annotation.Scheduled;

@PropertySource("classpath:right-thread.properties")
@Configuration
@Slf4j
@Setter
@Getter
public class HttpClientConfig {

    // 资源
    protected CloseableHttpAsyncClient resource;

    // 流式
    protected CloseableHttpAsyncClient stream;

    // Tools
    protected CloseableHttpAsyncClient tools;

    // Other
    protected CloseableHttpAsyncClient other;

    // 非流式
    protected CloseableHttpAsyncClient once;

    @Value("${httpclient.selectInterval:1000}")
    protected Integer selectInterval;

    // Http Client的发送Buffer
    @Value("${httpclient.buffer.send:65536}")
    protected Integer bufferSend;

    // Http Client的接收Buffer
    @Value("${httpclient.buffer.recv:65536}")
    protected Integer bufferRecv;

    /**
     * Request Timeout
     */
    @Value("${httpclient.timeout.resource.request:1000}")
    // Resource客户端的Request超时
    protected Integer request4resource;

    /**
     * Connect Timeout
     */
    @Value("${httpclient.timeout.resource.connect:2000}")
    // Resource客户端的Connect超时
    protected Integer connect4resource;

    /**
     * Socket Timeout
     */
    @Value("${httpclient.timeout.resource.socket:10000}")
    // Resource客户端的Socket超时
    protected Integer socket4resource;

    /**
     * Request Timeout
     */
    @Value("${httpclient.timeout.stream.request:1000}")
    // Stream客户端的Request超时
    protected Integer request4stream;

    /**
     * Connect Timeout
     */
    @Value("${httpclient.timeout.stream.connect:2000}")
    // Stream客户端的Connect超时
    protected Integer connect4stream;

    /**
     * Socket Timeout
     */
    @Value("${httpclient.timeout.stream.socket:10000}")
    // Stream客户端的Socket超时
    protected Integer socket4stream;

    /**
     * Request Timeout
     */
    @Value("${httpclient.timeout.other.request:1000}")
    // Other客户端的Request超时
    protected Integer request4other;

    /**
     * Connect Timeout
     */
    @Value("${httpclient.timeout.other.connect:2000}")
    // Other客户端的Connect超时
    protected Integer connect4other;

    /**
     * Socket Timeout
     */
    @Value("${httpclient.timeout.other.socket:10000}")
    // Other客户端的Socket超时
    protected Integer socket4other;

    /**
     * Request Timeout
     */
    @Value("${httpclient.timeout.tools.request:1000}")
    // Tools客户端的Request超时
    protected Integer request4tools;

    /**
     * Connect Timeout
     */
    @Value("${httpclient.timeout.tools.connect:2000}")
    // Tools客户端的Connect超时
    protected Integer connect4tools;

    /**
     * Socket Timeout
     */
    @Value("${httpclient.timeout.tools.socket:10000}")
    // Tools客户端的Socket超时
    protected Integer socket4tools;

    /**
     * Request Timeout
     */
    @Value("${httpclient.timeout.once.request:1000}")
    // Once客户端的Request超时
    protected Integer request4once;

    /**
     * Connect Timeout
     */
    @Value("${httpclient.timeout.once.connect:2000}")
    // Once客户端的Connect超时
    protected Integer connect4once;

    /**
     * Socket Timeout
     */
    @Value("${httpclient.timeout.once.socket:600000}")
    // Once客户端的Socket超时
    protected Integer socket4once;

    /**
     * KeepAlive
     */
    @Value("${httpclient.keepalive:300000}")
    // Resource客户端的Request超时
    protected Integer keepalive;


    @Value("${extreme.enable:false}")
    protected Boolean extreme;

    /**
     * IO Thread（默认8）
     */
    @Value("${httpclient.threads:8}")
    // Resource客户端的Request超时
    protected Integer threads;

    /**
     * Connection per router
     */
    @Value("${httpclient.router:2000}")
    // 单Host最大连接
    protected Integer router;

    /**
     * Connection total
     */
    @Value("${httpclient.total:2000}")
    // 总连接
    protected Integer total;


    /// /////////////////////////////////////////////////////////////
    /// ConnectionRequestTimeout: 向连接池申请可用连接
    /// ConnectTimeout: 建立TCP连接
    /// SocketTimeout: 传输/读取数据（两次数据包到达之间的最大间隔时间）
    /// /////////////////////////////////////////////////////////////
    @Bean("resource")
    @DependsOn("once")
    @ConditionalOnMissingBean(name = "resource")
    public CloseableHttpAsyncClient resource() throws Exception {
        if (!this.extreme) {
            RequestConfig.Builder rconfig_builder = RequestConfig.custom();
            rconfig_builder.setConnectionRequestTimeout(this.request4resource);
            rconfig_builder.setConnectTimeout(this.connect4resource);
            rconfig_builder.setSocketTimeout(this.socket4resource);
            RequestConfig rConfig = rconfig_builder.build();
            IOReactorConfig.Builder iconfig_builder = IOReactorConfig.custom();
            iconfig_builder.setIoThreadCount(this.threads != null ? this.threads : Runtime.getRuntime().availableProcessors());
            iconfig_builder.setConnectTimeout(this.connect4resource);
            iconfig_builder.setSelectInterval(this.selectInterval);
            iconfig_builder.setSoTimeout(this.socket4resource);
            iconfig_builder.setRcvBufSize(this.bufferRecv);
            iconfig_builder.setSndBufSize(this.bufferSend);
            iconfig_builder.setSoKeepAlive(true);
            iconfig_builder.setTcpNoDelay(true);
            IOReactorConfig iConfig = iconfig_builder.build();
            log.info("The resource connect timeout={}", this.connect4resource);
            log.info("The resource request timeout={}", this.request4resource);
            log.info("The resource socket timeout={}", this.socket4resource);
            log.info("The resource conns total={}", this.total);
            HttpAsyncClientBuilder builder = HttpAsyncClientBuilder.create();
            builder.setConnectionReuseStrategy(DefaultConnectionReuseStrategy.INSTANCE);
            builder.setKeepAliveStrategy(new CustomKeepAliveStrategy(this.keepalive));
            builder.setRedirectStrategy(CustomRedirectStrategy.INSTANCE);
            builder.setDefaultIOReactorConfig(iConfig);
            builder.setDefaultRequestConfig(rConfig);
            builder.setMaxConnPerRoute(this.router);
            builder.setMaxConnTotal(this.total);
            this.resource = builder.build();
            this.resource.start();
            log.info("The resource start ... ");
            return this.resource;
        } else {
            log.info("The resource bean reuses once client (extreme.enable=true) ...");
            return this.once;
        }
    }

    @Bean("stream")
    @DependsOn("once")
    @ConditionalOnMissingBean(name = "stream")
    public CloseableHttpAsyncClient stream() throws Exception {
        if (!this.extreme) {
            RequestConfig.Builder rconfig_builder = RequestConfig.custom();
            rconfig_builder.setConnectionRequestTimeout(this.request4stream);
            rconfig_builder.setConnectTimeout(this.connect4stream);
            rconfig_builder.setSocketTimeout(this.socket4stream);
            RequestConfig rConfig = rconfig_builder.build();
            IOReactorConfig.Builder iconfig_builder = IOReactorConfig.custom();
            iconfig_builder.setIoThreadCount(this.threads != null ? this.threads : Runtime.getRuntime().availableProcessors());
            iconfig_builder.setConnectTimeout(this.connect4stream);
            iconfig_builder.setSelectInterval(this.selectInterval);
            iconfig_builder.setSoTimeout(this.socket4stream);
            iconfig_builder.setRcvBufSize(this.bufferRecv);
            iconfig_builder.setSndBufSize(this.bufferSend);
            iconfig_builder.setSoKeepAlive(true);
            iconfig_builder.setTcpNoDelay(true);
            IOReactorConfig iConfig = iconfig_builder.build();
            log.info("The stream connect timeout={}", this.connect4stream);
            log.info("The stream request timeout={}", this.request4stream);
            log.info("The stream socket timeout={}", this.socket4stream);
            log.info("The stream conns total={}", this.total);
            HttpAsyncClientBuilder builder = HttpAsyncClientBuilder.create();
            builder.setConnectionReuseStrategy(DefaultConnectionReuseStrategy.INSTANCE);
            builder.setKeepAliveStrategy(new CustomKeepAliveStrategy(this.keepalive));
            builder.setRedirectStrategy(CustomRedirectStrategy.INSTANCE);
            builder.setDefaultIOReactorConfig(iConfig);
            builder.setDefaultRequestConfig(rConfig);
            builder.setMaxConnPerRoute(this.router);
            builder.setMaxConnTotal(this.total);
            this.stream = builder.build();
            this.stream.start();
            log.info("The stream start ... ");
            return this.stream;
        } else {
            log.info("The stream bean reuses once client (extreme.enable=true) ...");
            return this.once;
        }
    }

    @Bean("tools")
    @DependsOn("once")
    @ConditionalOnMissingBean(name = "tools")
    public CloseableHttpAsyncClient tools() throws Exception {
        if (!this.extreme) {
            RequestConfig.Builder rconfig_builder = RequestConfig.custom();
            rconfig_builder.setConnectionRequestTimeout(this.request4tools);
            rconfig_builder.setConnectTimeout(this.connect4tools);
            rconfig_builder.setSocketTimeout(this.socket4tools);
            RequestConfig rConfig = rconfig_builder.build();
            IOReactorConfig.Builder iconfig_builder = IOReactorConfig.custom();
            iconfig_builder.setIoThreadCount(this.threads != null ? this.threads : Runtime.getRuntime().availableProcessors());
            iconfig_builder.setSelectInterval(this.selectInterval);
            iconfig_builder.setConnectTimeout(this.connect4tools);
            iconfig_builder.setSoTimeout(this.socket4tools);
            iconfig_builder.setRcvBufSize(this.bufferRecv);
            iconfig_builder.setSndBufSize(this.bufferSend);
            iconfig_builder.setSoKeepAlive(true);
            iconfig_builder.setTcpNoDelay(true);
            IOReactorConfig iConfig = iconfig_builder.build();
            log.info("The tools connect timeout={}", this.connect4tools);
            log.info("The tools request timeout={}", this.request4tools);
            log.info("The tools socket timeout={}", this.socket4tools);
            log.info("The tools conns total={}", this.total);
            HttpAsyncClientBuilder builder = HttpAsyncClientBuilder.create();
            builder.setConnectionReuseStrategy(DefaultConnectionReuseStrategy.INSTANCE);
            builder.setKeepAliveStrategy(new CustomKeepAliveStrategy(this.keepalive));
            builder.setRedirectStrategy(CustomRedirectStrategy.INSTANCE);
            builder.setDefaultIOReactorConfig(iConfig);
            builder.setDefaultRequestConfig(rConfig);
            builder.setMaxConnPerRoute(this.router);
            builder.setMaxConnTotal(this.total);
            this.tools = builder.build();
            this.tools.start();
            log.info("The tools start ... ");
            return this.tools;
        } else {
            log.info("The tools bean reuses once client (extreme.enable=true) ...");
            return this.once;
        }
    }

    @Bean("other")
    @DependsOn("once")
    @ConditionalOnMissingBean(name = "other")
    public CloseableHttpAsyncClient other() throws Exception {
        if (!this.extreme) {
            RequestConfig.Builder rconfig_builder = RequestConfig.custom();
            rconfig_builder.setConnectionRequestTimeout(this.request4other);
            rconfig_builder.setConnectTimeout(this.connect4other);
            rconfig_builder.setSocketTimeout(this.socket4other);
            RequestConfig rConfig = rconfig_builder.build();
            IOReactorConfig.Builder iconfig_builder = IOReactorConfig.custom();
            iconfig_builder.setIoThreadCount(this.threads != null ? this.threads : Runtime.getRuntime().availableProcessors());
            iconfig_builder.setSelectInterval(this.selectInterval);
            iconfig_builder.setConnectTimeout(this.connect4other);
            iconfig_builder.setSoTimeout(this.socket4other);
            iconfig_builder.setRcvBufSize(this.bufferRecv);
            iconfig_builder.setSndBufSize(this.bufferSend);
            iconfig_builder.setSoKeepAlive(true);
            iconfig_builder.setTcpNoDelay(true);
            IOReactorConfig iConfig = iconfig_builder.build();
            log.info("The other connect timeout={}", this.connect4other);
            log.info("The other request timeout={}", this.request4other);
            log.info("The other socket timeout={}", this.socket4other);
            log.info("The other conns total={}", this.total);
            HttpAsyncClientBuilder builder = HttpAsyncClientBuilder.create();
            builder.setConnectionReuseStrategy(DefaultConnectionReuseStrategy.INSTANCE);
            builder.setKeepAliveStrategy(new CustomKeepAliveStrategy(this.keepalive));
            builder.setRedirectStrategy(CustomRedirectStrategy.INSTANCE);
            builder.setDefaultIOReactorConfig(iConfig);
            builder.setDefaultRequestConfig(rConfig);
            builder.setMaxConnPerRoute(this.router);
            builder.setMaxConnTotal(this.total);
            this.other = builder.build();
            this.other.start();
            log.info("The other start ... ");
            return this.other;
        } else {
            log.info("The other bean reuses once client (extreme.enable=true) ...");
            return this.once;
        }
    }

    @Bean("once")
    @ConditionalOnMissingBean(name = "once")
    public CloseableHttpAsyncClient once() throws Exception {
        RequestConfig.Builder rconfig_builder = RequestConfig.custom();
        rconfig_builder.setConnectionRequestTimeout(this.request4once);
        rconfig_builder.setConnectTimeout(this.connect4once);
        rconfig_builder.setSocketTimeout(this.socket4once);
        RequestConfig rConfig = rconfig_builder.build();
        IOReactorConfig.Builder iconfig_builder = IOReactorConfig.custom();
        iconfig_builder.setIoThreadCount(this.threads != null ? this.threads : Runtime.getRuntime().availableProcessors());
        iconfig_builder.setSelectInterval(this.selectInterval);
        iconfig_builder.setConnectTimeout(this.connect4once);
        iconfig_builder.setSoTimeout(this.socket4once);
        iconfig_builder.setRcvBufSize(this.bufferRecv);
        iconfig_builder.setSndBufSize(this.bufferSend);
        iconfig_builder.setSoKeepAlive(true);
        iconfig_builder.setTcpNoDelay(true);
        IOReactorConfig iConfig = iconfig_builder.build();
        log.info("The once connect timeout={}", this.connect4once);
        log.info("The once request timeout={}", this.request4once);
        log.info("The once socket timeout={}", this.socket4once);
        log.info("The once conns total={}", this.total);
        HttpAsyncClientBuilder builder = HttpAsyncClientBuilder.create();
        builder.setConnectionReuseStrategy(DefaultConnectionReuseStrategy.INSTANCE);
        builder.setKeepAliveStrategy(new CustomKeepAliveStrategy(this.keepalive));
        builder.setRedirectStrategy(CustomRedirectStrategy.INSTANCE);
        builder.setDefaultIOReactorConfig(iConfig);
        builder.setDefaultRequestConfig(rConfig);
        builder.setMaxConnPerRoute(this.router);
        builder.setMaxConnTotal(this.total);
        this.once = builder.build();
        this.once.start();
        log.info("The once start ... ");
        return this.once;
    }

    @PreDestroy
    public void destroy() throws Exception {
        if (!this.extreme && this.resource != null) {
            this.resource.close();
            this.resource = null;
            log.info("The resource closed ...");
        }
        if (!this.extreme && this.stream != null) {
            this.stream.close();
            this.stream = null;
            log.info("The stream closed ...");
        }
        if (!this.extreme && this.other != null) {
            this.other.close();
            this.other = null;
            log.info("Other closed ...");
        }
        if (!this.extreme && this.tools != null) {
            this.tools.close();
            this.tools = null;
            log.info("The tools closed ...");
        }
        if (this.once != null) {
            this.once.close();
            this.once = null;
            log.info("The once closed ...");
        }
    }

    @Scheduled(initialDelayString = "${monitor.httpclient.initialDelay:30000}", fixedRateString = "${monitor.httpclient.fixedRate:30000}")
    public String monitor() throws Exception {
        StringBuffer content = new StringBuffer();
        if (!this.extreme) {
            // Leased (已占用 / 租借中)
            // Pending (等待中 / 阻塞中)，由于连接池已达到上限，无法获取新连接，正在排队等待可用连接的请求数量（重点指标）
            // Available (空闲中 / 可用)
            PoolingNHttpClientConnectionManager resource = PoolingNHttpClientConnectionManager.class.cast(FieldUtils.readField(this.resource, "connmgr", true));
            PoolingNHttpClientConnectionManager stream = PoolingNHttpClientConnectionManager.class.cast(FieldUtils.readField(this.stream, "connmgr", true));
            PoolingNHttpClientConnectionManager tools = PoolingNHttpClientConnectionManager.class.cast(FieldUtils.readField(this.tools, "connmgr", true));
            content.append("The http Client(resource)=").append(resource.getTotalStats()).append(System.lineSeparator());
            content.append("The http Client(stream)=").append(stream.getTotalStats()).append(System.lineSeparator());
            content.append("The http Client(tools)=").append(tools.getTotalStats()).append(System.lineSeparator());
        }
        PoolingNHttpClientConnectionManager once = PoolingNHttpClientConnectionManager.class.cast(FieldUtils.readField(this.once, "connmgr", true));
        content.append("The http Client(once)=").append(once.getTotalStats()).append(System.lineSeparator());
        if (log.isInfoEnabled()) {
            log.info(content.toString());
        }
        return content.toString();
    }

    public static class CustomKeepAliveStrategy implements ConnectionKeepAliveStrategy {

        // 毫秒
        protected final Integer timeout;

        public CustomKeepAliveStrategy(Integer timeout) {
            this.timeout = timeout;
        }

        @Override
        public long getKeepAliveDuration(HttpResponse response, HttpContext context) {
            // 尝试获取服务器响应头中的 timeout，果服务器没给 timeout，或者服务器给的时间太长，强制限制在Timeout秒内
            long duration = DefaultConnectionKeepAliveStrategy.INSTANCE.getKeepAliveDuration(response, context);
            if (duration <= 0 || duration > this.timeout) {
                return this.timeout;
            }
            return duration;
        }
    }

    public static class CustomRedirectStrategy extends DefaultRedirectStrategy {

        public static final CustomRedirectStrategy INSTANCE = new CustomRedirectStrategy();

        @Override
        public boolean isRedirected(final HttpRequest request, final HttpResponse response, final HttpContext context) throws ProtocolException {
            return response.getStatusLine().getStatusCode() == HttpStatus.SC_TEMPORARY_REDIRECT || super.isRedirected(request, response, context);
        }
    }
}
