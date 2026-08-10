package ai.open.right.workflow.flow.llm.provider.kimi;

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

import java.util.Collections;
import java.util.Map;
import java.util.Set;

@Setter
@Getter
@Slf4j
public class KimiRequestService extends OpenAiRequestService implements ProviderRequestModel {

    public static final Map<String, Object> THINK_CONFIG = ImmutableMap.of("type", "enabled");

    public static final String NAME = "KimiRequestService";

    public static final String MODEL = "kimi-k3";

    // Kimi模型
    protected String model = KimiRequestService.MODEL;

    // Kimi Token
    protected String token;

    @Override
    public String getModel(WorkflowTask workTask) throws Exception {
        return this.model;
    }

    @Override
    protected void request(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        super.request(request, llmConfig, llmQuery);
        request.setOpenAiMedia(KimiMedia.DEFAULT);
        this.adapt(request, llmConfig, llmQuery);
    }

    protected void adapt(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        OpenAiRequest openAiRequest = OpenAiRequest.class.cast(request);
        // https://platform.kimi.ai/docs/guide/kimi-k3-quickstart
        // 固定1.0
        openAiRequest.setTemperature(1.0);
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

    protected void reasoningEffort(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        String reasoningEffort = MapUtils.getString(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_REASONING_EFFORT);
        reasoningEffort = !StringUtils.isEmpty(reasoningEffort) ? reasoningEffort : MapUtils.getString(llmConfig.getAdditional(), ProviderRequestService.KEY_REASONING_EFFORT);
        reasoningEffort = !StringUtils.isEmpty(reasoningEffort) ? reasoningEffort : this.reasoningEffort;
        request.setReasoningEffort(reasoningEffort);
    }

    @Override
    protected void reasoning(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // https://platform.kimi.com/docs/guide/kimi-k3-quickstart#%E6%80%9D%E8%80%83%E5%8A%9B%E5%BA%A6
        // reasoning_effort配置同Open AI
        // K3始终开启思考模式，并支持通过请求reasoning_effort获取配置思考力量
        // completion = client.chat.completions.create(
        //    model="kimi-k3",
        //    messages=[
        //        {"role": "user", "content": "你好"}
        //    ],
        //    extra_body={
        //        "thinking": {"type": "disabled"}
        //    },
        //    max_tokens=1024*32,
        //)
        // K3强制开启思考模式，reasoning_effort
        if (StringUtils.containsIgnoreCase(request.getModel(), KimiRequestService.MODEL)) {
            // https://platform.kimi.com/docs/guide/use-kimi-k2-thinking-model
            request.setExtra(ProviderRequestService.KEY_THINKING, KimiRequestService.THINK_CONFIG);
            this.reasoningEffort(request, llmConfig, llmQuery);
        } else {
            Map<String, Object> thinking = MapUtils.getMap(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING);
            thinking = !MapUtils.isEmpty(thinking) ? thinking : MapUtils.getMap(llmConfig.getAdditional(), ProviderRequestService.KEY_THINKING);
            if (!MapUtils.isEmpty(thinking)) {
                request.setExtra(ProviderRequestService.KEY_THINKING, thinking);
                if (StringUtils.equalsAnyIgnoreCase(MapUtils.getString(thinking, "type"), "enabled", "adaptive")) {
                    this.reasoningEffort(request, llmConfig, llmQuery);
                }
            }
        }
    }

    @Override
    protected void extra(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
    }

    public static class KimiMedia extends OpenAiRequest.DefaultMedia {

        protected static final Set<String> VIDEO_TYPES = Collections.unmodifiableSet(Set.of(
                "video/mp4", "video/webm", "video/ogg"
        ));

        protected static final Set<String> AUDIO_TYPES = Collections.unmodifiableSet(Set.of(
                "audio/mpeg", "audio/mp3", "audio/wav"
        ));

        public static final String VIDEO = "video";

        public static final String AUDIO = "audio";

        public static final KimiMedia DEFAULT = new KimiMedia();

        public KimiMedia() {
            super();
            // 支持 图 视频 音频
            this.mimeTypes.addAll(KimiMedia.AUDIO_TYPES);
            this.mimeTypes.addAll(KimiMedia.VIDEO_TYPES);
        }

        @Override
        public String getKeyUrl(String type) throws Exception {
            // 仅支持图片和视频
            this.checkValid(type);
            // VIDEO / AUDIO / IMAGE
            return StringUtils.containsIgnoreCase(type, KimiMedia.VIDEO) || StringUtils.containsIgnoreCase(type, KimiMedia.AUDIO) ? "video_url" : super.getKeyUrl(type);
        }
    }

    @ConditionalOnProperty(name = "kimi.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderRequestInitConfig {

        @Value("${kimi.model:kimi-k3}")
        // Kimi模型
        protected String model = KimiRequestService.MODEL;

        @Value("${kimi.token:}")
        // Kimi Token
        protected String token;

        @Bean(name = KimiRequestService.NAME)
        @ConditionalOnMissingBean(name = KimiRequestService.NAME)
        public KimiRequestService kimiRequestService() throws Exception {
            KimiRequestService kimiRequestService = new KimiRequestService();
            BeanUtils.copyProperties(this, kimiRequestService);
            log.info("KimiRequestService inited. model={}, token={}, timeout={}", kimiRequestService.getModel(), StringUtils.repeat("*", StringUtils.length(kimiRequestService.getToken())), kimiRequestService.getFunCallTimeout());
            return kimiRequestService;
        }
    }
}