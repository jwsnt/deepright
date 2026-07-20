package ai.deepright.llm.provider.kimi;

import ai.deepright.llm.RetryUtils;
import ai.deepright.llm.provider.RequestContextUtils;
import ai.deepright.llm.provider.RequestModelSelect;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.kimi.KimiRequestService;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
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
public class CustomerKimiRequestService extends KimiRequestService {

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
        OpenAiRequest request = super.config(llmConfig, llmQuery);
        request.setModel(RequestModelSelect.select(llmQuery, RequestModelSelect.RequestModel.builder()
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
    public static class CustomerInitConfig extends InitConfig {

        @Value("${kimi.model.thinkingMedium:medium}")
        protected String thinkingMedium;

        @Value("${kimi.model.thinkingHigh:high}")
        protected String thinkingHigh;

        @Value("${kimi.model.multiInput:kimi-k3}")
        protected String multiInput;

        @Value("${kimi.model.thinking:kimi-k3}")
        protected String thinking;

        @Value("${kimi.model.fast:kimi-k2.7-code-highspeed}")
        protected String fast;

        @Value("${kimi.model.base:kimi-k3}")
        protected String base;

        @Override
        @Bean(name = KimiRequestService.NAME)
        @ConditionalOnMissingBean(name = KimiRequestService.NAME)
        public KimiRequestService kimiRequestService() throws Exception {
            CustomerKimiRequestService kimiRequestService = new CustomerKimiRequestService();
            BeanUtils.copyProperties(this, kimiRequestService);
            log.info("CustomerKimiModelRequestService inited, timeout={}", kimiRequestService.getFunCallTimeout());
            return kimiRequestService;
        }
    }
}
