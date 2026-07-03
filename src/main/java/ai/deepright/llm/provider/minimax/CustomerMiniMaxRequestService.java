package ai.deepright.llm.provider.minimax;

import ai.deepright.llm.RetryUtils;
import ai.deepright.llm.provider.RequestContextUtils;
import ai.deepright.llm.provider.RequestModelSelect;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.anthropic.AnthropicRequest;
import ai.open.right.workflow.flow.llm.provider.minimax.MiniMaxRequestService;
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
public class CustomerMiniMaxRequestService extends MiniMaxRequestService {

    protected String thinking;

    protected String fast;

    protected String base;

    protected Double rate;

    @Override
    protected Integer buildRecallOffset(AnthropicRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return (int) (super.buildRecallOffset(request, llmConfig, llmQuery) * this.rate);
    }

    @Override
    protected Integer buildRecallNums(AnthropicRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return (int) (super.buildRecallNums(request, llmConfig, llmQuery) * this.rate);
    }

    @Override
    protected void storeHistoryQuery(AnthropicRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        RetryUtils.storeQuery(request, llmConfig, this.historyStore, this.buildHistoryQuery(request, llmConfig));
    }

    @Override
    public AnthropicRequest config(LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        AnthropicRequest request = super.config(llmConfig, llmQuery);
        request.setModel(RequestModelSelect.select(llmQuery, RequestModelSelect.RequestModel.builder()
                .thinking(this.thinking)
                .fast(this.fast)
                .base(this.base)
                .build()));
        request.setMaxTokens((int) (RequestContextUtils.limit(llmQuery, request.getModel()) * this.rate));
        return request;
    }

    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    @Setter
    @Getter
    public static class CustomerInitConfig extends InitConfig {

        @Value("${minimax.model.thinking:MiniMax-M3}")
        protected String thinking;

        @Value("${minimax.model.fast:MiniMax-M2.7-highspeed}")
        protected String fast;

        @Value("${minimax.model.base:MiniMax-M3}")
        protected String base;

        @Value("${minimax.model.rate:0.65}")
        protected Double rate;

        @Override
        @Bean(name = MiniMaxRequestService.NAME)
        @ConditionalOnMissingBean(name = MiniMaxRequestService.NAME)
        public MiniMaxRequestService miniMaxRequestService() throws Exception {
            CustomerMiniMaxRequestService miniMaxRequestService = new CustomerMiniMaxRequestService();
            BeanUtils.copyProperties(this, miniMaxRequestService);
            log.info("CustomerMiniMaxModelRequestService inited, timeout={}", miniMaxRequestService.getFunCallTimeout());
            return miniMaxRequestService;
        }
    }
}
