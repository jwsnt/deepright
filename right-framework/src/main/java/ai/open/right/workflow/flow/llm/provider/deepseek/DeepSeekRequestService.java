package ai.open.right.workflow.flow.llm.provider.deepseek;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestModel;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequestService;
import com.google.common.collect.ImmutableMap;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.util.Map;

@Setter
@Getter
@Slf4j
public class DeepSeekRequestService extends OpenAiRequestService implements ProviderRequestModel {

    public static final Map<String, Object> THINK_CONFIG = ImmutableMap.of("type", "enabled");

    public static final String NAME = "DeepSeekRequestService";

    private String reasoningEffort;

    // DeepSeek模型
    protected String model;

    // DeepSeek Token
    protected String token;

    @Override
    protected String defToken(WorkflowTask workTask) throws Exception {
        return StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), "__token"), this.token);
    }

    @Override
    protected String defModel(WorkflowTask workTask) throws Exception {
        String model = StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), "__model"), this.getModel(workTask));
        Assert.hasText(model, "The model can not be empty");
        return model;
    }

    @Override
    public String getModel(WorkflowTask workTask) throws Exception {
        return this.model;
    }

    @Override
    protected void reasoning(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // https://api-docs.deepseek.com/zh-cn/guides/thinking_mode
        // response = client.chat.completions.create(
        // model="deepseek-v4-pro",
        // # ...
        // reasoning_effort="high",
        // extra_body={"thinking": {"type": "enabled"}}
        //)
        // 同标准协议thinking={"type": "enabled"}，D4 强制开启思考
        Map<String, Object> thinking = MapUtils.getMap(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING);
        thinking = !MapUtils.isEmpty(thinking) ? thinking : MapUtils.getMap(llmConfig.getAdditional(), ProviderRequestService.KEY_THINKING);
        thinking = !MapUtils.isEmpty(thinking) ? thinking : DeepSeekRequestService.THINK_CONFIG;
        request.setExtra(ProviderRequestService.KEY_THINKING, thinking);
        String reasoningEffort = MapUtils.getString(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_REASONING_EFFORT);
        reasoningEffort = !StringUtils.isEmpty(reasoningEffort) ? reasoningEffort : MapUtils.getString(llmConfig.getAdditional(), ProviderRequestService.KEY_REASONING_EFFORT);
        reasoningEffort = !StringUtils.isEmpty(reasoningEffort) ? reasoningEffort : this.reasoningEffort;
        request.setReasoningEffort(reasoningEffort);
    }

    @Override
    protected void extra(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
    }

    @ConditionalOnProperty(name = "deepseek.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderRequestInitConfig {

        @Value("${deepseek.model.reasoningEffort:low}")
        protected String reasoningEffort;

        @Value("${deepseek.model:deepseek-v4-flash}")
        // DeepSeek模型
        protected String model;

        @Value("${deepseek.token:}")
        // DeepSeek Token
        protected String token;

        @Bean(name = DeepSeekRequestService.NAME)
        @ConditionalOnMissingBean(name = DeepSeekRequestService.NAME)
        public DeepSeekRequestService deepSeekRequestService() throws Exception {
            DeepSeekRequestService deepSeekRequestService = new DeepSeekRequestService();
            BeanUtils.copyProperties(this, deepSeekRequestService);
            log.info("DeepSeekRequestService inited. model={}, token={}, timeout={}", deepSeekRequestService.getModel(), StringUtils.repeat("*", StringUtils.length(deepSeekRequestService.getToken())), deepSeekRequestService.getFunCallTimeout());
            return deepSeekRequestService;
        }
    }
}
