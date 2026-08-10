package ai.open.right.workflow.flow.llm.provider.coze;

import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderReader;
import ai.open.right.workflow.flow.llm.provider.ProviderReaderConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRouter;
import ai.open.right.workflow.flow.llm.store.history.History;
import com.fasterxml.jackson.annotation.JsonProperty;
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

import java.util.ArrayList;
import java.util.List;

@Slf4j
@Setter
@Getter
public class CozeRouter extends ProviderRouter<CozeRequest> {

    public static final String NAME = "CozeRouter";

    // Coze API URL
    protected String url;

    @Override
    protected void reConfig(CozeRequest request, LLMConfig llmConfig, HttpPost httpPost) throws Exception {
        super.reConfig(request, llmConfig, httpPost);
        httpPost.addHeader("Authorization", request.getToken());
    }

    @Override
    protected CozeReader reader(CozeRequest request, LLMConfig llmConfig, LLMCallback llmCallback) throws Exception {
        return new CozeReader(ProviderReaderConfig.<CozeRequest>builder()
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
    protected String url(CozeRequest request, LLMConfig llmConfig, String t) throws Exception {
        String url = MapUtils.getString(request.getMessage().getMetadata(), "__url", StringUtils.defaultIfEmpty(request.getUrl(), this.url));
        Assert.hasText(url, "Url can not be empty");
        return url;
    }

    @Override
    protected Object body(CozeRequest cozeRequest) throws Exception {
        return new CozeMessage(cozeRequest);
    }

    @Getter
    public static class CozeMessage {

        @JsonProperty("chat_history")
        protected List<CozeHistory> histories;

        @JsonProperty("conversation_id")
        protected final String conversation;

        @JsonProperty("stream")
        protected final Boolean stream;

        @JsonProperty("query")
        protected final String query;

        @JsonProperty("bot_id")
        protected final String botId;

        @JsonProperty("user")
        protected final String user;

        public CozeMessage(CozeRequest cozeRequest) {
            this.user = cozeRequest.getMessage().getUserContext().getDevice();
            this.conversation = cozeRequest.getMessage().getConversation();
            this.query = cozeRequest.getMessage().getQuery();
            this.stream = cozeRequest.getStream();
            this.botId = cozeRequest.getBotId();
            if (cozeRequest.getMessage().hasHistory()) {
                this.histories = new ArrayList<CozeHistory>();
                for (History each : cozeRequest.getMessage().getHistories()) {
                    this.histories.add(new CozeHistory(each));
                }
            }
        }
    }

    @Getter
    public static class CozeHistory {

        protected static final String ROLE_ASSISTANT = "assistant";

        protected static final String ROLE_USER = "user";

        protected static final String TYPE_ANSWER = "answer";

        protected static final String TYPE_QUERY = "query";

        @JsonProperty("content_type")
        protected final String contentType = "text";

        protected final String content;

        protected final String role;

        protected final String type;

        public CozeHistory(History history) {
            this.role = history.isRole(History.ROLE_USER) ? CozeHistory.ROLE_USER : CozeHistory.ROLE_ASSISTANT;
            this.type = history.isType(History.TYPE_ANSWER) ? CozeHistory.TYPE_ANSWER : CozeHistory.TYPE_QUERY;
            this.content = history.getContent();
        }
    }

    @ConditionalOnProperty(name = "coze.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderRouterInitConfig {

        @Value("${coze.url:https://api.coze.cn/open_api/v2/chat}")
        // Coze API URL
        protected String url;

        @Bean(name = CozeRouter.NAME)
        @ConditionalOnMissingBean(name = CozeRouter.NAME)
        public CozeRouter cozeRouter() throws Exception {
            CozeRouter cozeRouter = new CozeRouter();
            BeanUtils.copyProperties(this, cozeRouter);
            log.info("CozeRouter inited. url={}", cozeRouter.getUrl());
            return cozeRouter;
        }
    }
}
