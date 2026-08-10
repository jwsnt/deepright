package ai.open.right.workflow.flow.llm.provider.bigmodel;

import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.provider.ProviderRouter;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiQueryService;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiStream;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiStreamFunCall;
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
public class BigModelQueryService extends OpenAiQueryService {

    protected BigModelRequestService bigModelRequestService;

    protected BigModelRouter bigModelRouter;

    @Override
    public OpenAiStream stream(SignalStream signalStream, OpenAiRequest request) throws Exception {
        return new OpenAiStreamFunCall(ProviderStreamConfig.<OpenAiRequest>builder()
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
        return this.bigModelRequestService;
    }

    @Override
    protected ProviderRouter<OpenAiRequest> router() {
        return this.bigModelRouter;
    }

    @ConditionalOnProperty(name = "bigmodel.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderQueryInitConfig {

        @Autowired
        protected BigModelRequestService bigModelRequestService;

        @Autowired
        protected BigModelRouter bigModelRouter;

        @Bean(name = LLMQueryService.LLM_BIGMODEL)
        @ConditionalOnMissingBean(name = LLMQueryService.LLM_BIGMODEL)
        public BigModelQueryService bigModelQueryService() throws Exception {
            BigModelQueryService bigModelQueryService = new BigModelQueryService();
            BeanUtils.copyProperties(this, bigModelQueryService);
            log.info("BigModelQueryService inited");
            return bigModelQueryService;
        }
    }
}

