package ai.open.right.workflow.flow.llm.provider.seedream;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestModel;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequestService;
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

@Setter
@Getter
@Slf4j
public class SeedreamRequestService extends ProviderRequestService<SeedreamRequest> implements ProviderRequestModel {

    public static final String KEY_SEQUENTIAL_IMAGE_GENERATION_OPTIONS = "sequential_image_generation_options";

    public static final String KEY_SEQUENTIAL_IMAGE_GENERATION = "sequential_image_generation";

    public static final String KEY_OPTIMIZE_PROMPT_OPTIONS = "optimize_prompt_options";

    public static final String KEY_RESPONSE_FORMAT = "response_format";

    public static final String KEY_GUIDANCE_SCALE = "guidance_scale";

    // 参考图数量限制
    public static final String KEY_IMAGES = "images";

    public static final String KEY_SIZE = "size";

    public static final String NAME = "SeedRequestService";

    // https://console.volcengine.com/ark/region:ark+cn-beijing/model/detail?Id=doubao-seedream-4-5
    protected String model;

    protected String token;

    @Override
    public String getModel(WorkflowTask workTask) throws Exception {
        return this.model;
    }

    @Override
    protected SeedreamRequest build() throws Exception {
        return new SeedreamRequest();
    }

    @Override
    protected void request(SeedreamRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        super.request(request, llmConfig, llmQuery);
        request.setSequential(MapUtils.getString(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + SeedreamRequestService.KEY_SEQUENTIAL_IMAGE_GENERATION, MapUtils.getString(llmConfig.getAdditional(), SeedreamRequestService.KEY_SEQUENTIAL_IMAGE_GENERATION, "disabled")));
        request.setSequentialOptions(MapUtils.getMap(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + SeedreamRequestService.KEY_SEQUENTIAL_IMAGE_GENERATION_OPTIONS, MapUtils.getMap(llmConfig.getAdditional(), SeedreamRequestService.KEY_SEQUENTIAL_IMAGE_GENERATION_OPTIONS)));
        request.setFormat(MapUtils.getString(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + SeedreamRequestService.KEY_RESPONSE_FORMAT, MapUtils.getString(llmConfig.getAdditional(), SeedreamRequestService.KEY_RESPONSE_FORMAT, "url")));
        request.setOptimizeOptions(MapUtils.getMap(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + SeedreamRequestService.KEY_OPTIMIZE_PROMPT_OPTIONS, MapUtils.getMap(llmConfig.getAdditional(), SeedreamRequestService.KEY_OPTIMIZE_PROMPT_OPTIONS)));
        request.setGuidance(MapUtils.getDouble(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + SeedreamRequestService.KEY_GUIDANCE_SCALE, MapUtils.getDouble(llmConfig.getAdditional(), SeedreamRequestService.KEY_GUIDANCE_SCALE)));
        request.setImages(MapUtils.getInteger(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + SeedreamRequestService.KEY_IMAGES, MapUtils.getInteger(llmConfig.getAdditional(), SeedreamRequestService.KEY_IMAGES, 1)));
        request.setSize(MapUtils.getString(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + SeedreamRequestService.KEY_SIZE, MapUtils.getString(llmConfig.getAdditional(), SeedreamRequestService.KEY_SIZE, "2k")));
        request.setMimeType(MapUtils.getString(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + SeedreamRequestService.KEY_MIMETYPE, MapUtils.getString(llmConfig.getAdditional(), OpenAiRequestService.KEY_MIMETYPE)));
        request.setSeed(MapUtils.getInteger(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + OpenAiRequestService.KEY_SEED, MapUtils.getInteger(llmConfig.getAdditional(), OpenAiRequestService.KEY_SEED)));
        request.setApi(ProviderRequest.REQUEST_SEEDREAM);
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

    @ConditionalOnProperty(name = "seedream.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderRequestInitConfig {

        @Value("${seedream.model:doubao-seedream-4-5-251128}")
        protected String model;

        @Value("${seedream.token:}")
        protected String token;

        @Bean(name = SeedreamRequestService.NAME)
        @ConditionalOnMissingBean(name = SeedreamRequestService.NAME)
        public SeedreamRequestService seedreamRequestService() throws Exception {
            SeedreamRequestService seedRequestService = new SeedreamRequestService();
            BeanUtils.copyProperties(this, seedRequestService);
            log.info("SeedreamRequestService inited. model={}, token={}, timeout={}", seedRequestService.getModel(), StringUtils.repeat("*", StringUtils.length(seedRequestService.getToken())), seedRequestService.getFunCallTimeout());
            return seedRequestService;
        }
    }
}
