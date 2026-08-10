package ai.open.right.workflow.flow.resource.impl;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.resource.ResourceConfig;
import ai.open.right.workflow.flow.resource.ResourceFetcher;
import ai.open.right.workflow.flow.resource.ResourceRequest;
import ai.open.right.workflow.flow.resource.ResourceResponse;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.ArrayUtils;
import org.apache.commons.lang3.StringUtils;
import org.apache.http.Header;
import org.apache.http.HttpResponse;
import org.apache.http.client.config.RequestConfig;
import org.apache.http.client.methods.HttpGet;
import org.apache.http.client.methods.HttpPost;
import org.apache.http.client.methods.HttpRequestBase;
import org.apache.http.entity.StringEntity;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.concurrent.TimeUnit;

@Setter
@Getter
@Slf4j
public class ResourceFetcherImpl implements ResourceFetcher {

    public static final String HEADER_KEY = "__";

    protected CloseableHttpAsyncClient resource;

    // Resource客户端的Request超时
    protected Integer request4resource;

    // Resource客户端的Connect超时
    protected Integer connect4resource;

    // Resource客户端的Socket超时
    protected Integer socket4resource;

    @Override
    public ResourceResponse fetch(ResourceConfig resourceConfig, WorkflowTask workTask) throws Exception {
        ResourceRequest request = this.buildResourceRequest(resourceConfig, workTask);
        Assert.isTrue(request != null && request.isValid(), "The resource request is not valid: " + workTask.getQuery());
        if (log.isDebugEnabled()) {
            log.debug("The resource request: {}", JsonUtils.write(request));
        }
        // 构建请求
        HttpRequestBase httpBase = StringUtils.equalsIgnoreCase(ResourceRequest.METHOD_GET, request.getMethod()) ? this.buildHttpGet(resourceConfig, workTask, request) : this.buildHttpPost(resourceConfig, workTask, request);
        // 构建响应
        HttpResponse response = this.buildHttpResponse(resourceConfig, workTask, request, httpBase);
        return this.buildResourceResponse(resourceConfig, workTask, request, this.checkHttpResponse(resourceConfig, workTask, request, response));
    }

    protected ResourceResponse buildResourceResponse(ResourceConfig resourceConfig, WorkflowTask workTask, ResourceRequest request, HttpResponse response) throws Exception {
        ResourceResponse resourceResponse = new ResourceResponse();
        String content = "";
        if (response.getEntity() != null) {
            try (InputStream input = response.getEntity().getContent()) {
                content = IOUtils.toString(input, StandardCharsets.UTF_8);
            }
        }
        if (log.isDebugEnabled()) {
            log.debug("The resource response content={}", content);
        }
        resourceResponse.setContent(this.buildContent(resourceConfig, workTask, request, content));
        if (ArrayUtils.getLength(response.getAllHeaders()) > 0) {
            if (log.isDebugEnabled()) {
                log.debug("The resource response header={}", Arrays.toString(response.getAllHeaders()));
            }
            for (Header header : response.getAllHeaders()) {
                resourceResponse.addHeader(header.getName(), header.getValue());
            }
        }
        return resourceResponse;
    }

    protected HttpResponse buildHttpResponse(ResourceConfig resourceConfig, WorkflowTask workTask, ResourceRequest request, HttpRequestBase httpBase) throws Exception {
        HttpRequestBase httpRequest = this.buildHttpHeader(resourceConfig, workTask, request, httpBase);
        Integer timeout = this.buildTimeout(resourceConfig, workTask, request);
        timeout = timeout != null ? timeout : this.socket4resource;
        if (log.isDebugEnabled()) {
            log.debug("The resource request timeout={}", timeout);
        }
        httpRequest.setConfig(RequestConfig.custom()
                // rconfig_builder.setConnectionRequestTimeout(this.request4stream);
                // rconfig_builder.setConnectTimeout(this.connect4stream);
                // rconfig_builder.setSocketTimeout(this.socket4stream);
                .setConnectionRequestTimeout(this.request4resource)
                .setConnectTimeout(this.connect4resource)
                .setSocketTimeout(timeout)
                .build());
        return this.resource.execute(httpRequest, null).get(timeout, TimeUnit.MILLISECONDS);
    }

    protected HttpRequestBase buildHttpHeader(ResourceConfig resourceConfig, WorkflowTask workTask, ResourceRequest request, HttpRequestBase httpBase) throws Exception {
        if (resourceConfig != null && resourceConfig.hasHeaders() && resourceConfig.getAutoCopy()) {
            for (String key : resourceConfig.getHeaders().keySet()) {
                if (this.allowedHeader(resourceConfig, workTask, request, key)) {
                    httpBase.setHeader(key, resourceConfig.getHeaders().get(key));
                }
            }
        }
        if (!MapUtils.isEmpty(request.getHeaders())) {
            for (String key : request.getHeaders().keySet()) {
                if (this.allowedHeader(resourceConfig, workTask, request, key)) {
                    httpBase.setHeader(key, request.getHeaders().get(key));
                }
            }
        }
        if (!httpBase.containsHeader("Content-Type")) {
            httpBase.setHeader("Content-Type", "application/json");
        }
        if (log.isDebugEnabled()) {
            log.debug("The resource request header={}", request.getHeaders());
        }
        return httpBase;
    }

    protected HttpResponse checkHttpResponse(ResourceConfig resourceConfig, WorkflowTask workTask, ResourceRequest request, HttpResponse response) throws Exception {
        Integer statusCode = response.getStatusLine().getStatusCode();
        if (!ProtocolCode.range2xx(statusCode)) {
            try (InputStream input = response.getEntity().getContent()) {
                if (log.isDebugEnabled()) {
                    log.debug("The resource response error message={}", IOUtils.toString(input, StandardCharsets.UTF_8));
                }
                throw new WorkflowException(this.buildException(resourceConfig, workTask, request, response), statusCode);
            }
        }
        return response;
    }

    protected String buildException(ResourceConfig resourceConfig, WorkflowTask workTask, ResourceRequest request, HttpResponse response) throws Exception {
        return System.lineSeparator() + "The internal error occurred, status code=" + response.getStatusLine().getStatusCode() + ", if the request is terminated, please try again later." + System.lineSeparator();
    }

    protected String buildContent(ResourceConfig resourceConfig, WorkflowTask workTask, ResourceRequest request, String content) throws Exception {
        return content;
    }

    protected Boolean allowedHeader(ResourceConfig resourceConfig, WorkflowTask workTask, ResourceRequest request, String key) throws Exception {
        return !StringUtils.startsWithIgnoreCase(key, ResourceFetcherImpl.HEADER_KEY) && !StringUtils.endsWithIgnoreCase(key, ResourceFetcherImpl.HEADER_KEY);
    }

    protected HttpPost buildHttpPost(ResourceConfig resourceConfig, WorkflowTask workTask, ResourceRequest request) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("The resource post request url={}, content={}", request.getUrl(), request.getContent());
        }
        HttpPost httpPost = new HttpPost(request.getUrl());
        httpPost.setEntity(new StringEntity(this.buildEntity(resourceConfig, workTask, request)));
        return httpPost;
    }

    protected HttpGet buildHttpGet(ResourceConfig resourceConfig, WorkflowTask workTask, ResourceRequest request) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("The resource get request url={}", request.getUrl());
        }
        return new HttpGet(request.getUrl());
    }

    protected Integer buildTimeout(ResourceConfig resourceConfig, WorkflowTask workTask, ResourceRequest request) throws Exception {
        return resourceConfig != null && resourceConfig.getTimeout() != null ? resourceConfig.getTimeout() : null;
    }

    protected String buildEntity(ResourceConfig resourceConfig, WorkflowTask workTask, ResourceRequest request) throws Exception {
        String entity = StringUtils.defaultIfEmpty(JsonUtils.write(request.getContent()), "");
        if (log.isDebugEnabled()) {
            log.debug("The resource request entity={}", entity);
        }
        return entity;
    }

    protected ResourceRequest buildResourceRequest(ResourceConfig resourceConfig, WorkflowTask workTask) throws Exception {
        return workTask.getObjectQuery(ResourceRequest.class);
    }

    @ConditionalOnProperty(name = "resource.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected CloseableHttpAsyncClient resource;

        @Value("${httpclient.timeout.resource.request:1000}")
        protected Integer request4resource;

        @Value("${httpclient.timeout.resource.connect:2000}")
        // Resource客户端的Connect超时
        protected Integer connect4resource;

        @Value("${httpclient.timeout.resource.socket:10000}")
        // Resource客户端的Socket超时
        protected Integer socket4resource;

        @Bean
        @ConditionalOnMissingBean(value = ResourceFetcher.class)
        public ResourceFetcher resourceFetcher() throws Exception {
            ResourceFetcherImpl resourceFetcher = new ResourceFetcherImpl();
            BeanUtils.copyProperties(this, resourceFetcher);
            log.info("ResourceServiceImpl inited");
            return resourceFetcher;
        }
    }
}
