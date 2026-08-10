package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.resouce.ResourceService;
import ai.open.right.utils.HostUtils;
import ai.open.right.utils.IPUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestModel;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.boot.context.event.ApplicationReadyEvent;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.event.EventListener;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.util.Assert;

import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.HashMap;
import java.util.Map;

@Slf4j
@Setter
@Getter
public class VertexRequestService extends GoogleRequestService<GoogleRequest> implements ProviderRequestModel, VertexTokenFetcher {

    public static final String NAME = "VertexRequestService";

    public static final Integer MASK = 5;

    protected VertexTokenExchange vertexTokenExchange;

    protected ResourceService resourceService;

    // Vertex Token资源文件位置（URI）
    protected String tokenUri;

    protected String tokenKey;

    protected Integer seconds;

    // Vertex Policy
    protected String policy;

    protected String model;

    protected volatile String token;

    @Override
    protected void request(GoogleRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // Labels强制追加
        super.request(request, llmConfig, llmQuery);
        request.setLabels(this.labels(llmConfig, llmQuery, Map.class.cast(MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + GoogleRequestService.KEY_LABELS, llmConfig.getAdditional().get(GoogleRequestService.KEY_LABELS)))));
    }

    @Override
    protected String buildToken(GoogleRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return ProviderRequestService.KEY_PREFIX + super.buildToken(request, llmConfig, llmQuery);
    }

    protected Map<String, String> labels(LLMConfig llmConfig, LLMQuery llmQuery, Map<String, String> labelsConfig) throws Exception {
        labelsConfig = labelsConfig != null ? labelsConfig : new HashMap<String, String>();
        labelsConfig.put("client", HostUtils.getHostName());
        labelsConfig.put("ip", IPUtils.getIP());
        labelsConfig.put("app", this.appName);
        return labelsConfig;
    }

    @Scheduled(initialDelayString = "${vertex.token.initialDelay:60000}", fixedRateString = "${provider.vertex.token.initialDelay:60000}")
    // Vertex Token Refresh间隔
    public void refresh() throws Exception {
        if (!StringUtils.isEmpty(this.tokenUri)) {
            // 如果有Token文件则使用Token文件（tokenKey 非空时 URL 指向私钥 PEM，密文见 VertexToken）
            try (InputStream created = this.resourceService.url(this.tokenUri).openStream()) {
                if (!StringUtils.isEmpty(this.tokenKey)) {
                    try (InputStream key = this.resourceService.url(this.tokenKey).openStream()) {
                        this.token = VertexToken.token(IOUtils.toByteArray(created), IOUtils.toString(key, StandardCharsets.UTF_8), this.seconds);
                    }
                } else {
                    this.token = VertexToken.token(IOUtils.toByteArray(created), null, this.seconds);
                }
            }
        } else if (this.vertexTokenExchange != null) {
            // 否则使用Token Exchange远程获取
            this.token = this.vertexTokenExchange.exchange();
        }
        Assert.hasText(this.token, "The token can not be empty");
        if (!StringUtils.isEmpty(this.token) && log.isInfoEnabled()) {
            log.info("The refresh token={}", StringUtils.overlay(this.token, "****", VertexRequestService.MASK, VertexRequestService.MASK));
        }
    }

    @EventListener(ApplicationReadyEvent.class)
    public void init() throws Exception {
        this.init(this.policy);
        try {
            this.refresh();
        } catch (Exception e) {
            // 首次启动不阻断
            if (log.isWarnEnabled()) {
                log.warn(e.getMessage(), e);
            }
        }
    }

    @Override
    public GoogleRequest build() throws Exception {
        return new GoogleRequest();
    }

    @Override
    public String token() throws Exception {
        return this.token;
    }

    @Override
    protected String defToken(WorkflowTask workTask) throws Exception {
        return StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), "__token"), this.token);
    }

    @Override
    protected String defModel(WorkflowTask workTask) throws Exception {
        String model = StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), "__model"), this.getModel(workTask));
        Assert.hasText(model, "The model can not be empty");
        return model;
    }

    @Override
    public String getModel(WorkflowTask workTask) throws Exception {
        return this.model;
    }

    @ConditionalOnProperty(name = "vertex.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends GoogleRequestInitConfig {

        @Autowired(required = false)
        protected VertexTokenExchange vertexTokenExchange;

        @Autowired
        protected ResourceService resourceService;

        @Value("${vertex.token.uri:}")
        // Vertex Token资源文件位置（URI）
        protected String tokenUri;

        @Value("${vertex.token.key:}")
        protected String tokenKey;

        @Value("${vertex.seconds:300}")
        protected Integer seconds;

        @Value("${vertex.policy:BLOCK_NONE}")
        // Vertex Policy
        protected String policy;

        // https://cloud.google.com/vertex-ai/generative-ai/docs/learn/models?hl=zh-cn
        @Value("${vertex.model:gemini-3.6-flash}")
        // Vertex模型，同步VertexRouter
        protected String model;

        @Bean(name = VertexRequestService.NAME)
        @ConditionalOnMissingBean(name = VertexRequestService.NAME)
        public VertexRequestService vertexRequestService() throws Exception {
            VertexRequestService vertexRequestService = new VertexRequestService();
            BeanUtils.copyProperties(this, vertexRequestService);
            log.info("VertexRequestService inited. tokenUri={}, tokenKeyConfigured={}, policy={}, timeout={}",
                    vertexRequestService.getTokenUri(),
                    StringUtils.isNotEmpty(vertexRequestService.getTokenKey()),
                    vertexRequestService.getPolicy(),
                    vertexRequestService.getFunCallTimeout());
            return vertexRequestService;
        }
    }
}
