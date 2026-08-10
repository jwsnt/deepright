package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRouter;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
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

@Slf4j
@Setter
@Getter
public class GeminiRouter extends GoogleRouter {

    public static final String NAME = "GeminiRouter";

    public static final String MODEL = "model";

    // Gemini Stream API URL
    protected String urlStream;

    // Gemini非流式API URL
    protected String urlOnce;

    @Override
    protected String url(GoogleRequest request, LLMConfig llmConfig, String t) throws Exception {
        Assert.hasText(request.getModel(), "Gemini model can not be empty");
        String url = null;
        if (ProviderRouter.URL_STREAM.equals(t)) {
            // 精确匹配
            url = MapUtils.getString(request.getMessage().getMetadata(), "__url", StringUtils.defaultIfEmpty(request.getUrl(), this.urlStream)).replace("#model", request.getModel());
        } else {
            url = MapUtils.getString(request.getMessage().getMetadata(), "__url", StringUtils.defaultIfEmpty(request.getUrl(), this.urlOnce)).replace("#model", request.getModel());
        }
        Assert.hasText(url, "Url can not be empty");
        return url;
    }

    @Override
    protected void addHeader(GoogleRequest request, HttpPost httpPost) throws Exception {
        httpPost.addHeader("x-goog-api-key", request.getToken());
    }

    @ConditionalOnProperty(name = "gemini.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderRouterInitConfig {

        @Value("${gemini.url.stream:https://generativelanguage.googleapis.com/v1beta/models/#model:streamGenerateContent}")
        // Gemini Stream API URL
        protected String urlStream;

        @Value("${gemini.url.once:https://generativelanguage.googleapis.com/v1beta/models/#model:generateContent}")
        // Gemini非流式API URL
        protected String urlOnce;

        @Bean(name = GeminiRouter.NAME)
        @ConditionalOnMissingBean(name = GeminiRouter.NAME)
        public GeminiRouter geminiRouter() throws Exception {
            GeminiRouter geminiRouter = new GeminiRouter();
            BeanUtils.copyProperties(this, geminiRouter);
            log.info("GeminiRouter inited. urlStream={}, urlOnce={}", geminiRouter.getUrlStream(), geminiRouter.getUrlOnce());
            return geminiRouter;
        }
    }
}
