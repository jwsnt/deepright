package ai.open.right.workflow.flow.tools.remote;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.rag.remote.RagRemote;
import ai.open.right.workflow.flow.tools.ToolsConfig;
import ai.open.right.workflow.flow.tools.ToolsHeader;
import ai.open.right.workflow.flow.tools.ToolsRequest;
import ai.open.right.workflow.flow.tools.ToolsService;
import ai.open.right.workflow.notify.NotifierService;
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
import java.util.Map;
import java.util.concurrent.TimeUnit;

@Slf4j
@Setter
@Getter
public class ToolsServiceImpl implements ToolsService {

    protected CloseableHttpAsyncClient toolsClient;

    protected NotifierService notifierService;

    // Tools调用下游服务超时
    protected Integer timeout4Service;

    // Tools调用下游思考链（Workflow）超时
    protected Integer timeout4Llm;

    @Override
    public String execute(ToolsConfig toolsConfig, WorkflowTask workTask) throws Exception {
        String request = this.getToolsRequest(toolsConfig, workTask);
        if (log.isDebugEnabled()) {
            log.debug("Tools request={}", request);
        }
        HttpRequestBase httpRequest = this.buildRequest(toolsConfig, workTask, request);
        HttpResponse httpResponse = this.toolsClient.execute(httpRequest, null).get(toolsConfig.getTimeout4Service(this.timeout4Service), TimeUnit.MILLISECONDS);
        try (InputStream input = new BufferedInputStream(httpResponse.getEntity().getContent())) {
            if (!ProtocolCode.range2xx(httpResponse.getStatusLine().getStatusCode())) {
                String response = IOUtils.toString(input, StandardCharsets.UTF_8);
                if (log.isInfoEnabled()) {
                    log.info("Tools exception={}", response);
                }
                // Not check, McpError content may be empty
                throw new WorkflowException(new StringBuffer(request).append(System.lineSeparator()).append(response).toString(), httpResponse.getStatusLine().getStatusCode());
            }
            return this.getToolsResponse(toolsConfig, workTask, input);
        } catch (Exception e) {
            // 40x 30x
            throw new WorkflowException("Tools response code: " + httpResponse.getStatusLine().getStatusCode(), httpResponse.getStatusLine().getStatusCode());
        }
    }

    protected HttpRequestBase buildHeaders(HttpRequestBase httpRequestBase, ToolsConfig toolsConfig, WorkflowTask workTask) throws Exception {
        httpRequestBase.addHeader("Content-Type", "application/json");
        if (toolsConfig.hasHeaders()) {
            for (ToolsHeader header : toolsConfig.getHeaders()) {
                Assert.hasText(header.getKey(), "Header key can not be empty");
                String val = null;
                if (header.hasDynamic()) {
                    try {
                        SyncConfig syncConfig = SyncConfig.builder()
                                .timeout(toolsConfig.getTimeout4Llm(this.timeout4Llm))
                                .workflow(header.getDynamic())
                                .reQuery(workTask.getQuery())
                                .workTask(workTask)
                                .build();
                        val = SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get();
                        if (log.isDebugEnabled()) {
                            log.info("Using dynamic header: key={},value={}", header.getKey(), val);
                        }
                    } catch (Exception e) {
                        if (!header.getStopOnFailed() && !StringUtils.isEmpty(header.getVal())) {
                            val = header.getVal();
                            if (log.isDebugEnabled()) {
                                log.info("Using static header after exception: key={},value={}", header.getKey(), val);
                            }
                        } else {
                            throw e;
                        }
                    }
                } else {
                    Assert.hasText(header.getVal(), "Header val can not be empty");
                    val = header.getVal();
                    if (log.isDebugEnabled()) {
                        log.info("Using static header: key={},value={}", header.getKey(), val);
                    }
                }
                httpRequestBase.setHeader(header.getKey(), val.trim());
            }
        }
        if (log.isDebugEnabled()) {
            log.debug("The headers for tools={}", Arrays.toString(httpRequestBase.getAllHeaders()));
        }
        return httpRequestBase;
    }

    protected HttpRequestBase buildRequest(ToolsConfig toolsConfig, WorkflowTask workTask, String body) throws Exception {
        Assert.hasText(toolsConfig.getService(), "Tools service can not be empty");
        HttpRequestBase request = null;
        String service = toolsConfig.getService();
        if (RagRemote.METHOD_GET.equalsIgnoreCase(toolsConfig.getMethod())) {
            request = new HttpGet(StringUtils.defaultIfEmpty(service.contains("?") ? (service + "&" + URI.create(body).toString()) : (service + "?" + URI.create(body).toString()), ""));
        } else {
            HttpPost httpPost = new HttpPost(service);
            httpPost.setEntity(new StringEntity(JsonUtils.clean(body), StandardCharsets.UTF_8));
            request = httpPost;
        }
        if (log.isInfoEnabled()) {
            log.info("The url for rag remote: method={},uri={}", request.getMethod(), request.getURI());
        }
        return this.buildHeaders(request, toolsConfig, workTask);
    }

    protected String getToolsResponse(ToolsConfig toolsConfig, WorkflowTask workTask, InputStream input) throws Exception {
        String responseBody = IOUtils.toString(input, StandardCharsets.UTF_8);
        if (log.isDebugEnabled()) {
            log.debug("Tools response={}", responseBody);
        }
        if (toolsConfig.hasOrchestrator() && toolsConfig.getToolsOrchestrator().hasAfter()) {
            // 如果需要清洗
            SyncConfig syncConfig = SyncConfig.builder()
                    .workflow(toolsConfig.getToolsOrchestrator().getAfter())
                    .timeout(toolsConfig.getTimeout4Llm(this.timeout4Llm))
                    .reQuery(responseBody)
                    .workTask(workTask)
                    .build();
            responseBody = JsonUtils.clean(SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get());
            if (log.isInfoEnabled()) {
                log.info("Tools response after dynamic cleaning={}", responseBody);
            }
        }
        Assert.hasText(responseBody, "Tools response can not be empty");
        return responseBody;
    }

    protected String getToolsRequest(ToolsConfig toolsConfig, WorkflowTask workTask) throws Exception {
        if (toolsConfig.hasOrchestrator() && toolsConfig.getToolsOrchestrator().hasParam()) {
            // For Get
            String request = toolsConfig.getToolsOrchestrator().getParam() + workTask.getQuery();
            if (log.isDebugEnabled()) {
                log.debug("Tools request after append params={}", request);
            }
            return request;
        }
        if (toolsConfig.hasOrchestrator() && toolsConfig.getToolsOrchestrator().hasBefore()) {
            SyncConfig syncConfig = SyncConfig.builder()
                    .workflow(toolsConfig.getToolsOrchestrator().getBefore())
                    .timeout(toolsConfig.getTimeout4Llm(this.timeout4Llm))
                    .reQuery(workTask.getQuery())
                    .workTask(workTask)
                    .build();
            String response = SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get();
            if (log.isInfoEnabled()) {
                log.info("Tools request after dynamic cleaning={}", response);
            }
            Assert.hasText(response, "Tools response can not be empty");
            return this.wrapRequest(toolsConfig, workTask, JsonUtils.clean(response));
        } else {
            return this.wrapRequest(toolsConfig, workTask, workTask.getQuery());
        }
    }

    public String wrapRequest(ToolsConfig toolsConfig, WorkflowTask workTask, String response) throws Exception {
        if (toolsConfig.shouldWrap()) {
            Assert.isTrue(toolsConfig.isValidWrap(), "Wrap value is invalid");
            switch (toolsConfig.getWrap()) {
                case ToolsConfig.WRAP_STRING:
                    String wrapString = JsonUtils.write(new ToolsRequest(workTask, StringUtils.defaultIfEmpty(response, "")));
                    if (log.isInfoEnabled()) {
                        log.info("Tools wrap request with string={}", wrapString);
                    }
                    return wrapString;
                case ToolsConfig.WRAP_OBJECT:
                    String wrapObject = JsonUtils.write(new ToolsRequest(workTask, JsonUtils.read(response, Map.class)));
                    if (log.isInfoEnabled()) {
                        log.info("Tools wrap request with object={}", wrapObject);
                    }
                    return wrapObject;
                case ToolsConfig.WRAP_SOURCE:
                    if (log.isInfoEnabled()) {
                        log.info("Tools wrap request with source={}", response);
                    }
                    return response;
            }
        }
        if (log.isWarnEnabled()) {
            log.warn("Tools request will be empty");
        }
        return "";
    }

    @ConditionalOnProperty(name = "tools.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        @Qualifier("other")
        protected CloseableHttpAsyncClient toolsClient;

        @Autowired
        protected NotifierService notifierService;

        @Value("${tools.timeout.service:1800000}")
        // Tools调用下游服务超时
        protected Integer timeout4Service;

        @Value("${tools.timeout.llm:1800000}")
        // Tools调用下游思考链（Workflow）超时
        protected Integer timeout4Llm;

        @Bean
        @ConditionalOnMissingBean(value = ToolsService.class)
        public ToolsService toolsService() throws Exception {
            ToolsServiceImpl toolsService = new ToolsServiceImpl();
            BeanUtils.copyProperties(this, toolsService);
            log.info("ToolsServiceImpl inited: timeout4Service={},timeout4Llm={}", toolsService.getTimeout4Service(), toolsService.getTimeout4Llm());
            return toolsService;
        }
    }
}
