package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.provider.ProviderQueryService;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.provider.ProviderRouter;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Setter
@Getter
@Slf4j
public class GeminiQueryService extends ProviderQueryService<GoogleRequest> {

    protected GeminiRequestService geminiRequestService;

    protected GeminiRouter geminiRouter;

    @Override
    protected GoogleStream stream(SignalStream signalStream, GoogleRequest request) throws Exception {
        return new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
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
    protected ProviderRequestService<GoogleRequest> request() {
        return this.geminiRequestService;
    }

    @Override
    protected ProviderRouter<GoogleRequest> router() {
        return this.geminiRouter;
    }

    @ConditionalOnProperty(name = "gemini.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderQueryInitConfig {

        @Autowired
        protected GeminiRequestService geminiRequestService;

        @Autowired
        protected GeminiRouter geminiRouter;

        @Bean(name = LLMQueryService.LLM_GEMINI)
        @ConditionalOnMissingBean(name = LLMQueryService.LLM_GEMINI)
        public GeminiQueryService geminiQueryService() throws Exception {
            GeminiQueryService geminiQueryService = new GeminiQueryService();
            BeanUtils.copyProperties(this, geminiQueryService);
            log.info("GeminiQueryService inited");
            return geminiQueryService;
        }
    }
}
