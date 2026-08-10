package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.WorkflowException;
import ai.open.right.config.HttpClientConfig;
import ai.open.right.listener.EventListenerService;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.notify.NotifierService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.lang3.StringUtils;
import org.apache.http.client.config.RequestConfig;
import org.apache.http.client.methods.HttpPost;
import org.apache.http.entity.StringEntity;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.apache.http.nio.client.methods.HttpAsyncMethods;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.util.Assert;

import java.net.URI;
import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.concurrent.ExecutorService;

@Slf4j
@Setter
@Getter
abstract public class ProviderRouter<T extends ProviderRequest> {

    public final static String URL_STREAM = "stream";

    public final static String URL_ONCE = "once";

    protected EventListenerService eventListenerService;

    protected HttpClientConfig httpClientConfig;

    protected ExecutorService executorService;

    protected NotifierService notifierService;

    protected CloseableHttpAsyncClient stream;

    protected CloseableHttpAsyncClient once;

    // 队列时间
    protected Integer queueTimeout;

    protected Double maxSizeRate;

    // LLM最大长度
    protected Integer maxSize = 10 * 1024 * 1024;

    protected Double timeoutRate;

    protected Integer capacity;

    // LLM调用超时（默认/最大）
    protected Integer timeout;

    // 兜底资源泄露检查
    protected Integer discard;

    protected Integer buffer;

    protected Integer queue;

    abstract protected ProviderReader<T> reader(T request, LLMConfig llmConfig, LLMCallback llmCallback) throws Exception;

    abstract protected String url(T request, LLMConfig llmConfig, String t) throws Exception;

    abstract protected Object body(T request) throws Exception;

    public void route(T request, LLMConfig llmConfig, LLMCallback llmCallback) throws Exception {
        CloseableHttpAsyncClient httpClient = null;
        String httpURL = null;
        if (request.getStream()) {
            httpURL = this.url(request, llmConfig, ProviderRouter.URL_STREAM);
            httpClient = this.stream;
        } else {
            httpURL = this.url(request, llmConfig, ProviderRouter.URL_ONCE);
            httpClient = this.once;
        }
        Assert.hasText(httpURL, "Url can not be empty");
        // 回写URL
        request.setUrl(httpURL);
        HttpPost httpRequest = this.buildRequest(request, llmConfig, httpURL);
        ProviderReader<T> reader = this.reader(request, llmConfig, llmCallback);
        try {
            // 开启监听（consuming）并提交请求
            httpClient.execute(HttpAsyncMethods.create(httpRequest), reader, reader.consuming(this.executorService));
            if (log.isInfoEnabled()) {
                log.info("The request biz={}, workflow={}, model={}, deepness={}, content_length={}k, socket_timeout={}, connection_timeout={}, connection_request_timeout={}, stream={}, url={}", request.getMessage().getBiz(), request.getMessage().getWorkflow(), request.getModel(), request.getMessage().getDeepness(), httpRequest.getEntity().getContentLength() / 1024, httpRequest.getConfig().getSocketTimeout(), httpRequest.getConfig().getConnectTimeout(), httpRequest.getConfig().getConnectionRequestTimeout(), request.getStream(), httpRequest.getURI());
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
            httpRequest.abort();
            reader.released();
            throw e;
        }
    }

    protected void reConfig(T request, LLMConfig llmConfig, HttpPost httpPost) throws Exception {
        Integer timeout = this.buildTimeout(request, llmConfig);
        // 如果没有Timeout则使用默认值，最终超时不大于this.timeout * this.timeoutRate
        timeout = Math.min(timeout != null ? timeout : (request.getStream() ? this.httpClientConfig.getSocket4stream() : this.httpClientConfig.getSocket4once()), (int) (this.timeout * this.timeoutRate));
        // 如果没有特别配置Request则与请求调整一致
        request.setFunCallTimeout(request.getFunCallTimeout() != null ? request.getFunCallTimeout() : timeout);
        request.setTimeout(timeout);
        if (log.isDebugEnabled()) {
            log.debug("The request reconfig timeout={}", timeout);
        }
        /// /////////////////////////////////////////////////////////////
        /// ConnectionRequestTimeout: 向连接池申请可用连接
        /// ConnectTimeout: 建立TCP连接
        /// SocketTimeout: 传输/读取数据
        /// /////////////////////////////////////////////////////////////
        httpPost.setConfig(RequestConfig.custom()
                // rconfig_builder.setConnectionRequestTimeout(this.request4stream);
                // rconfig_builder.setConnectTimeout(this.connect4stream);
                // rconfig_builder.setSocketTimeout(this.socket4stream);
                .setConnectionRequestTimeout(request.getStream() ? this.httpClientConfig.getRequest4stream() : this.httpClientConfig.getRequest4once())
                .setConnectTimeout(request.getStream() ? this.httpClientConfig.getConnect4stream() : this.httpClientConfig.getConnect4once())
                .setSocketTimeout(timeout)
                .build());
    }

    protected HttpPost buildRequest(T request, LLMConfig llmConfig, String url) throws Exception {
        HttpPost httpPost = new HttpPost(this.checkUrl(url));
        this.reConfig(request, llmConfig, this.buildHeaders(llmConfig, httpPost));
        StringEntity httpEntity = this.buildEntity(request, llmConfig);
        // 如果存在MediaContext则跳过
        int limitSize = (int) (this.maxSize * this.maxSizeRate);
        Assert.isTrue(!CollectionUtils.isEmpty(request.getMediaContext()) || httpEntity.getContentLength() <= limitSize, "Request body length must be less than: " + httpEntity.getContentLength() + "/" + limitSize);
        httpPost.setEntity(httpEntity);
        return httpPost;
    }

    protected HttpPost buildHeaders(LLMConfig llmConfig, HttpPost httpPost) throws Exception {
        httpPost.addHeader("Content-Type", "application/json");
        if (llmConfig.hasHeaders()) {
            for (String key : llmConfig.getHeaders().keySet()) {
                httpPost.addHeader(key, llmConfig.getHeaders().get(key));
            }
        }
        if (log.isDebugEnabled()) {
            log.debug("The request's headers={}", Arrays.toString(httpPost.getAllHeaders()));
        }
        return httpPost;
    }

    protected StringEntity buildEntity(T request, LLMConfig llmConfig) throws Exception {
        String entity = JsonUtils.write(this.body(request));
        if (log.isDebugEnabled()) {
            log.debug("The request's entity={}", entity);
        }
        request.appendRequest(entity);
        return new StringEntity(entity, StandardCharsets.UTF_8);
    }

    protected Integer buildTimeout(T request, LLMConfig llmConfig) throws Exception {
        // Request.Timeout -> UpStream Timeout -> FunCall Timeout -> DefTimeout
        Integer timeout = request.getTimeout();
        if (request.getTimeout() == null) {
            if (!StringUtils.isEmpty(request.getMessage().getUpstream())) {
                timeout = request.getUpstreamTimeout();
            } else if (request.getMessage().isFromFunCall()) {
                timeout = request.getFunCallTimeout();
            }
        }
        return timeout;
    }

    protected String checkUrl(String url) throws Exception {
        Assert.hasText(url, "Url can not be empty");
        URI uri = new URI(url);
        Assert.hasText(uri.getHost(), "Url host can not be empty: " + url);
        Assert.isTrue(StringUtils.containsAny(uri.getScheme(), "http", "https"), "Url scheme must be http or https: " + url);
        return url;
    }

    @Setter
    @Getter
    public static class ProviderRouterInitConfig {

        @Autowired(required = false)
        protected EventListenerService eventListenerService;

        @Autowired
        protected HttpClientConfig httpClientConfig;

        @Autowired
        protected NotifierService notifierService;

        @Autowired
        @Qualifier("stream")
        protected CloseableHttpAsyncClient stream;

        @Autowired
        @Qualifier("once")
        protected CloseableHttpAsyncClient once;

        @Autowired
        @Qualifier("executor")
        protected ExecutorService executorService;

        @Value("${request.timeout.queue:1000}")
        protected Integer queueTimeout;

        @Value("${request.maxRate:1.5}")
        protected Double maxSizeRate = 1.5;

        @Value("${request.maxSize:10485760}")
        // LLM最大长度
        protected Integer maxSize = 10 * 1024 * 1024;

        @Value("${request.timeout.rate:2}")
        protected Double timeoutRate;

        // Chunk Buffer(1M)
        @Value("${request.capacity:1048576}")
        protected Integer capacity;

        @Value("${request.timeout:900000}")
        protected Integer timeout;

        // 兜底最终泄露检查（10分钟）
        @Value("${request.discard:600000}")
        protected Integer discard;

        // Chunk Buffer(32K)
        @Value("${request.buffer:8992}")
        protected Integer buffer;

        @Value("${request.queue:1000}")
        protected Integer queue;
    }
}
