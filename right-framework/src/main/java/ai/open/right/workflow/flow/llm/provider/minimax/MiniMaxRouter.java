package ai.open.right.workflow.flow.llm.provider.minimax;

import ai.open.right.workflow.flow.llm.LLMFunCallRequest;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.provider.anthropic.AnthropicRequest;
import ai.open.right.workflow.flow.llm.provider.anthropic.AnthropicRouter;
import ai.open.right.workflow.flow.llm.store.history.History;
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

import java.util.HashMap;
import java.util.Map;

@Setter
@Getter
@Slf4j
public class MiniMaxRouter extends AnthropicRouter {

    public static final String NAME = "MiniMaxRouter";

    protected String url;

    @Override
    public String url(AnthropicRequest request, LLMConfig llmConfig, String t) throws Exception {
        String url = MapUtils.getString(request.getMessage().getMetadata(), "__url", StringUtils.defaultIfEmpty(request.getUrl(), this.url));
        Assert.hasText(url, "Url can not be empty");
        return url;
    }

    @Override
    public Object body(AnthropicRequest request) throws Exception {
        return new MiniMaxMessage(request);
    }

    public static class MiniMaxMessage extends AnthropicMessage {

        public MiniMaxMessage(AnthropicRequest anthropicRequest) throws Exception {
            super(anthropicRequest);
            this.thinking = MapUtils.getMap(anthropicRequest.getExtraBody(), ProviderRequestService.KEY_THINKING);
        }

        @Override
        protected void buildFunCallRequest(LLMFunCallRequest request) throws Exception {
            this.messages.add(new MiniMaxContent(request));
        }

        @Override
        protected void buildChat(History history) throws Exception {
            this.messages.add(new MiniMaxContent(history));
        }
    }

    public static class MiniMaxContent extends AnthropicContent {

        public MiniMaxContent(LLMFunCallRequest llmFunCallRequest) throws Exception {
            super(llmFunCallRequest.getCreated());
            this.role = AnthropicContent.ROLE_ASSISTANT;
            this.requestToolCallId = llmFunCallRequest.getId();
            Map<String, Object> request = this.buildFunCallRequest(llmFunCallRequest);
            if (!StringUtils.isEmpty(llmFunCallRequest.getReason())) {
                this.content = new Object[]{this.buildThinking(llmFunCallRequest.getReason()), request};
            } else {
                this.content = new Object[]{request};
            }
        }

        public MiniMaxContent(History history) throws Exception {
            super(history.getCreated());
            Assert.hasText(history.getContent(), "History content can not be empty");
            Assert.notNull(history.getRole(), "History role can not be empty");
            this.role = history.isRole(History.ROLE_USER) ? AnthropicContent.ROLE_USER : AnthropicContent.ROLE_ASSISTANT;
            Map<String, Object> request = this.buildHistory(history);
            if (!StringUtils.isEmpty(history.getReason())) {
                this.content = new Object[]{this.buildThinking(history.getReason()), request};
            } else {
                this.content = new Object[]{request};
            }
        }

        protected Map<String, Object> buildFunCallRequest(LLMFunCallRequest llmFunCallRequest) throws Exception {
            Map<String, Object> request = new HashMap<String, Object>();
            // 强制格式，保证模型间兼容
            request.put("input", this.buildJson(llmFunCallRequest.getArgs()));
            request.put("name", llmFunCallRequest.getName());
            request.put("type", AnthropicContent.TOOL_USE);
            request.put("id", llmFunCallRequest.getId());
            return request;
        }

        protected Map<String, Object> buildThinking(String thinking) throws Exception {
            Map<String, Object> request = new HashMap<String, Object>();
            request.put(ProviderRequestService.KEY_THINKING, thinking);
            request.put("type", ProviderRequestService.KEY_THINKING);
            return request;
        }

        protected Map<String, Object> buildHistory(History history) throws Exception {
            Map<String, Object> request = new HashMap<String, Object>();
            request.put("text", history.getContent());
            request.put("type", "text");
            return request;
        }
    }

    @ConditionalOnProperty(name = "minimax.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderRouterInitConfig {

        @Value("${minimax.url:https://api.minimaxi.com/anthropic/v1/messages}")
        protected String url;

        @Bean(name = MiniMaxRouter.NAME)
        @ConditionalOnMissingBean(name = MiniMaxRouter.NAME)
        public MiniMaxRouter miniMaxRouter() throws Exception {
            MiniMaxRouter miniMaxRouter = new MiniMaxRouter();
            BeanUtils.copyProperties(this, miniMaxRouter);
            log.info("MiniMaxRouter inited. url={}", miniMaxRouter.getUrl());
            return miniMaxRouter;
        }
    }
}
