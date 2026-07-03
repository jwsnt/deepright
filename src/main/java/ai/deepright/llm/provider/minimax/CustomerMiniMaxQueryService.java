package ai.deepright.llm.provider.minimax;

import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.anthropic.AnthropicRequest;
import ai.open.right.workflow.flow.llm.provider.anthropic.AnthropicStream;
import ai.open.right.workflow.flow.llm.provider.minimax.MiniMaxQueryService;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.deepright.llm.provider.anthropic.CustomerAnthropicStream;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;

@Slf4j
public class CustomerMiniMaxQueryService extends MiniMaxQueryService {

    @Override
    public AnthropicStream stream(SignalStream signalStream, AnthropicRequest request) throws Exception {
        return new CustomerAnthropicStream(ProviderStreamConfig.<AnthropicRequest>builder()
                .providerStorePolicy(this.providerStorePolicy)
                .trackFunCallService(this.trackFunCallService)
                .mediaInlineService(this.mediaInlineService)
                .notifierService(this.notifierService)
                .providerReason(this.providerReason)
                .tokenStatistic(this.tokenStatistic)
                .historyStore(this.historyStore)
                .namesService(this.namesService)
                .signalStream(signalStream)
                .request(request)
                .build().check());
    }

    @ConditionalOnProperty(name = "minimax.enable", havingValue = "true", matchIfMissing = false)
    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    @Setter
    @Getter
    public static class CustomerInitConfig extends InitConfig {

        @Override
        @Bean(name = LLMQueryService.LLM_MINIMAX)
        @ConditionalOnMissingBean(name = LLMQueryService.LLM_MINIMAX)
        public CustomerMiniMaxQueryService miniMaxQueryService() throws Exception {
            CustomerMiniMaxQueryService miniMaxQueryService = new CustomerMiniMaxQueryService();
            BeanUtils.copyProperties(this, miniMaxQueryService);
            log.info("CustomerMiniMaxQueryService inited");
            return miniMaxQueryService;
        }
    }
}
