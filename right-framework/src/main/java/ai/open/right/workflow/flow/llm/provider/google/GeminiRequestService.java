package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestModel;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

@Setter
@Getter
@Slf4j
public class GeminiRequestService extends GoogleRequestService<GoogleRequest> implements ProviderRequestModel {

    public static final String NAME = "GeminiRequestService";

    // Gemini Policy
    protected String policy;

    // Gemini Token
    protected String token;

    protected String model;

    @PostConstruct
    public void init() throws Exception {
        this.init(this.policy);
    }

    @Override
    protected String buildToken(GoogleRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return super.buildToken(request, llmConfig, llmQuery);
    }

    @Override
    public GoogleRequest build() throws Exception {
        return new GoogleRequest();
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

    @ConditionalOnProperty(name = "gemini.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends GoogleRequestInitConfig {

        @Value("${gemini.policy:BLOCK_NONE}")
        // Gemini Policy
        protected String policy;

        @Value("${gemini.model:gemini-3.6-flash")
        // Gemini模型，同步GeminiRouter
        protected String model = "gemini-3.6-flash";

        @Value("${gemini.token:}")
        // Gemini Token
        protected String token;

        @Bean(name = GeminiRequestService.NAME)
        @ConditionalOnMissingBean(name = GeminiRequestService.NAME)
        public GeminiRequestService geminiRequestService() throws Exception {
            GeminiRequestService geminiRequestService = new GeminiRequestService();
            BeanUtils.copyProperties(this, geminiRequestService);
            log.info("GeminiRequestService inited. policy={}, timeout={}", geminiRequestService.getPolicy(), geminiRequestService.getFunCallTimeout());
            return geminiRequestService;
        }
    }
}
