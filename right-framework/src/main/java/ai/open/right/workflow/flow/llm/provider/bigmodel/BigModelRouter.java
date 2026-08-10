package ai.open.right.workflow.flow.llm.provider.bigmodel;

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
public class BigModelRouter extends OpenAiRouter {

    public static final String NAME = "BigModelRouter";

    // BigModel API URL
    protected String url;

    @Override
    public String url(OpenAiRequest request, LLMConfig llmConfig, String t) throws Exception {
        String url = MapUtils.getString(request.getMessage().getMetadata(), "__url", StringUtils.defaultIfEmpty(request.getUrl(), this.url));
        Assert.hasText(url, "Url can not be empty");
        return url;
    }

    @ConditionalOnProperty(name = "bigmodel.enable", havingValue = "true", matchIfMissing = false)
    @Setter
    @Getter
    @Configuration
    public static class InitConfig extends ProviderRouterInitConfig {

        @Value("${bigModel.url:https://open.bigmodel.cn/api/paas/v4/chat/completions}")
        // Kimi API URL
        protected String url;

        @Bean(name = BigModelRouter.NAME)
        @ConditionalOnMissingBean(name = BigModelRouter.NAME)
        public BigModelRouter bigModelRouter() throws Exception {
            BigModelRouter bigModelRouter = new BigModelRouter();
            BeanUtils.copyProperties(this, bigModelRouter);
            log.info("BigModelRouter inited. url={}", bigModelRouter.getUrl());
            return bigModelRouter;
        }
    }
}
