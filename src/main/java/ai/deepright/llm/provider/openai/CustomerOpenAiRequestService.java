package ai.deepright.llm.provider.openai;

import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequestService;
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

@Slf4j
@Getter
@Setter
public class CustomerOpenAiRequestService extends OpenAiRequestService {

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

    @Configuration
    @Setter
    @Getter
    public static class CustomerInitConfig extends InitConfig {

        @Value("${openai.model.multiInput:gpt-5.4}")
        protected String multiInput;

        @Value("${openai.model.thinking:gpt-5.4}")
        protected String thinking;

        @Value("${openai.model.fast:gpt-5.3-codex}")
        protected String fast;

        @Value("${openai.model.base:gpt-5.4}")
        protected String base;

        @Override
        @Bean(name = OpenAiRequestService.NAME)
        @ConditionalOnMissingBean(name = OpenAiRequestService.NAME)
        public OpenAiRequestService openAiRequestService() throws Exception {
            CustomerOpenAiRequestService openAiRequestService = new CustomerOpenAiRequestService();
            BeanUtils.copyProperties(this, openAiRequestService);
            log.info("CustomerOpenAiRequestService inited, timeout={}", openAiRequestService.getFunCallTimeout());
            return openAiRequestService;
        }
    }
}
