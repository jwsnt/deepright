package ai.deepright.llm.provider.google;

import ai.deepright.llm.RetryUtils;
import ai.deepright.llm.provider.RequestModelSelect;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.google.GeminiRequestService;
import ai.open.right.workflow.flow.llm.provider.google.GoogleRequest;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
@Getter
@Setter
public class CustomerGeminiRequestService extends GeminiRequestService {

    protected String multiOutput;

    protected String multiInput;

    protected String thinking;

    protected String fast;

    protected String base;

    protected Double rate;

    @Override
    protected Integer buildRecallOffset(GoogleRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return (int)(super.buildRecallOffset(request, llmConfig, llmQuery) * this.rate);
    }

    @Override
    protected Integer buildRecallNums(GoogleRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return (int)(super.buildRecallNums(request, llmConfig, llmQuery) * this.rate);
    }

    @Override
    protected void storeHistoryQuery(GoogleRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        RetryUtils.storeQuery(request, llmConfig, this.historyStore, this.buildHistoryQuery(request, llmConfig));
    }

    @Override
    public GoogleRequest config(LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        GoogleRequest request = super.config(llmConfig, llmQuery);
        request.setModel(RequestModelSelect.select(llmQuery, RequestModelSelect.RequestModel.builder()
                .multiOutput(this.multiOutput)
                .multiInput(this.multiInput)
                .thinking(this.thinking)
                .fast(this.fast)
                .base(this.base)
                .build()));
        return request;
    }

    @Configuration
    @Setter
    @Getter
    public static class CustomerInitConfig extends GeminiRequestService.InitConfig {

        @Value("${gemini.model.multiOutput:gemini-3.1-flash-image-preview}")
        protected String multiOutput;

        @Value("${gemini.model.multiInput:gemini-3.5-flash}")
        protected String multiInput;

        @Value("${gemini.model.thinking:gemini-3.1-pro-preview}")
        protected String thinking;

        @Value("${gemini.model.fast:gemini-3.5-flash}")
        protected String fast;

        @Value("${gemini.model.base:gemini-3.5-flash}")
        protected String base;

        @Value("${gemini.model.rate:0.65}")
        protected Double rate;

        @Override
        @Bean(name = GeminiRequestService.NAME)
        @ConditionalOnMissingBean(name = GeminiRequestService.NAME)
        public GeminiRequestService geminiRequestService() throws Exception {
            CustomerGeminiRequestService geminiRequestService = new CustomerGeminiRequestService();
            BeanUtils.copyProperties(this, geminiRequestService);
            log.info("CustomerGeminiRequestService inited. policy={}, timeout={}", geminiRequestService.getPolicy(), geminiRequestService.getFunCallTimeout());
            return geminiRequestService;
        }
    }
}
