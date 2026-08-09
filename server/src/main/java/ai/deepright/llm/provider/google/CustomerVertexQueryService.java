package ai.deepright.llm.provider.google;

import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.google.GoogleRequest;
import ai.open.right.workflow.flow.llm.provider.google.GoogleStream;
import ai.open.right.workflow.flow.llm.provider.google.VertexQueryService;
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
public class CustomerVertexQueryService extends VertexQueryService {

    @Override
    public GoogleStream stream(SignalStream signalStream, GoogleRequest request) throws Exception {
        return new CustomerGoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
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

    @ConditionalOnProperty(name = "vertex.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class CustomerInitConfig extends InitConfig {

        @Override
        @Bean(name = LLMQueryService.LLM_VERTEX)
        @ConditionalOnMissingBean(name = LLMQueryService.LLM_VERTEX)
        public CustomerVertexQueryService vertexQueryService() throws Exception {
            CustomerVertexQueryService vertexQueryService = new CustomerVertexQueryService();
            BeanUtils.copyProperties(this, vertexQueryService);
            log.info("CustomerVertexQueryService inited");
            return vertexQueryService;
        }
    }
}
