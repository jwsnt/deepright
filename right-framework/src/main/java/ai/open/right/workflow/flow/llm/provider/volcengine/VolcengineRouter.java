package ai.open.right.workflow.flow.llm.provider.volcengine;

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
public class VolcengineRouter extends OpenAiRouter {

    public static final String NAME = "VolcengineRouter";

    // 火山引擎API URL
    protected String url;

    @Override
    public String url(OpenAiRequest request, LLMConfig llmConfig, String t) throws Exception {
        String url = MapUtils.getString(request.getMessage().getMetadata(), "__url", StringUtils.defaultIfEmpty(request.getUrl(), this.url));
        Assert.hasText(url, "Url can not be empty");
        return url;
    }

    @ConditionalOnProperty(name = "volcengine.enable", havingValue = "true", matchIfMissing = false)
    @Setter
    @Getter
    @Configuration
    public static class InitConfig extends ProviderRouterInitConfig {

        @Value("${volcengine.url:https://ark.cn-beijing.volces.com/api/v3/chat/completions}")
        // 火山引擎API URL
        protected String url;

        @Bean(name = VolcengineRouter.NAME)
        @ConditionalOnMissingBean(name = VolcengineRouter.NAME)
        public VolcengineRouter volcengineRouter() throws Exception {
            VolcengineRouter volcengineRouter = new VolcengineRouter();
            BeanUtils.copyProperties(this, volcengineRouter);
            log.info("VolcengineRouter inited. url={}", volcengineRouter.getUrl());
            return volcengineRouter;
        }
    }
}