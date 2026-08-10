package ai.open.right.workflow.flow.llm.provider.xiaomi;

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
public class XiaomiRouter extends OpenAiRouter {

    public static final String NAME = "XiaomiRouter";

    // Xiaomi API URL
    protected String url;

    @Override
    public String url(OpenAiRequest request, LLMConfig llmConfig, String t) throws Exception {
        String url = MapUtils.getString(request.getMessage().getMetadata(), "__url", StringUtils.defaultIfEmpty(request.getUrl(), this.url));
        Assert.hasText(url, "Url can not be empty");
        return url;
    }

    @ConditionalOnProperty(name = "xiaomi.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderRouterInitConfig {

        @Value("${xiaomi.url:https://api.xiaomimimo.com/v1}")
        // Xiaomi API URL
        protected String url;

        @Bean(name = XiaomiRouter.NAME)
        @ConditionalOnMissingBean(name = XiaomiRouter.NAME)
        public XiaomiRouter xiaomiRouter() throws Exception {
            XiaomiRouter xiaomiRouter = new XiaomiRouter();
            BeanUtils.copyProperties(this, xiaomiRouter);
            log.info("XiaomiRouter inited. url={}", xiaomiRouter.getUrl());
            return xiaomiRouter;
        }
    }

}
