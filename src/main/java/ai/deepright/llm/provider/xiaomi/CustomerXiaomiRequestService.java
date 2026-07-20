package ai.deepright.llm.provider.xiaomi;

import ai.deepright.llm.RetryUtils;
import ai.deepright.llm.provider.RequestContextUtils;
import ai.deepright.llm.provider.RequestModelSelect;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.xiaomi.XiaomiRequestService;
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
public class CustomerXiaomiRequestService extends XiaomiRequestService {

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

        @Value("${xiaomi.model.thinkingMedium:medium}")
        protected String thinkingMedium;

        @Value("${xiaomi.model.thinkingHigh:high}")
        protected String thinkingHigh;

        @Value("${xiaomi.model.thinking:mimo-v2.5}")
        protected String multiInput;

        @Value("${xiaomi.model.thinking:mimo-v2.5-pro}")
        protected String thinking;

        @Value("${xiaomi.model.fast:mimo-v2-flash}")
        protected String fast;

        @Value("${xiaomi.model.base:mimo-v2-flash}")
        protected String base;

        @Override
        @Bean(name = XiaomiRequestService.NAME)
        @ConditionalOnMissingBean(name = XiaomiRequestService.NAME)
        public XiaomiRequestService xiaomiRequestService() throws Exception {
            CustomerXiaomiRequestService xiaomiRequestService = new CustomerXiaomiRequestService();
            BeanUtils.copyProperties(this, xiaomiRequestService);
            log.info("CustomerXiaomiRequestService inited, timeout={}", xiaomiRequestService.getFunCallTimeout());
            return xiaomiRequestService;
        }
    }
}
