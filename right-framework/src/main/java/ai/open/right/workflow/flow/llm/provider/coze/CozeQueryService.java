package ai.open.right.workflow.flow.llm.provider.coze;

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
public class CozeQueryService extends ProviderQueryService<CozeRequest> {

    protected CozeRequestService cozeRequestService;

    protected CozeRouter cozeRouter;

    @Override
    protected CozeStream stream(SignalStream signalStream, CozeRequest request) throws Exception {
        return new CozeStream(ProviderStreamConfig.<CozeRequest>builder()
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
    protected ProviderRequestService<CozeRequest> request() {
        return this.cozeRequestService;
    }

    @Override
    protected ProviderRouter<CozeRequest> router() {
        return this.cozeRouter;
    }

    @ConditionalOnProperty(name = "coze.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderQueryInitConfig {

        @Autowired
        protected CozeRequestService cozeRequestService;

        @Autowired
        protected CozeRouter cozeRouter;

        @Bean(name = LLMQueryService.LLM_COZE)
        @ConditionalOnMissingBean(name = LLMQueryService.LLM_COZE)
        public CozeQueryService cozeQueryService() throws Exception {
            CozeQueryService cozeQueryService = new CozeQueryService();
            BeanUtils.copyProperties(this, cozeQueryService);
            log.info("CozeQueryService inited");
            return cozeQueryService;
        }
    }
}
