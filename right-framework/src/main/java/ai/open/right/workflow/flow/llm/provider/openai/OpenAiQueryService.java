package ai.open.right.workflow.flow.llm.provider.openai;

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
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Setter
@Getter
@Slf4j
public class OpenAiQueryService extends ProviderQueryService<OpenAiRequest> {

    protected OpenAiRequestService openAiRequestService;

    protected OpenAiRouter openAiRouter;

    @Override
    public OpenAiStream stream(SignalStream signalStream, OpenAiRequest request) throws Exception {
        ProviderStreamConfig<OpenAiRequest> providerRequestConfig = ProviderStreamConfig.<OpenAiRequest>builder()
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
                .build().check();
        return request.getFunCallStream() ? new OpenAiStreamFunCall(providerRequestConfig) : new OpenAiStream(providerRequestConfig);
    }

    @Override
    protected ProviderRequestService<OpenAiRequest> request() {
        return this.openAiRequestService;
    }

    @Override
    protected ProviderRouter<OpenAiRequest> router() {
        return this.openAiRouter;
    }

    @Setter
    @Getter
    public static class OpenAiQueryInitConfig extends ProviderQueryInitConfig {

        @Autowired
        @Qualifier(OpenAiRequestService.NAME)
        protected OpenAiRequestService openAiRequestService;

        @Autowired
        @Qualifier(OpenAiRouter.NAME)
        protected OpenAiRouter openAiRouter;
    }

    @ConditionalOnProperty(name = "openai.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends OpenAiQueryInitConfig {

        @Bean(name = LLMQueryService.LLM_OPENAI)
        @ConditionalOnMissingBean(name = LLMQueryService.LLM_OPENAI)
        public OpenAiQueryService openAiQueryService() throws Exception {
            OpenAiQueryService openAiQueryService = new OpenAiQueryService();
            BeanUtils.copyProperties(this, openAiQueryService);
            log.info("OpenAiQueryService inited.");
            return openAiQueryService;
        }
    }

}
