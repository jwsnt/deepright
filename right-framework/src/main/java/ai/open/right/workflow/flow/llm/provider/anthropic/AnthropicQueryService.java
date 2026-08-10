package ai.open.right.workflow.flow.llm.provider.anthropic;

import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.provider.ProviderQueryService;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Setter
@Getter
@Slf4j
public class AnthropicQueryService extends ProviderQueryService<AnthropicRequest> {

    protected AnthropicRequestService anthropicRequestService;

    protected AnthropicRouter anthropicRouter;

    @Override
    public AnthropicStream stream(SignalStream signalStream, AnthropicRequest request) throws Exception {
        return new AnthropicStream(ProviderStreamConfig.<AnthropicRequest>builder()
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

    @Override
    protected AnthropicRequestService request() {
        return this.anthropicRequestService;
    }

    @Override
    protected AnthropicRouter router() {
        return this.anthropicRouter;
    }

    @ConditionalOnProperty(name = "anthropic.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderQueryInitConfig {

        @Autowired
        @Qualifier(AnthropicRequestService.NAME)
        protected AnthropicRequestService anthropicRequestService;

        @Autowired
        @Qualifier(AnthropicRouter.NAME)
        protected AnthropicRouter anthropicRouter;

        @Bean(name = LLMQueryService.LLM_ANTHROPIC)
        @ConditionalOnMissingBean(name = LLMQueryService.LLM_ANTHROPIC)
        public AnthropicQueryService anthropicQueryService() throws Exception {
            AnthropicQueryService anthropicQueryService = new AnthropicQueryService();
            BeanUtils.copyProperties(this, anthropicQueryService);
            log.info("AnthropicRequestService inited");
            return anthropicQueryService;
        }
    }
}
