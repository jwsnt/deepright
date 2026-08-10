package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.workflow.flow.llm.provider.ProviderImageConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Setter
@Getter
public class GoogleRequest extends ProviderRequest implements ProviderImageConfig {

    protected List<Map<String, Object>> safetySettings;

    // JSON Schema
    protected Map<String, Object> responseSchema;

    protected Map<String, Object> thinkingConfig;

    // 输出限制
    protected List<String> responseModalities;

    // "imageConfig": {
    //      "aspectRatio": "16:9",
    //      "imageSize": "2K"
    // }
    // imageSize: 1K、2K、4K
    protected Map<String, Object> imageConfig;

    protected Map<String, Object> toolsConfig;

    protected Map<String, String> labels;

    // https://www.yuque.com/shenjiawei-ejwcz/ebplfi/gv5cynr113cno74u
    protected Integer maxOutputTokens;

    protected Double frequencyPenalty;

    protected String mediaResolution;

    protected Double presencePenalty;

    protected Double temperature = 0.7;

    protected String mimeType;

    protected Integer seed;

    protected Integer topK;

    protected Double topP;

    @Override
    public Map<String, Object> getResponseSchema() {
        return this.responseSchema;
    }

    @Override
    public void setImageConfig(Map<String, Object> imageConfig) {
        if (!MapUtils.isEmpty(imageConfig)) {
            if (MapUtils.isEmpty(this.imageConfig)) {
                this.imageConfig = imageConfig;
            } else {
                this.imageConfig.putAll(imageConfig);
            }
        }
    }

    public Map<String, Object> configs() {
        if (this.responseModalities != null || this.thinkingConfig != null || this.maxOutputTokens != null || this.responseSchema != null || this.frequencyPenalty != null || this.presencePenalty != null || this.temperature != null || imageConfig != null || !StringUtils.isEmpty(this.mediaResolution) || !StringUtils.isEmpty(this.mimeType) || this.seed != null || this.topP != null || this.topK != null) {
            Map<String, Object> configs = new HashMap<String, Object>();
            configs.put(GoogleRequestService.KEY_RESPONSE_MODALITIES, this.responseModalities);
            configs.put(GoogleRequestService.KEY_FREQUENCY_PENALTY, this.frequencyPenalty);
            configs.put(GoogleRequestService.KEY_MAX_OUTPUT_TOKENS, this.maxOutputTokens);
            configs.put(GoogleRequestService.KEY_PRESENCE_PENALTY, this.presencePenalty);
            configs.put(GoogleRequestService.KEY_MEDIA_RESOLUTION, this.mediaResolution);
            configs.put(GoogleRequestService.KEY_THINKING_CONFIG, this.thinkingConfig);
            configs.put(GoogleRequestService.KEY_IMAGE_CONFIG, this.imageConfig);
            configs.put(GoogleRequestService.KEY_TEMPERATURE, this.temperature);
            configs.put(GoogleRequestService.KEY_MIMETYPE, this.mimeType);
            configs.put(GoogleRequestService.KEY_TOP_P, this.topP);
            configs.put(GoogleRequestService.KEY_TOP_K, this.topK);
            configs.put(GoogleRequestService.KEY_SEED, this.seed);
            if (this.responseSchema != null) {
                configs.put(GoogleRequestService.KEY_RESPONSE_MIME_TYPE, "application/json");
                configs.put(GoogleRequestService.KEY_RESPONSE_SCHEMA, this.responseSchema);
            }
            return configs;
        }
        return null;
    }
}
