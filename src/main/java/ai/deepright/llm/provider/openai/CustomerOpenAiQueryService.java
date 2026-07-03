package ai.deepright.llm.provider.openai;

import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiQueryService;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiStream;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
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
public class CustomerOpenAiQueryService extends OpenAiQueryService {

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
                .request(request).build().check();
        return request.getFunCallStream() ? new CustomerOpenAiStreamFunCall(providerRequestConfig) : new CustomerOpenAiStream(providerRequestConfig);
    }

    @ConditionalOnProperty(name = "openai.enable", havingValue = "true", matchIfMissing = false)
    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    @Setter
    @Getter
    public static class CustomerInitConfig extends InitConfig {

        @Override
        @Bean(name = LLMQueryService.LLM_OPENAI)
        @ConditionalOnMissingBean(name = LLMQueryService.LLM_OPENAI)
        public CustomerOpenAiQueryService openAiQueryService() throws Exception {
            CustomerOpenAiQueryService openAiQueryService = new CustomerOpenAiQueryService();
            BeanUtils.copyProperties(this, openAiQueryService);
            log.info("CustomerOpenAiQueryService inited");
            return openAiQueryService;
        }
    }
}
