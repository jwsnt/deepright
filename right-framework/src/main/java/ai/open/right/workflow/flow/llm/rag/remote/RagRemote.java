package ai.open.right.workflow.flow.llm.rag.remote;

import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAsync;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.apache.http.HttpResponse;
import org.apache.http.client.methods.HttpGet;
import org.apache.http.client.methods.HttpPost;
import org.apache.http.client.methods.HttpRequestBase;
import org.apache.http.entity.StringEntity;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.io.BufferedInputStream;
import java.io.InputStream;
import java.net.URI;
import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.concurrent.Callable;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;

@Setter
@Getter
@Slf4j
// 使用远程服务增强内容
public class RagRemote extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_remote";

    public static final String METHOD_POST = "post";

    public static final String METHOD_GET = "get";

    protected CloseableHttpAsyncClient ragClient;

    protected ExecutorService executorService;

    // Rag Remote调用远程服务超时
    protected Integer timeout4Service;

    // Rag Remote调用下游思考链（Workflow）超时
    protected Integer timeout4Llm;

    // Rag Remote整体超时
    protected Integer timeout;

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        if (log.isDebugEnabled()) {
            log.debug("Rag remote start");
        }
        return new RagAsync(ragConfig, this.executorService.submit(new RemoteFuture(ragConfig, ragData)), ragConfig.getTimeout(this.timeout));
    }

    // 构建远程请求的Headers
    protected HttpRequestBase buildHeaders(HttpRequestBase httpRequestBase, RagConfig ragConfig, RagData ragData) throws Exception {
        httpRequestBase.addHeader("Content-Type", "application/json");
        if (ragConfig.hasRagRemote() && ragConfig.getRagRemoteConfig().hasHeaders()) {
            // 追加Headers
            for (RagRemoteHeader header : ragConfig.getRagRemoteConfig().getHeaders()) {
                Assert.hasText(header.getKey(), "Header key can not be empty");
                String val = null;
                if (header.hasDynamic()) {
                    try {
                        SyncConfig syncConfig = SyncConfig.builder()
                                .timeout(ragConfig.getTimeout4Llm(RagRemote.this.timeout4Llm))
                                .reQuery(ragData.getQuery().getQuery())
                                .workflow(header.getDynamic())
                                .workTask(ragData.getQuery())
                                .build();
                        val = SyncWorkflowTask.exeWorkflow(RagRemote.this.getNotifierService(), syncConfig).get();
                        Assert.hasText(val, "Header value can not be empty");
                        if (log.isDebugEnabled()) {
                            log.debug("Using dynamic header {}-{}", header.getKey(), val);
                        }
                    } catch (Exception e) {
                        if (log.isDebugEnabled()) {
                            log.debug(e.getMessage(), e);
                        }
                        if (!header.getStopOnFailed() && !StringUtils.isEmpty(header.getVal())) {
                            Assert.hasText(header.getVal(), "Header value can not be empty");
                            val = header.getVal();
                            if (log.isDebugEnabled()) {
                                log.debug("Using static header after exception {}-{}", header.getKey(), val);
                            }
                        } else {
                            throw e;
                        }
                    }
                } else {
                    Assert.hasText(header.getVal(), "Header value can not be empty");
                    val = header.getVal();
                    if (log.isDebugEnabled()) {
                        log.debug("Using static header {}-{}", header.getKey(), val);
                    }
                }
                httpRequestBase.setHeader(header.getKey(), val.trim());
            }
        }
        if (log.isDebugEnabled()) {
            log.debug("The headers for rag remote={}", Arrays.toString(httpRequestBase.getAllHeaders()));
        }
        return httpRequestBase;
    }

    // 构建远程请求
    protected HttpRequestBase buildRequest(RagConfig ragConfig, RagData ragData, String body) throws Exception {
        HttpRequestBase request = null;
        Assert.hasText(ragConfig.getService(), "Service can not be empty");
        if (RagRemote.METHOD_GET.equalsIgnoreCase(ragConfig.getMethod())) {
            String url = StringUtils.trim(ragConfig.getService() + (!StringUtils.isEmpty(body) ? ("?" + URI.create(body)) : ""));
            request = new HttpGet(url);
        } else {
            HttpPost httpPost = new HttpPost(ragConfig.getService());
            httpPost.setEntity(new StringEntity(body, StandardCharsets.UTF_8));
            request = httpPost;
        }
        if (log.isInfoEnabled()) {
            log.info("The url for rag remote: method={},url={}", request.getMethod(), request.getURI());
        }
        return this.buildHeaders(request, ragConfig, ragData);
    }

    // 获取远程服务响应
    protected HttpResponse getResponse(RagConfig ragConfig, HttpRequestBase httpRequest) throws Exception {
        Future<HttpResponse> futureResponse = this.ragClient.execute(httpRequest, null);
        return futureResponse.get(ragConfig.getTimeout4Service(this.timeout4Service), TimeUnit.MILLISECONDS);
    }

    public class RemoteFuture implements Callable<Void> {

        protected final RagConfig ragConfig;

        protected final RagData ragData;

        public RemoteFuture(RagConfig ragConfig, RagData ragData) {
            this.ragConfig = ragConfig;
            this.ragData = ragData;
        }

        @Override
        public Void call() throws Exception {
            // 检查Http状态
            HttpResponse httpResponse = RagRemote.this.getResponse(this.ragConfig, RagRemote.this.buildRequest(this.ragConfig, this.ragData, this.getRequest()));
            Assert.isTrue(ProtocolCode.range2xx(httpResponse.getStatusLine().getStatusCode()), "The rag remote's http status code," + httpResponse.getStatusLine().getStatusCode());
            try (InputStream input = new BufferedInputStream(httpResponse.getEntity().getContent())) {
                RagService.updatePrompt(this.ragConfig, this.ragData, this.ragConfig.getReplace(), this.getResponse(input));
            }
            return null;
        }

        protected String getResponse(InputStream input) throws Exception {
            String originalResponse = IOUtils.toString(input, StandardCharsets.UTF_8);
            if (log.isDebugEnabled()) {
                log.debug("The rag remote's original response={}", originalResponse);
            }
            if (this.ragConfig.hasRagOrchestrator() && this.ragConfig.getRagOrchestrator().hasAfter()) {
                // 如果需要清洗响应
                SyncConfig syncConfig = SyncConfig.builder()
                        .timeout(this.ragConfig.getTimeout4Llm(RagRemote.this.timeout4Llm))
                        .workflow(this.ragConfig.getRagOrchestrator().getAfter())
                        .workTask(this.ragData.getQuery())
                        .reQuery(originalResponse)
                        .build();
                originalResponse = SyncWorkflowTask.exeWorkflow(RagRemote.this.getNotifierService(), syncConfig).get();
                if (log.isDebugEnabled()) {
                    log.debug("The rag remote's response after dynamic cleaning, response={}", originalResponse);
                }
            }
            Assert.hasText(originalResponse, "Response can not be empty");
            if (log.isInfoEnabled()) {
                log.info("The rag remote's response={}", originalResponse);
            }
            return originalResponse;
        }

        protected String getRequest() throws Exception {
            if (this.ragConfig.hasRagOrchestrator() && this.ragConfig.getRagOrchestrator().hasBefore()) {
                // 如果需要清洗请求
                SyncConfig syncConfig = SyncConfig.builder()
                        .timeout(this.ragConfig.getTimeout4Llm(RagRemote.this.timeout4Llm))
                        .workflow(this.ragConfig.getRagOrchestrator().getBefore())
                        .workTask(this.ragData.getQuery())
                        .build();
                String request = SyncWorkflowTask.exeWorkflow(RagRemote.this.getNotifierService(), syncConfig).get();
                if (log.isInfoEnabled()) {
                    log.info("The rag remote's request after dynamic cleaning, request={}", request);
                }
                return request;
            } else if (RagRemote.METHOD_POST.equalsIgnoreCase(this.ragConfig.getMethod())) {
                // 不需要清理且为Post则自动转为Json（用于Post的标准请求）
                String request = JsonUtils.write(new RagRequest(this.ragData.getQuery()));
                if (log.isInfoEnabled()) {
                    log.info("The rag remote's request after formatting to json, request={}", request);
                }
                return request;
            }
            if (log.isWarnEnabled()) {
                log.warn("The rag remote's request will be empty");
            }
            return "";
        }
    }

    @ConditionalOnProperty(name = "remote.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Autowired
        @Qualifier("other")
        protected CloseableHttpAsyncClient ragClient;

        @Autowired
        @Qualifier("executor")
        protected ExecutorService executorService;

        @Value("${remote.timeout.service:1800000}")
        // Rag Remote调用远程服务超时
        protected Integer timeout4Service;

        @Value("${remote.timeout.llm:1800000}")
        // Rag Remote调用下游思考链（Workflow）超时
        protected Integer timeout4Llm;

        @Value("${remote.timeout:1800000}")
        // Rag Remote整体超时
        protected Integer timeout;

        @Bean(RagRemote.RAG_KEY)
        @ConditionalOnMissingBean(name = RagRemote.RAG_KEY)
        public RagRemote ragRemote() throws Exception {
            RagRemote ragRemote = new RagRemote();
            BeanUtils.copyProperties(this, ragRemote);
            log.info("RagRemote inited, timeout={},timeout4Llm={},timeout4Service={},timeout4Condition={}", ragRemote.getTimeout(), ragRemote.getTimeout4Llm(), ragRemote.getTimeout4Service(), ragRemote.getTimeout4Condition());
            return ragRemote;
        }
    }
}
