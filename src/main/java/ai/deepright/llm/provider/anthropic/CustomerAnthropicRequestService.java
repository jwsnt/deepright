package ai.deepright.llm.provider.anthropic;

import ai.deepright.llm.RetryUtils;
import ai.deepright.llm.provider.RequestContextUtils;
import ai.deepright.llm.provider.RequestModelSelect;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.anthropic.AnthropicRequest;
import ai.open.right.workflow.flow.llm.provider.anthropic.AnthropicRequestService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
@Getter
@Setter
public class CustomerAnthropicRequestService extends AnthropicRequestService {

    public static final String MAX_TOKENS = "__max_tokens";

    protected String thinkingMedium;

    protected String thinkingHigh;

    protected Integer maxTokens;

    protected String multiInput;

    protected String thinking;

    protected String fast;

    protected String base;

    protected Double rate;

    @Override
    protected void storeHistoryQuery(AnthropicRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        RetryUtils.storeQuery(request, llmConfig, this.historyStore, this.buildHistoryQuery(request, llmConfig));
    }

    @Override
    public AnthropicRequest config(LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        RequestContextUtils.thinking(llmQuery, this.thinkingMedium, this.thinkingHigh);
        return super.config(llmConfig, llmQuery);
    }

    @Override
    protected void request(AnthropicRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        super.request(request, llmConfig, llmQuery);
        request.setModel(RequestModelSelect.select(llmQuery, RequestModelSelect.RequestModel.builder()
                .multiInput(this.multiInput)
                .thinking(this.thinking)
                .fast(this.fast)
                .base(this.base)
                .build()));
        // 优先客户端配置，如果没配置则取推算值和Max_tokens的最小值
        request.setMaxTokens(MapUtils.getInteger(llmQuery.getMetadata(), CustomerAnthropicRequestService.MAX_TOKENS, Math.min(this.maxTokens, (int) (RequestContextUtils.limit(llmQuery, request.getModel()) * this.rate))));
    }

    @Configuration
    @Setter
    @Getter
    public static class CustomerInitConfig extends InitConfig {

        @Value("${anthropic.model.thinkingMedium:medium}")
        protected String thinkingMedium;

        @Value("${anthropic.model.thinkingHigh:high}")
        protected String thinkingHigh;

        @Value("${anthropic.model.max_tokens:128000}")
        protected Integer maxTokens;

        @Value("${anthropic.model.multiInput:claude-opus-4-6}")
        protected String multiInput;

        @Value("${anthropic.model.thinking:claude-opus-4-6}")
        protected String thinking;

        @Value("${anthropic.model.fast:claude-haiku-4-5-20251001}")
        protected String fast;

        @Value("${anthropic.model.base:claude-haiku-4-5-20251001}")
        protected String base;

        @Value("${anthropic.model.rate:0.15}")
        protected Double rate;

        @Override
        @Bean(name = AnthropicRequestService.NAME)
        @ConditionalOnMissingBean(name = AnthropicRequestService.NAME)
        public AnthropicRequestService anthropicRequestService() throws Exception {
            CustomerAnthropicRequestService anthropicRequestService = new CustomerAnthropicRequestService();
            BeanUtils.copyProperties(this, anthropicRequestService);
            log.info("CustomerAnthropicRequestService inited, timeout={}", anthropicRequestService.getFunCallTimeout());
            return anthropicRequestService;
        }
    }
}
