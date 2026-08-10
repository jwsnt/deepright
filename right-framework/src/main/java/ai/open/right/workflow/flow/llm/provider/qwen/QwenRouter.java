package ai.open.right.workflow.flow.llm.provider.qwen;

import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRouter;
import com.fasterxml.jackson.annotation.JsonProperty;
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
public class QwenRouter extends OpenAiRouter {

    public static final String NAME = "QwenRouter";

    // Qwen API URL
    protected String url;

    @Override
    public String url(OpenAiRequest request, LLMConfig llmConfig, String t) throws Exception {
        String url = MapUtils.getString(request.getMessage().getMetadata(), "__url", StringUtils.defaultIfEmpty(request.getUrl(), this.url));
        Assert.hasText(url, "Url can not be empty");
        return url;
    }

    @Override
    public Object body(OpenAiRequest request) throws Exception {
        return new QwenMessage(request);
    }

    @Getter
    public static class QwenMessage extends OpenAiMessage {

        @JsonProperty("enable_thinking")
        protected Boolean enableThinking;

        public QwenMessage(OpenAiRequest openAiRequest) throws Exception {
            super(openAiRequest);
            this.enableThinking = MapUtils.getBoolean(openAiRequest.getExtraBody(), "enable_thinking");
        }
    }

    @ConditionalOnProperty(name = "qwen.enable", havingValue = "true", matchIfMissing = false)
    @Setter
    @Getter
    @Configuration
    public static class InitConfig extends ProviderRouterInitConfig {

        @Value("${qwen.url:https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions}")
        // Qwen API URL
        protected String url;

        @Bean(name = QwenRouter.NAME)
        @ConditionalOnMissingBean(name = QwenRouter.NAME)
        public QwenRouter qwenRouter() throws Exception {
            QwenRouter qwenRouter = new QwenRouter();
            BeanUtils.copyProperties(this, qwenRouter);
            log.info("QwenRouter inited. url={}", qwenRouter.getUrl());
            return qwenRouter;
        }
    }
}
