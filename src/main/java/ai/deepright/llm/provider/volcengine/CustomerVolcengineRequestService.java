package ai.deepright.llm.provider.volcengine;

import ai.deepright.llm.RetryUtils;
import ai.deepright.llm.provider.RequestContextUtils;
import ai.deepright.llm.provider.RequestModelSelect;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.volcengine.VolcengineRequestService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
public class CustomerVolcengineRequestService extends VolcengineRequestService {

    protected String thinkingMedium;

    protected String thinkingHigh;

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

        @Value("${volcengine.model.thinkingMedium:medium}")
        protected String thinkingMedium;

        @Value("${volcengine.model.thinkingHigh:high}")
        protected String thinkingHigh;

        @Value("${volcengine.model.thinking:doubao-seed-2-0-pro-260215}")
        protected String thinking;

        @Value("${volcengine.model.fast:doubao-seed-2-0-lite-260215}")
        protected String fast;

        @Value("${volcengine.model.base:doubao-seed-2-0-lite-260215}")
        protected String base;

        @Override
        @Bean(name = VolcengineRequestService.NAME)
        @ConditionalOnMissingBean(name = VolcengineRequestService.NAME)
        public VolcengineRequestService volcengineRequestService() throws Exception {
            CustomerVolcengineRequestService volcengineRequestService = new CustomerVolcengineRequestService();
            BeanUtils.copyProperties(this, volcengineRequestService);
            log.info("CustomerVolcengineModelRequestService inited, timeout={}", volcengineRequestService.getFunCallTimeout());
            return volcengineRequestService;
        }
    }
}
