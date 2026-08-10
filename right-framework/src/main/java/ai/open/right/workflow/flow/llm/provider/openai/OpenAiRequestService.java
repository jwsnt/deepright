package ai.open.right.workflow.flow.llm.provider.openai;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestModel;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import com.google.common.collect.ImmutableMap;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
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
public class OpenAiRequestService extends ProviderRequestService<OpenAiRequest> implements ProviderRequestModel {

    public static final String KEY_FREQUENCY_PENALTY = "frequency_penalty";

    public static final String KEY_PRESENCE_PENALTY = "presence_penalty";

    public static final String KEY_PARALLEL_TOOL = "parallel_tool_calls";

    public static final String KEY_REASONING = "reasoning";

    public static final String KEY_TEMPERATURE = "temperature";

    public static final String KEY_MAX_TOKENS = "max_tokens";

    public static final String KEY_TOP_P = "top_p";

    public static final String NAME = "OpenAiRequestService";

    protected String reasoningEffort;

    protected Boolean parallelTool;

    protected String token;

    protected String model;

    @Override
    public OpenAiRequest build() throws Exception {
        return new OpenAiRequest();
    }

    @Override
    public OpenAiRequest config(LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        OpenAiRequest request = super.config(llmConfig, llmQuery);
        Assert.hasText(request.getToken(), "Token can not be empty");
        return request;
    }

    @Override
    public String getModel(WorkflowTask workTask) throws Exception {
        return this.model;
    }

    @Override
    protected void request(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        super.request(request, llmConfig, llmQuery);
        Object temperature = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + OpenAiRequestService.KEY_TEMPERATURE, llmConfig.getAdditional().getOrDefault(OpenAiRequestService.KEY_TEMPERATURE, "0.7"));
        Object frequencyPenalty = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + OpenAiRequestService.KEY_FREQUENCY_PENALTY, llmConfig.getAdditional().get(OpenAiRequestService.KEY_FREQUENCY_PENALTY));
        Object presencePenalty = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + OpenAiRequestService.KEY_PRESENCE_PENALTY, llmConfig.getAdditional().get(OpenAiRequestService.KEY_PRESENCE_PENALTY));
        Object responseSchema = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + OpenAiRequestService.KEY_RESPONSE_SCHEMA, llmConfig.getAdditional().get(OpenAiRequestService.KEY_RESPONSE_SCHEMA));
        Object maxTokens = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + OpenAiRequestService.KEY_MAX_TOKENS, llmConfig.getAdditional().get(OpenAiRequestService.KEY_MAX_TOKENS));
        Object mimeType = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + OpenAiRequestService.KEY_MIMETYPE, llmConfig.getAdditional().get(OpenAiRequestService.KEY_MIMETYPE));
        Object topP = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + OpenAiRequestService.KEY_TOP_P, llmConfig.getAdditional().get(OpenAiRequestService.KEY_TOP_P));
        if (temperature != null) {
            request.setTemperature(Double.valueOf(temperature.toString()));
        }
        if (frequencyPenalty != null) {
            request.setFrequencyPenalty(Double.valueOf(frequencyPenalty.toString()));
        }
        if (presencePenalty != null) {
            request.setPresencePenalty(Double.valueOf(presencePenalty.toString()));
        }
        if (responseSchema != null) {
            request.setResponseFormat(Map.class.cast(responseSchema));
        }
        if (maxTokens != null) {
            request.setMaxTokens(Integer.valueOf(maxTokens.toString()));
        }
        if (mimeType != null) {
            request.setMimeType(String.valueOf(mimeType));
        }
        if (topP != null) {
            request.setTopP(Double.valueOf(topP.toString()));
        }
        request.setExtraBody(MapUtils.getMap(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_EXTRA_BODY, MapUtils.getMap(llmConfig.getAdditional(), ProviderRequestService.KEY_EXTRA_BODY)));
        request.setApi(ProviderRequest.REQUEST_OPENAI);
        request.setFunCallStream(true);
        this.reasoning(request, llmConfig, llmQuery);
        this.extra(request, llmConfig, llmQuery);
        this.json(request, llmConfig, llmQuery);
    }

    @Override
    protected String buildToken(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return ProviderRequestService.KEY_PREFIX + super.buildToken(request, llmConfig, llmQuery);
    }

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

    protected void reasoning(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // https://developers.openai.com/api/docs/guides/reasoning
        // response = client.responses.create(
        // model="gpt-5.6",
        // reasoning={"effort": "low"},
        // input=[
        // {
        //  "role": "user",
        //  "content": prompt
        //  }
        //  ]
        //)
        String reasoning = MapUtils.getString(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + OpenAiRequestService.KEY_REASONING_EFFORT);
        reasoning = !StringUtils.isEmpty(reasoning) ? reasoning : MapUtils.getString(llmConfig.getAdditional(), OpenAiRequestService.KEY_REASONING_EFFORT);
        // 兼容标准协议thinking={"type": "enabled"}
        if (StringUtils.isEmpty(reasoning)) {
            // 回退检查type=enabled
            reasoning = StringUtils.equalsIgnoreCase(MapUtils.getString(MapUtils.getMap(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING), "type"), "enabled") || StringUtils.equalsIgnoreCase(MapUtils.getString(MapUtils.getMap(llmConfig.getAdditional(), ProviderRequestService.KEY_THINKING), "type"), "enabled") ? this.reasoningEffort : null;
        }
        if (!StringUtils.isEmpty(reasoning)) {
            request.setExtra(OpenAiRequestService.KEY_REASONING, ImmutableMap.of("effort", reasoning));
        }
    }

    protected void extra(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // ParallelTool在无工具时禁止开启
        if (this.parallelTool && !CollectionUtils.isEmpty(request.getFunCalls())) {
            request.setExtra(OpenAiRequestService.KEY_PARALLEL_TOOL, true);
        }
    }

    // 子类决定调用
    protected void json(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        if (!MapUtils.isEmpty(request.getResponseFormat())) {
            // 重写类型（兼容配置）
            request.setResponseFormat(ImmutableMap.of("type", "json_object"));
        }
    }

    @ConditionalOnProperty(name = "openai.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderRequestInitConfig {

        @Value("${openai.model.reasoningEffort:low}")
        protected String reasoningEffort;

        @Value("${openai.model.parallel_tool:true}")
        protected Boolean parallelTool;

        @Value("${openai.token:}")
        protected String token;

        @Value("${openai.model:gpt-5.6-terra}")
        protected String model;

        @Bean(name = OpenAiRequestService.NAME)
        @ConditionalOnMissingBean(name = OpenAiRequestService.NAME)
        public OpenAiRequestService openAiRequestService() throws Exception {
            OpenAiRequestService openAiRequestService = new OpenAiRequestService();
            BeanUtils.copyProperties(this, openAiRequestService);
            log.info("OpenAiRequestService inited, timeout={}", openAiRequestService.getFunCallTimeout());
            return openAiRequestService;
        }
    }
}
