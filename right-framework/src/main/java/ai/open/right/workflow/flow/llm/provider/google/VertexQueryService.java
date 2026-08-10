package ai.open.right.workflow.flow.llm.provider.google;

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
public class VertexQueryService extends ProviderQueryService<GoogleRequest> {

    protected VertexRequestService vertexRequestService;

    protected VertexRouter vertexRouter;

    @Override
    protected GoogleStream stream(SignalStream signalStream, GoogleRequest request) throws Exception {
        return new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
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
    protected ProviderRequestService<GoogleRequest> request() {
        return this.vertexRequestService;
    }

    @Override
    protected ProviderRouter<GoogleRequest> router() {
        return this.vertexRouter;
    }

    @ConditionalOnProperty(name = "vertex.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderQueryInitConfig {

        @Autowired
        protected VertexRequestService vertexRequestService;

        @Autowired
        protected VertexRouter vertexRouter;

        @Bean(name = LLMQueryService.LLM_VERTEX)
        @ConditionalOnMissingBean(name = LLMQueryService.LLM_VERTEX)
        public VertexQueryService vertexQueryService() throws Exception {
            VertexQueryService vertexQueryService = new VertexQueryService();
            BeanUtils.copyProperties(this, vertexQueryService);
            log.info("VertexQueryService inited");
            return vertexQueryService;
        }
    }
}
