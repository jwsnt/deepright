package ai.deepright.llm.provider.kimi;

import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.kimi.KimiRequestService;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.deepright.llm.RetryUtils;
import ai.deepright.llm.provider.RequestModelSelect;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;

@Slf4j
@Getter
@Setter
public class CustomerKimiRequestService extends KimiRequestService {

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
        OpenAiRequest request = super.config(llmConfig, llmQuery);
        request.setModel(RequestModelSelect.select(llmQuery, RequestModelSelect.RequestModel.builder()
                .multiInput(this.multiInput)
                .thinking(this.thinking)
                .fast(this.fast)
                .base(this.base)
                .build()));
        return request;
    }

    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    @Setter
    @Getter
    public static class CustomerInitConfig extends InitConfig {

        @Value("${kimi.model.multiInput:kimi-k2.6}")
        protected String multiInput;

        @Value("${kimi.model.thinking:kimi-k2.6}")
        protected String thinking;

        @Value("${kimi.model.fast:kimi-k2-turbo-preview}")
        protected String fast;

        @Value("${kimi.model.base:kimi-k2.6}")
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
