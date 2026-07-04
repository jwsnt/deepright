package ai.deepright.llm.provider.volcengine;

import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiStream;
import ai.open.right.workflow.flow.llm.provider.volcengine.VolcengineQueryService;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.deepright.llm.provider.openai.CustomerOpenAiStream;
import ai.deepright.llm.provider.openai.CustomerOpenAiStreamFunCall;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
public class CustomerVolcengineQueryService extends VolcengineQueryService {

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

    @ConditionalOnProperty(name = "volcengine.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class CustomerInitConfig extends InitConfig {

        @Override
        @Bean(name = LLMQueryService.LLM_VOLCENGINE)
        @ConditionalOnMissingBean(name = LLMQueryService.LLM_VOLCENGINE)
        public CustomerVolcengineQueryService volcengineQueryService() throws Exception {
            CustomerVolcengineQueryService volcengineQueryService = new CustomerVolcengineQueryService();
            BeanUtils.copyProperties(this, volcengineQueryService);
            log.info("CustomerVolcengineQueryService inited");
            return volcengineQueryService;
        }
    }
}
