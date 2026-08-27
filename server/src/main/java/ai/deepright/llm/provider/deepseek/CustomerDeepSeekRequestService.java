package ai.deepright.llm.provider.deepseek;

import ai.deepright.llm.RetryUtils;
import ai.deepright.llm.provider.RequestContextUtils;
import ai.deepright.llm.provider.RequestModelSelect;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.deepseek.DeepSeekRequestService;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
@Getter
@Setter
public class CustomerDeepSeekRequestService extends DeepSeekRequestService {

    protected String thinkingMedium;

    protected String thinkingHigh;

    protected String multiInput;

    protected String thinking;

    protected String fast;

    protected String base;

    @Override
    protected void storeHistoryQuery(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        RetryUtils.storeQuery(request, llmConfig, this.historyStore, this.buildHistoryQuery(request, llmConfig));
    }

    @Override
    public OpenAiRequest config(LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        RequestContextUtils.thinking(llmQuery, this.thinkingMedium, this.thinkingHigh);
        return super.config(llmConfig, llmQuery);
    }

    @Override
    protected void request(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        super.request(request, llmConfig, llmQuery);
        request.setModel(RequestModelSelect.select(llmQuery, RequestModelSelect.RequestModel.builder()
                .multiInput(this.multiInput)
                .thinking(this.thinking)
                .fast(this.fast)
                .base(this.base)
                .build()));
    }

    @ConditionalOnProperty(name = "deepseek.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class CustomerInitConfig extends InitConfig {

        @Value("${deepseek.model.thinkingMedium:medium}")
        protected String thinkingMedium;

        @Value("${deepseek.model.thinkingHigh:high}")
        protected String thinkingHigh;

        @Value("${deepseek.model.multiInput:deepseek-v4-flash-vision-exp}")
        protected String multiInput;

        @Value("${deepseek.model.thinking:deepseek-v4-pro}")
        protected String thinking;

        @Value("${deepseek.model.fast:deepseek-v4-flash}")
        protected String fast;

        @Value("${deepseek.model.base:deepseek-v4-flash}")
        protected String base;

        @Override
        @Bean(name = DeepSeekRequestService.NAME)
        @ConditionalOnMissingBean(name = DeepSeekRequestService.NAME)
        public DeepSeekRequestService deepSeekRequestService() throws Exception {
            CustomerDeepSeekRequestService deepSeekRequestService = new CustomerDeepSeekRequestService();
            BeanUtils.copyProperties(this, deepSeekRequestService);
            log.info("CustomerDeepSeekModelRequestService inited, timeout={}", deepSeekRequestService.getFunCallTimeout());
            return deepSeekRequestService;
        }
    }
}
