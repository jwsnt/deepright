package ai.deepright.llm.provider.google;

import ai.deepright.llm.RetryUtils;
import ai.deepright.llm.provider.RequestContextUtils;
import ai.deepright.llm.provider.RequestModelSelect;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.google.GoogleRequest;
import ai.open.right.workflow.flow.llm.provider.google.VertexRequestService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
@Getter
@Setter
public class CustomerVertexRequestService extends VertexRequestService {

    protected String thinkingMedium;

    protected String thinkingHigh;

    protected String multiOutput;

    protected String multiInput;

    protected String thinking;

    protected String fast;

    protected String base;

    protected Double rate;

    @Override
    public GoogleRequest config(LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        RequestContextUtils.thinking(llmQuery, this.thinkingMedium, this.thinkingHigh);
        return super.config(llmConfig, llmQuery);
    }

    @Override
    protected Integer buildRecallOffset(GoogleRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return (int) (super.buildRecallOffset(request, llmConfig, llmQuery) * this.rate);
    }

    @Override
    protected Integer buildRecallNums(GoogleRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return (int) (super.buildRecallNums(request, llmConfig, llmQuery) * this.rate);
    }

    @Override
    protected void storeHistoryQuery(GoogleRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        RetryUtils.storeQuery(request, llmConfig, this.historyStore, this.buildHistoryQuery(request, llmConfig));
    }

    @Override
    protected void request(GoogleRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        super.request(request, llmConfig, llmQuery);
        request.setModel(RequestModelSelect.select(llmQuery, RequestModelSelect.RequestModel.builder()
                .multiOutput(this.multiOutput)
                .multiInput(this.multiInput)
                .thinking(this.thinking)
                .fast(this.fast)
                .base(this.base)
                .build()));
    }

    @Configuration
    @Setter
    @Getter
    public static class CustomerInitConfig extends InitConfig {

        @Value("${vertex.model.thinkingMedium:medium}")
        protected String thinkingMedium;

        @Value("${vertex.model.thinkingHigh:high}")
        protected String thinkingHigh;

        @Value("${vertex.model.multiOutput:gemini-3.1-flash-image}")
        protected String multiOutput;

        @Value("${vertex.model.multiInput:gemini-3.5-flash}")
        protected String multiInput;

        @Value("${vertex.model.thinking:gemini-3.5-flash}")
        protected String thinking;

        @Value("${vertex.model.fast:gemini-3.5-flash}")
        protected String fast;

        @Value("${vertex.model.base:gemini-3.5-flash}")
        protected String base;

        @Value("${vertex.model.rate:0.65}")
        protected Double rate;

        @Override
        @Bean(name = VertexRequestService.NAME)
        @ConditionalOnMissingBean(name = VertexRequestService.NAME)
        public CustomerVertexRequestService vertexRequestService() throws Exception {
            CustomerVertexRequestService vertexRequestService = new CustomerVertexRequestService();
            BeanUtils.copyProperties(this, vertexRequestService);
            log.info("CustomerVertexRequestService inited. tokenUri={}, policy={}, timeout={}", vertexRequestService.getTokenUri(), vertexRequestService.getPolicy(), vertexRequestService.getFunCallTimeout());
            return vertexRequestService;
        }
    }
}
