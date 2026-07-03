package ai.deepright.llm.provider.deepseek;

import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.deepseek.DeepSeekRequestService;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.deepright.complex.ComplexityMode;
import ai.deepright.complex.ComplexityUtils;
import ai.deepright.llm.RetryUtils;
import ai.deepright.llm.provider.RequestModelSelect;
import com.google.common.collect.ImmutableMap;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;

import java.util.HashMap;
import java.util.Map;

@Slf4j
@Getter
@Setter
public class CustomerDeepSeekRequestService extends DeepSeekRequestService {

    protected String thinking;

    protected String fast;

    protected String base;

    @Override
    protected void storeHistoryQuery(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        RetryUtils.storeQuery(request, llmConfig, this.historyStore, this.buildHistoryQuery(request, llmConfig));
    }

    @Override
    public OpenAiRequest config(LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        OpenAiRequest request = super.config(llmConfig, llmQuery);
        request.setModel(RequestModelSelect.select(llmQuery, RequestModelSelect.RequestModel.builder()
                .thinking(this.thinking)
                .fast(this.fast)
                .base(this.base)
                .build()));
        return request;
    }

    @Override
    protected void extra(OpenAiRequest request) throws Exception {
        super.extra(request);
        ComplexityMode lastResult = ComplexityUtils.result(request.getMessage());
        if (lastResult.is(ComplexityMode.DEEP_THINKING, ComplexityMode.TASK_PLANNING)) {
            Map<String, Object> extra = request.getExtraBody();
            extra = extra != null ? extra : new HashMap<String, Object>();
            extra.put("reasoning_effort", lastResult.is(ComplexityMode.DEEP_THINKING) ? "high" : "max");
            extra.put("thinking", ImmutableMap.of("type", "enabled"));
            request.setExtraBody(extra);
        }
    }

    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    @Setter
    @Getter
    public static class CustomerInitConfig extends InitConfig {

        @Value("${deepseek.model.thinking:deepseek-v4-pro}")
        protected String thinking;

        @Value("${deepseek.model.fast:deepseek-v4-flash}")
        protected String fast;

        @Value("${deepseek.model.base:deepseek-v4-flash}")
        protected String base;

        @Override
        @Bean(name = DeepSeekRequestService.NAME)
        @ConditionalOnMissingBean(name = DeepSeekRequestService.NAME)
        public DeepSeekRequestService deepSeekRequestService() throws Exception {
            CustomerDeepSeekRequestService deepSeekRequestService = new CustomerDeepSeekRequestService();
            BeanUtils.copyProperties(this, deepSeekRequestService);
            log.info("CustomerDeepSeekModelRequestService inited, timeout={}", deepSeekRequestService.getFunCallTimeout());
            return deepSeekRequestService;
        }
    }
}
