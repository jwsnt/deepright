package ai.open.right.workflow.flow.llm.provider.seedream;

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
public class SeedreamQueryService extends ProviderQueryService<SeedreamRequest> {

    protected SeedreamRequestService seedRequestService;

    protected SeedreamRouter seedRouter;

    @Override
    public SeedreamStream stream(SignalStream signalStream, SeedreamRequest seedRequest) throws Exception {
        return new SeedreamStream(ProviderStreamConfig.<SeedreamRequest>builder()
                .providerStorePolicy(this.providerStorePolicy)
                .trackFunCallService(this.trackFunCallService)
                .mediaInlineService(this.mediaInlineService)
                .notifierService(this.notifierService)
                .providerReason(this.providerReason)
                .tokenStatistic(this.tokenStatistic)
                .historyStore(this.historyStore)
                .namesService(this.namesService)
                .signalStream(signalStream)
                .request(seedRequest)
                .build().check());
    }

    @Override
    protected SeedreamRequestService request() {
        return this.seedRequestService;
    }

    @Override
    protected SeedreamRouter router() {
        return this.seedRouter;
    }

    @ConditionalOnProperty(name = "seedream.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderQueryInitConfig {

        @Autowired
        @Qualifier(SeedreamRequestService.NAME)
        protected SeedreamRequestService seedRequestService;

        @Autowired
        @Qualifier(SeedreamRouter.NAME)
        protected SeedreamRouter seedRouter;

        @Bean(name = LLMQueryService.LLM_SEEDREAM)
        @ConditionalOnMissingBean(name = LLMQueryService.LLM_SEEDREAM)
        public SeedreamQueryService seedreamQueryService() throws Exception {
            SeedreamQueryService seedQueryService = new SeedreamQueryService();
            BeanUtils.copyProperties(this, seedQueryService);
            log.info("SeedreamQueryService inited");
            return seedQueryService;
        }
    }
}
