package ai.open.right.workflow.flow.llm.provider.seedream;

import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderReader;
import ai.open.right.workflow.flow.llm.provider.ProviderReaderConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRouter;
import ai.open.right.workflow.flow.media.MediaContext;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.apache.http.client.methods.HttpPost;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

@Setter
@Getter
@Slf4j
public class SeedreamRouter extends ProviderRouter<SeedreamRequest> {

    public static final String NAME = "SeedreamRouter";

    protected String url;

    @Override
    public void reConfig(SeedreamRequest request, LLMConfig llmConfig, HttpPost httpPost) throws Exception {
        super.reConfig(request, llmConfig, httpPost);
        httpPost.addHeader("Authorization", request.getToken());
    }

    @Override
    protected SeedreamReader reader(SeedreamRequest request, LLMConfig llmConfig, LLMCallback llmCallback) throws Exception {
        return new SeedreamReader(ProviderReaderConfig.<SeedreamRequest>builder()
                .buffer(llmConfig.hasNetworkBuffer() ? llmConfig.getNetworkBuffer() : this.buffer)
                .eventListenerService(this.eventListenerService)
                .notifierService(this.notifierService)
                .extension(ProviderReader.EXTENSION)
                .timeout(this.queueTimeout)
                .llmCallback(llmCallback)
                .capacity(this.capacity)
                .discard(this.discard)
                .request(request)
                .queue(this.queue).build().check());
    }

    @Override
    public String url(SeedreamRequest request, LLMConfig llmConfig, String t) throws Exception {
        String url = MapUtils.getString(request.getMessage().getMetadata(), "__url", StringUtils.defaultIfEmpty(request.getUrl(), this.url));
        Assert.hasText(url, "Url can not be empty");
        return url;
    }

    @Override
    public Object body(SeedreamRequest request) throws Exception {
        return new SeedreamMessage(request);
    }

    @Getter
    public static class SeedreamMessage {

        @JsonProperty("sequential_image_generation_options")
        protected Map<String, Object> sequentialOptions;

        @JsonProperty("optimize_prompt_options")
        protected Map<String, Object> optimizeOptions;

        // 固定
        protected Boolean watermark = false;

        @JsonProperty("sequential_image_generation")
        protected String sequential;

        @JsonProperty("guidance_scale")
        protected Double guidance;

        protected Boolean stream;

        @JsonProperty("response_format")
        protected String format;

        protected String prompt;

        protected Integer seed;

        protected String model;

        // 单图String，多图String[]
        protected Object image;

        protected String size;

        public SeedreamMessage(SeedreamRequest seedreamRequest) throws Exception {
            this.check(seedreamRequest);
            this.sequentialOptions = seedreamRequest.getSequentialOptions();
            this.optimizeOptions = seedreamRequest.getOptimizeOptions();
            this.prompt = seedreamRequest.getMessage().getQuery();
            this.sequential = seedreamRequest.getSequential();
            this.guidance = seedreamRequest.getGuidance();
            this.format = seedreamRequest.getFormat();
            this.stream = seedreamRequest.getStream();
            this.model = seedreamRequest.getModel();
            this.seed = seedreamRequest.getSeed();
            this.size = seedreamRequest.getSize();
            // 加载参考图
            if (!CollectionUtils.isEmpty(seedreamRequest.getMediaContext())) {
                List<String> imageContext = new ArrayList<String>();
                for (MediaContext media : seedreamRequest.getMediaContext()) {
                    String mediaType = media.getType(seedreamRequest.getMimeType());
                    if (MediaContext.isInline(mediaType)) {
                        imageContext.add(seedreamRequest.getSeedMedia().getPrefix(MediaContext.pureType(mediaType)) + media.getData());
                    } else {
                        imageContext.add(media.getData());
                    }
                }
                this.image = imageContext.size() == 1 ? imageContext.getFirst() : imageContext;
            }
        }

        protected void check(SeedreamRequest seedreamRequest) throws Exception {
            int images = !CollectionUtils.isEmpty(seedreamRequest.getMediaContext()) ? CollectionUtils.size(seedreamRequest.getMediaContext()) : 0;
            Assert.isTrue(images <= seedreamRequest.getImages(), "The media context quantity is incorrect: " + seedreamRequest.getImages() + " ,please check the `images` parameter");
        }
    }

    @ConditionalOnProperty(name = "seedream.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderRouterInitConfig {

        @Value("${seedream.url:https://ark.cn-beijing.volces.com/api/v3/images/generations}")
        protected String url;

        @Bean(name = SeedreamRouter.NAME)
        @ConditionalOnMissingBean(name = SeedreamRouter.NAME)
        public SeedreamRouter seedreamRouter() throws Exception {
            SeedreamRouter seedRouter = new SeedreamRouter();
            BeanUtils.copyProperties(this, seedRouter);
            log.info("SeedreamRouter inited. url={}", seedRouter.getUrl());
            return seedRouter;
        }
    }
}
