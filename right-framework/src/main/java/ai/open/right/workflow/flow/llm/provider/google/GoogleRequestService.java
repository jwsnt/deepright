package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import com.google.common.collect.ImmutableMap;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.util.Assert;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Setter
@Getter
abstract public class GoogleRequestService<T extends GoogleRequest> extends ProviderRequestService<T> {

    public static final String KEY_RESPONSE_MODALITIES = "responseModalities";

    public static final String KEY_RESPONSE_MIME_TYPE = "response_mime_type";

    public static final String KEY_FREQUENCY_PENALTY = "frequencyPenalty";

    public static final String KEY_PRESENCE_PENALTY = "presencePenalty";

    // https://www.yuque.com/shenjiawei-ejwcz/ebplfi/gv5cynr113cno74u
    public static final String KEY_MAX_OUTPUT_TOKENS = "maxOutputTokens";

    // https://docs.cloud.google.com/vertex-ai/generative-ai/docs/start/get-started-with-gemini-3?hl=zh-cn#media_resolution
    /// ////////////////////////////////////////////////////////
    /// https://docs.cloud.google.com/vertex-ai/generative-ai/docs/start/get-started-with-gemini-3?hl=zh-cn#media_resolution
    /// https://docs.cloud.google.com/vertex-ai/generative-ai/docs/reference/rpc/google.cloud.aiplatform.v1#mediaresolution
    /// "llm": {
    ///     "additional": {
    ///         "media_resolution": "MEDIA_RESOLUTION_LOW"
    ///     }
    /// }
    /// ////////////////////////////////////////////////////////
    public static final String KEY_MEDIA_RESOLUTION = "media_resolution";

    public static final String KEY_SAFETY_SETTINGS = "safetySettings";

    /// ////////////////////////////////////////////////////////
    /// https://ai.google.dev/gemini-api/docs/thinking?hl=zh-cn
    /// "llm": {
    ///     "additional": {
    ///         "thinkingConfig": {
    ///             "thinking_level": "low"
    ///         }
    ///     }
    /// }
    /// ////////////////////////////////////////////////////////
    public static final String KEY_THINKING_CONFIG = "thinkingConfig";

    public static final String KEY_IMAGE_CONFIG = "imageConfig";

    public static final String KEY_TOOL_CONFIG = "tool_config";

    public static final String KEY_TEMPERATURE = "temperature";

    /// ////////////////////////////////////////////////////////
    /// https://docs.cloud.google.com/vertex-ai/generative-ai/docs/model-reference/inference?hl=zh-cn
    /// ////////////////////////////////////////////////////////
    public static final String KEY_MIMETYPE = "mimeType";

    public static final String KEY_LABELS = "labels";

    public static final String KEY_TOP_P = "topP";

    public static final String KEY_TOP_K = "topK";

    protected List<Map<String, Object>> safeSettings;

    protected String reasoningEffort;

    protected String appName;

    public void init(String policy) throws Exception {
        Map<String, Object> HARM_CATEGORY_DANGEROUS_CONTENT = new HashMap<String, Object>();
        HARM_CATEGORY_DANGEROUS_CONTENT.put("category", "HARM_CATEGORY_DANGEROUS_CONTENT");
        HARM_CATEGORY_DANGEROUS_CONTENT.put("threshold", policy);
        Map<String, Object> HARM_CATEGORY_SEXUALLY_EXPLICIT = new HashMap<String, Object>();
        HARM_CATEGORY_SEXUALLY_EXPLICIT.put("category", "HARM_CATEGORY_SEXUALLY_EXPLICIT");
        HARM_CATEGORY_SEXUALLY_EXPLICIT.put("threshold", policy);
        Map<String, Object> HARM_CATEGORY_HATE_SPEECH = new HashMap<String, Object>();
        HARM_CATEGORY_HATE_SPEECH.put("category", "HARM_CATEGORY_HATE_SPEECH");
        HARM_CATEGORY_HATE_SPEECH.put("threshold", policy);
        Map<String, Object> HARM_CATEGORY_HARASSMENT = new HashMap<String, Object>();
        HARM_CATEGORY_HARASSMENT.put("category", "HARM_CATEGORY_HARASSMENT");
        HARM_CATEGORY_HARASSMENT.put("threshold", policy);
        this.safeSettings = List.of(HARM_CATEGORY_SEXUALLY_EXPLICIT, HARM_CATEGORY_DANGEROUS_CONTENT, HARM_CATEGORY_HATE_SPEECH, HARM_CATEGORY_HARASSMENT);
    }

    @Override
    public T config(LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        GoogleRequest request = super.config(llmConfig, llmQuery);
        Assert.hasText(request.getToken(), "Token can not be empty");
        return (T) request;
    }

    @Override
    protected void request(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        super.request(request, llmConfig, llmQuery);
        Object responseModalities = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + GoogleRequestService.KEY_RESPONSE_MODALITIES, llmConfig.getAdditional().get(GoogleRequestService.KEY_RESPONSE_MODALITIES));
        Object frequencyPenalty = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + GoogleRequestService.KEY_FREQUENCY_PENALTY, llmConfig.getAdditional().get(GoogleRequestService.KEY_FREQUENCY_PENALTY));
        Object maxOutputTokens = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + GoogleRequestService.KEY_MAX_OUTPUT_TOKENS, llmConfig.getAdditional().get(GoogleRequestService.KEY_MAX_OUTPUT_TOKENS));
        Object presencePenalty = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + GoogleRequestService.KEY_PRESENCE_PENALTY, llmConfig.getAdditional().get(GoogleRequestService.KEY_PRESENCE_PENALTY));
        Object mediaResolution = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + GoogleRequestService.KEY_MEDIA_RESOLUTION, llmConfig.getAdditional().get(GoogleRequestService.KEY_MEDIA_RESOLUTION));
        Object responseSchema = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + GoogleRequestService.KEY_RESPONSE_SCHEMA, llmConfig.getAdditional().get(GoogleRequestService.KEY_RESPONSE_SCHEMA));
        Object imageConfig = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + GoogleRequestService.KEY_IMAGE_CONFIG, llmConfig.getAdditional().get(GoogleRequestService.KEY_IMAGE_CONFIG));
        Object temperature = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + GoogleRequestService.KEY_TEMPERATURE, llmConfig.getAdditional().get(GoogleRequestService.KEY_TEMPERATURE));
        Object toolConfig = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + GoogleRequestService.KEY_TOOL_CONFIG, llmConfig.getAdditional().get(GoogleRequestService.KEY_TOOL_CONFIG));
        Object mimeType = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + GoogleRequestService.KEY_MIMETYPE, llmConfig.getAdditional().get(GoogleRequestService.KEY_MIMETYPE));
        Object topP = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + GoogleRequestService.KEY_TOP_P, llmConfig.getAdditional().get(GoogleRequestService.KEY_TOP_P));
        Object topK = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + GoogleRequestService.KEY_TOP_K, llmConfig.getAdditional().get(GoogleRequestService.KEY_TOP_K));
        Object seed = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + GoogleRequestService.KEY_SEED, llmConfig.getAdditional().get(GoogleRequestService.KEY_SEED));
        if (responseModalities != null) {
            request.setResponseModalities(List.class.cast(responseModalities));
        }
        if (frequencyPenalty != null) {
            request.setFrequencyPenalty(Double.valueOf(frequencyPenalty.toString()));
        }
        if (presencePenalty != null) {
            request.setPresencePenalty(Double.valueOf(presencePenalty.toString()));
        }
        if (maxOutputTokens != null) {
            request.setMaxOutputTokens(Integer.valueOf(maxOutputTokens.toString()));
        }
        if (temperature != null) {
            request.setTemperature(Double.valueOf(temperature.toString()));
        }
        if (mediaResolution != null) {
            request.setMediaResolution(String.class.cast(mediaResolution));
        }
        if (responseSchema != null) {
            request.setResponseSchema(Map.class.cast(responseSchema));
        }
        if (imageConfig != null) {
            request.setImageConfig(this.image(llmConfig, llmQuery, Map.class.cast(imageConfig)));
        }
        if (toolConfig != null) {
            request.setToolsConfig(Map.class.cast(toolConfig));
        }
        if (mimeType != null) {
            request.setMimeType(String.class.cast(mimeType));
        }
        if (seed != null) {
            request.setSeed(Integer.valueOf(seed.toString()));
        }
        if (topK != null) {
            request.setTopK(Integer.valueOf(topK.toString()));
        }
        if (topP != null) {
            request.setTopP(Double.valueOf(topP.toString()));
        }
        this.safetySettings(request, llmConfig, llmQuery);
        this.reasoning(request, llmConfig, llmQuery);
        request.setApi(ProviderRequest.REQUEST_GOOGLE);
    }

    protected Map<String, Object> image(LLMConfig llmConfig, LLMQuery llmQuery, Map<String, Object> imageConfig) throws Exception {
        // 如果请求里带imageConfig则覆盖默认配置中未配置的部分
        if (llmQuery.containMetadata(GoogleRequestService.KEY_IMAGE_CONFIG)) {
            Map<String, Object> requestConfig = llmQuery.getMetadata(GoogleRequestService.KEY_IMAGE_CONFIG, Map.class);
            if (requestConfig != null) {
                // 初始化：默认配置优先，请求中的 imageConfig 仅填充未配置的项
                imageConfig = imageConfig != null ? imageConfig : new HashMap<String, Object>();
                for (Map.Entry<String, Object> entry : requestConfig.entrySet()) {
                    imageConfig.putIfAbsent(entry.getKey(), entry.getValue());
                }
            }
        }
        return imageConfig;
    }

    protected void safetySettings(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // About SafetySetting: https://cloud.google.com/vertex-ai/generative-ai/docs/multimodal/configure-safety-filters?hl=zh-cn#harm_categories
        Object safetySettings = MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + GoogleRequestService.KEY_SAFETY_SETTINGS, llmConfig.getAdditional().get(GoogleRequestService.KEY_SAFETY_SETTINGS));
        if (safetySettings != null) {
            request.setSafetySettings(List.class.cast(safetySettings));
        } else {
            request.setSafetySettings(this.safeSettings);
        }
    }

    protected void reasoning(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // 谷歌私有配置
        Map<String, Object> googleThinking = MapUtils.getMap(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + GoogleRequestService.KEY_THINKING_CONFIG, Map.class.cast(llmConfig.getAdditional().get(GoogleRequestService.KEY_THINKING_CONFIG)));
        // OpenAi配置
        Map<String, Object> openAiThinking = MapUtils.getMap(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING);
        openAiThinking = !MapUtils.isEmpty(openAiThinking) ? openAiThinking : MapUtils.getMap(llmConfig.getAdditional(), ProviderRequestService.KEY_THINKING);
        if (StringUtils.equalsAnyIgnoreCase(MapUtils.getString(openAiThinking, "type"), "enabled", "adaptive") || !MapUtils.isEmpty(googleThinking)) {
            String reasoningEffort = MapUtils.getString(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_REASONING_EFFORT);
            reasoningEffort = !StringUtils.isEmpty(reasoningEffort) ? reasoningEffort : MapUtils.getString(llmConfig.getAdditional(), ProviderRequestService.KEY_REASONING_EFFORT);
            // 优先使用OpenAi兼容配置，如果不存在则回退谷歌配置，如果谷歌配置不存在则回退到默认值
            request.setThinkingConfig(!StringUtils.isEmpty(reasoningEffort) ? ImmutableMap.of("thinking_level", reasoningEffort) : !MapUtils.isEmpty(googleThinking) ? googleThinking : ImmutableMap.of("thinking_level", this.reasoningEffort));
        }
    }

    @Setter
    @Getter
    public static class GoogleRequestInitConfig extends ProviderRequestInitConfig {

        @Value("${google.model.reasoningEffort:low}")
        protected String reasoningEffort;

        @Value("${spring.application.name:}")
        protected String appName;
    }
}