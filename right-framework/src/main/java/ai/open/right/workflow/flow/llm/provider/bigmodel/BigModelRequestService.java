package ai.open.right.workflow.flow.llm.provider.bigmodel;

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
public class BigModelRequestService extends OpenAiRequestService implements ProviderRequestModel {

    public static final Map<String, Object> THINK_CONFIG = ImmutableMap.of("type", "enabled");

    public static final String NAME = "BigModelRequestService";

    private String reasoningEffort;

    // BigModel模型
    protected String model;

    // BigModel Token
    protected String token;

    @Override
    protected void request(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        super.request(request, llmConfig, llmQuery);
        request.setOpenAiMedia(BigModelMedia.DEFAULT);
        request.setFunCallStream(false);
        this.json(request, llmConfig, llmQuery);
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

    @Override
    public String getModel(WorkflowTask workTask) throws Exception {
        return this.model;
    }

    @Override
    protected void reasoning(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // https://docs.bigmodel.cn/cn/guide/develop/openai/introduction#%E6%8E%A8%E7%90%86%EF%BC%88thinking%EF%BC%89
        // https://docs.bigmodel.cn/cn/guide/start/migrate-to-glm-new#3-%E6%B7%B1%E5%BA%A6%E6%80%9D%E8%80%83%EF%BC%88%E5%8F%AF%E9%80%89%EF%BC%89
        // resp = client.chat.completions.create(
        // model="glm-5.2",
        // messages=[{"role": "user", "content": "为我设计一个三层微服务架构"}],
        // thinking={"type": "enabled"},
        // reasoning_effort="max"
        // )
        // 同标准协议thinking={"type": "enabled"}，GLM 强制开启思考
        Map<String, Object> thinking = MapUtils.getMap(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING);
        thinking = !MapUtils.isEmpty(thinking) ? thinking : MapUtils.getMap(llmConfig.getAdditional(), ProviderRequestService.KEY_THINKING);
        thinking = !MapUtils.isEmpty(thinking) ? thinking : BigModelRequestService.THINK_CONFIG;
        request.setExtra(ProviderRequestService.KEY_THINKING, thinking);
        String reasoningEffort = MapUtils.getString(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_REASONING_EFFORT);
        reasoningEffort = !StringUtils.isEmpty(reasoningEffort) ? reasoningEffort : MapUtils.getString(llmConfig.getAdditional(), ProviderRequestService.KEY_REASONING_EFFORT);
        reasoningEffort = !StringUtils.isEmpty(reasoningEffort) ? reasoningEffort : this.reasoningEffort;
        request.setReasoningEffort(reasoningEffort);
    }

    @Override
    protected void extra(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
    }

    public static class BigModelMedia extends OpenAiRequest.DefaultMedia {

        protected static final Set<String> TEXT_TYPES = Collections.unmodifiableSet(Set.of("application/pdf", "text/plain"));

        protected static final Set<String> VIDEO_TYPES = Collections.unmodifiableSet(Set.of("video/mp4", "video/webm", "video/ogg"));

        protected static final Set<String> AUDIO_TYPES = Collections.unmodifiableSet(Set.of("audio/mpeg", "audio/mp3", "audio/wav"));

        protected static final String VIDEO = "video";

        protected static final String AUDIO = "audio";

        protected static final String IMAGE = "image";

        public static final BigModelMedia DEFAULT = new BigModelMedia();

        public BigModelMedia() {
            super();
            this.mimeTypes.addAll(BigModelMedia.VIDEO_TYPES);
            this.mimeTypes.addAll(BigModelMedia.AUDIO_TYPES);
            this.mimeTypes.addAll(BigModelMedia.TEXT_TYPES);
        }

        @Override
        public String getKeyUrl(String type) throws Exception {
            // 仅支持图片和视频
            this.checkValid(type);
            // VIDEO / AUDIO / IMAGE / FILE
            if (StringUtils.containsIgnoreCase(type, BigModelMedia.VIDEO) || StringUtils.containsIgnoreCase(type, BigModelMedia.AUDIO)) {
                return "video_url";
            } else if (StringUtils.containsIgnoreCase(type, BigModelMedia.IMAGE)) {
                return "image_url";
            } else {
                return "file_url";
            }
        }
    }

    @ConditionalOnProperty(name = "bigmodel.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderRequestInitConfig {

        @Value("${bigmodel.model.reasoningEffort:low}")
        protected String reasoningEffort;

        @Value("${bigmodel.model:glm-5.2}")
        // BigModel模型
        protected String model = "glm-5.2";

        @Value("${bigmodel.token:}")
        // BigModel Token
        protected String token;

        @Bean(name = BigModelRequestService.NAME)
        @ConditionalOnMissingBean(name = BigModelRequestService.NAME)
        public BigModelRequestService bigModelRequestService() throws Exception {
            BigModelRequestService bigModelRequestService = new BigModelRequestService();
            BeanUtils.copyProperties(this, bigModelRequestService);
            log.info("BigModelRequestService inited. model={}, token={}, timeout={}", bigModelRequestService.getModel(), StringUtils.repeat("*", StringUtils.length(bigModelRequestService.getToken())), bigModelRequestService.getFunCallTimeout());
            return bigModelRequestService;
        }
    }
}