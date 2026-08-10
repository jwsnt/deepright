package ai.open.right.workflow.flow.llm.provider.kimi;

import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRouter;
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
public class KimiRouter extends OpenAiRouter {

    public static final String NAME = "KimiRouter";

    // Kimi API URL
    protected String url;

    @Override
    public String url(OpenAiRequest request, LLMConfig llmConfig, String t) throws Exception {
        String url = MapUtils.getString(request.getMessage().getMetadata(), "__url", StringUtils.defaultIfEmpty(request.getUrl(), this.url));
        Assert.hasText(url, "Url can not be empty");
        return url;
    }

    @ConditionalOnProperty(name = "kimi.enable", havingValue = "true", matchIfMissing = false)
    @Setter
    @Getter
    @Configuration
    public static class InitConfig extends ProviderRouterInitConfig {

        @Value("${kimi.url:https://api.moonshot.cn/v1/chat/completions}")
        // Kimi API URL
        protected String url;

        @Bean(name = KimiRouter.NAME)
        @ConditionalOnMissingBean(name = KimiRouter.NAME)
        public KimiRouter kimiRouter() throws Exception {
            KimiRouter kimiRouter = new KimiRouter();
            BeanUtils.copyProperties(this, kimiRouter);
            log.info("KimiRouter inited. url={}", kimiRouter.getUrl());
            return kimiRouter;
        }
    }
}
