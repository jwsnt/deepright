package ai.deepright.llm.provider.seedream;

import ai.deepright.cli.CliSubFetcher;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.seedream.SeedreamQueryService;
import ai.open.right.workflow.flow.llm.provider.seedream.SeedreamRequest;
import ai.open.right.workflow.flow.llm.provider.seedream.SeedreamStream;
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

@Slf4j
@Getter
@Setter
public class CustomerSeedreamQueryService extends SeedreamQueryService {

    protected CliSubFetcher cliSubFetcher;

    @Override
    public SeedreamStream stream(SignalStream signalStream, SeedreamRequest request) throws Exception {
        return new CustomerSeedreamStream(ProviderStreamConfig.<SeedreamRequest>builder()
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
                .build().check(), this.cliSubFetcher);
    }

    @ConditionalOnProperty(name = "seedream.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class CustomerInitConfig extends InitConfig {

        @Autowired
        protected CliSubFetcher cliSubFetcher;

        @Override
        @Bean(name = LLMQueryService.LLM_SEEDREAM)
        @ConditionalOnMissingBean(name = LLMQueryService.LLM_SEEDREAM)
        public SeedreamQueryService seedreamQueryService() throws Exception {
            CustomerSeedreamQueryService seedreamQueryService = new CustomerSeedreamQueryService();
            BeanUtils.copyProperties(this, seedreamQueryService);
            log.info("CustomerSeedreamQueryService inited");
            return seedreamQueryService;
        }
    }
}
