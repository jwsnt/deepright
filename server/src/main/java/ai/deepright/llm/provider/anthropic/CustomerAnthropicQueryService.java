package ai.deepright.llm.provider.anthropic;

import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.anthropic.AnthropicQueryService;
import ai.open.right.workflow.flow.llm.provider.anthropic.AnthropicRequest;
import ai.open.right.workflow.flow.llm.provider.anthropic.AnthropicStream;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
public class CustomerAnthropicQueryService extends AnthropicQueryService {

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

    @ConditionalOnProperty(name = "anthropic.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class CustomerInitConfig extends InitConfig {

        @Override
        @Bean(name = LLMQueryService.LLM_ANTHROPIC)
        @ConditionalOnMissingBean(name = LLMQueryService.LLM_ANTHROPIC)
        public AnthropicQueryService anthropicQueryService() throws Exception {
            CustomerAnthropicQueryService anthropicQueryService = new CustomerAnthropicQueryService();
            BeanUtils.copyProperties(this, anthropicQueryService);
            log.info("CustomerAnthropicQueryService inited");
            return anthropicQueryService;
        }
    }
}
