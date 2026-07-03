package ai.deepright.llm.provider.deepseek;

import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.deepseek.DeepSeekQueryService;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiStream;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.deepright.llm.provider.openai.CustomerOpenAiStreamFunCall;
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
public class CustomerDeepSeekQueryService extends DeepSeekQueryService {

    @Override
    public OpenAiStream stream(SignalStream signalStream, OpenAiRequest request) throws Exception {
        return new CustomerOpenAiStreamFunCall(ProviderStreamConfig.<OpenAiRequest>builder()
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

    @ConditionalOnProperty(name = "deepseek.enable", havingValue = "true", matchIfMissing = false)
    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    @Setter
    @Getter
    public static class CustomerInitConfig extends InitConfig {

        @Override
        @Bean(name = LLMQueryService.LLM_DEEPSEEK)
        @ConditionalOnMissingBean(name = LLMQueryService.LLM_DEEPSEEK)
        public CustomerDeepSeekQueryService deepSeekQueryService() throws Exception {
            CustomerDeepSeekQueryService deepSeekQueryService = new CustomerDeepSeekQueryService();
            BeanUtils.copyProperties(this, deepSeekQueryService);
            log.info("CustomerDeepSeekQueryService inited");
            return deepSeekQueryService;
        }
    }
}
