package ai.deepright.llm.provider.qwen;

import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.qwen.QwenRequestService;
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
public class CustomerQwenRequestService extends QwenRequestService {

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

        @Value("${qwen.model.multiInput:qwen3.6-plus}")
        protected String multiInput;

        @Value("${qwen.model.thinking:qwen3.6-plus}")
        protected String thinking;

        @Value("${qwen.model.fast:qwen3.5-flash}")
        protected String fast;

        @Value("${qwen.model.base:qwen3.5-flash}")
        protected String base;

        @Override
        @Bean(name = QwenRequestService.NAME)
        @ConditionalOnMissingBean(name = QwenRequestService.NAME)
        public QwenRequestService qwenRequestService() throws Exception {
            CustomerQwenRequestService qwenRequestService = new CustomerQwenRequestService();
            BeanUtils.copyProperties(this, qwenRequestService);
            log.info("CustomerQwenModelRequestService inited, timeout={}", qwenRequestService.getFunCallTimeout());
            return qwenRequestService;
        }
    }
}
