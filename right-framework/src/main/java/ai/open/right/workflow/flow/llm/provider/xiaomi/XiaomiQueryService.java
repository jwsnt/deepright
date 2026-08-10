package ai.open.right.workflow.flow.llm.provider.xiaomi;

import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.provider.ProviderRouter;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiQueryService;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiStream;
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
public class XiaomiQueryService extends OpenAiQueryService {

    protected XiaomiRequestService xiaomiRequestService;

    protected XiaomiRouter xiaomiRouter;

    @Override
    public OpenAiStream stream(SignalStream signalStream, OpenAiRequest request) throws Exception {
        return new XiaomiStream(ProviderStreamConfig.<OpenAiRequest>builder()
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
    protected ProviderRequestService<OpenAiRequest> request() {
        return this.xiaomiRequestService;
    }

    @Override
    protected ProviderRouter<OpenAiRequest> router() {
        return this.xiaomiRouter;
    }

    @ConditionalOnProperty(name = "xiaomi.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderQueryInitConfig {

        @Autowired
        protected XiaomiRequestService xiaomiRequestService;

        @Autowired
        protected XiaomiRouter xiaomiRouter;

        @Bean(name = LLMQueryService.LLM_XIAOMI)
        @ConditionalOnMissingBean(name = LLMQueryService.LLM_XIAOMI)
        public XiaomiQueryService xiaomiQueryService() throws Exception {
            XiaomiQueryService xiaomiQueryService = new XiaomiQueryService();
            BeanUtils.copyProperties(this, xiaomiQueryService);
            log.info("XiaomiQueryService inited");
            return xiaomiQueryService;
        }
    }
}
