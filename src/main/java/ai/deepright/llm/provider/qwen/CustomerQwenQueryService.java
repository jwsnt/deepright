package ai.deepright.llm.provider.qwen;

import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiStream;
import ai.open.right.workflow.flow.llm.provider.qwen.QwenQueryService;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
public class CustomerQwenQueryService extends QwenQueryService {

    @Override
    public OpenAiStream stream(SignalStream signalStream, OpenAiRequest request) throws Exception {
        return new CustomerQwenStream(ProviderStreamConfig.<OpenAiRequest>builder()
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

    @ConditionalOnProperty(name = "qwen.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class CustomerInitConfig extends InitConfig {

        @Override
        @Bean(name = LLMQueryService.LLM_QWEN)
        @ConditionalOnMissingBean(name = LLMQueryService.LLM_QWEN)
        public CustomerQwenQueryService qwenQueryService() throws Exception {
            CustomerQwenQueryService qwenQueryService = new CustomerQwenQueryService();
            BeanUtils.copyProperties(this, qwenQueryService);
            log.info("CustomerQwenQueryService inited");
            return qwenQueryService;
        }
    }
}
