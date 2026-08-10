package ai.open.right.workflow.flow.llm.provider.deepseek;

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
public class DeepSeekRouter extends OpenAiRouter {

    public static final String NAME = "DeepSeekRouter";

    // DeepSeek API URL
    protected String url;

    @Override
    public String url(OpenAiRequest request, LLMConfig llmConfig, String t) throws Exception {
        String url = MapUtils.getString(request.getMessage().getMetadata(), "__url", StringUtils.defaultIfEmpty(request.getUrl(), this.url));
        Assert.hasText(url, "Url can not be empty");
        return url;
    }

    @ConditionalOnProperty(name = "deepseek.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderRouterInitConfig {

        @Value("${deepseek.url:https://api.deepseek.com/chat/completions}")
        // DeepSeek API URL
        protected String url;

        @Bean(name = DeepSeekRouter.NAME)
        @ConditionalOnMissingBean(name = DeepSeekRouter.NAME)
        public DeepSeekRouter deepSeekRouter() throws Exception {
            DeepSeekRouter deepSeekRouter = new DeepSeekRouter();
            BeanUtils.copyProperties(this, deepSeekRouter);
            log.info("DeepSeekRouter inited. url={}", deepSeekRouter.getUrl());
            return deepSeekRouter;
        }
    }

}
