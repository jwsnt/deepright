package ai.deepright.llm.provider.bigmodel;

import ai.deepright.llm.RetryUtils;
import ai.deepright.llm.provider.RequestContextUtils;
import ai.deepright.llm.provider.RequestModelSelect;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.bigmodel.BigModelRequestService;
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
public class CustomerBigModelRequestService extends BigModelRequestService {

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

    @ConditionalOnProperty(name = "bigmodel.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class CustomerInitConfig extends InitConfig {

        @Value("${bigmodel.model.thinkingMedium:medium}")
        protected String thinkingMedium;

        @Value("${bigmodel.model.thinkingHigh:high}")
        protected String thinkingHigh;

        @Value("${bigmodel.model.multiInput:glm-5v-turbo}")
        protected String multiInput;

        @Value("${bigmodel.model.thinking:glm-5.1}")
        protected String thinking;

        @Value("${bigmodel.model.fast:glm-4.7-flashx}")
        protected String fast;

        @Value("${bigmodel.model.base:glm-4.7-flashx}")
        protected String base;

        @Override
        @Bean(name = BigModelRequestService.NAME)
        @ConditionalOnMissingBean(name = BigModelRequestService.NAME)
        public BigModelRequestService bigModelRequestService() throws Exception {
            CustomerBigModelRequestService bigModelRequestService = new CustomerBigModelRequestService();
            BeanUtils.copyProperties(this, bigModelRequestService);
            log.info("CustomerBigModelRequestService inited, timeout={}", bigModelRequestService.getFunCallTimeout());
            return bigModelRequestService;
        }
    }
}
