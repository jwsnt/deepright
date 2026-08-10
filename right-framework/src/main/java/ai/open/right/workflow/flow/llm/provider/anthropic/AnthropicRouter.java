package ai.open.right.workflow.flow.llm.provider.anthropic;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.LLMFunCallData;
import ai.open.right.workflow.flow.llm.LLMFunCallRequest;
import ai.open.right.workflow.flow.llm.LLMFunCallResponse;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.*;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import com.fasterxml.jackson.annotation.JsonIgnore;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.google.common.collect.ImmutableMap;
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

import java.util.*;

@Setter
@Getter
@Slf4j
public class AnthropicRouter extends ProviderRouter<AnthropicRequest> {

    public static final Short MAX_TOKEN = Short.MAX_VALUE;

    public static final String NAME = "AnthropicRouter";

    protected String url;

    @Override
    public void reConfig(AnthropicRequest request, LLMConfig llmConfig, HttpPost httpPost) throws Exception {
        super.reConfig(request, llmConfig, httpPost);
        httpPost.addHeader("X-Api-Key", request.getToken());
    }

    @Override
    protected AnthropicReader reader(AnthropicRequest request, LLMConfig llmConfig, LLMCallback llmCallback) throws Exception {
        return new AnthropicReader(ProviderReaderConfig.<AnthropicRequest>builder()
                .buffer(llmConfig.hasNetworkBuffer() ? llmConfig.getNetworkBuffer() : this.buffer)
                .eventListenerService(this.eventListenerService)
                .notifierService(this.notifierService)
                .extension(ProviderReader.EXTENSION)
                .timeout(this.queueTimeout)
                .llmCallback(llmCallback)
                .capacity(this.capacity)
                .discard(this.discard)
                .queue(this.queue)
                .request(request)
                .build().check());
    }

    @Override
    public String url(AnthropicRequest request, LLMConfig llmConfig, String t) throws Exception {
        String url = MapUtils.getString(request.getMessage().getMetadata(), "__url", StringUtils.defaultIfEmpty(request.getUrl(), this.url));
        Assert.hasText(url, "Url can not be empty");
        return url;
    }

    @Override
    public Object body(AnthropicRequest request) throws Exception {
        return new AnthropicMessage(request);
    }

    @Getter
    public static class AnthropicMessage {

        @JsonIgnore
        protected Map<String, LLMFunCallResponse> funCallResponses;

        @JsonProperty("output_config")
        protected Map<String, Object> responseFormat;

        @JsonProperty("messages")
        protected List<AnthropicContent> messages;

        @JsonProperty("thinking")
        protected Map<String, Object> thinking;

        @JsonProperty("tools")
        protected List<AnthropicTool> tools;

        @JsonProperty("temperature")
        protected Double temperature;

        @JsonProperty("max_tokens")
        protected Integer maxTokens;

        @JsonProperty("stream")
        protected Boolean stream;

        @JsonProperty("system")
        protected Object system;

        @JsonProperty("model")
        protected String model;

        // 采样时考虑的词元累计概率上限
        @JsonProperty("top_p")
        protected Double topP;

        public AnthropicMessage(AnthropicRequest anthropicRequest) throws Exception {
            // 强制MAX TOKEN
            this.maxTokens = anthropicRequest.getMaxTokens() != null ? anthropicRequest.getMaxTokens() : AnthropicRouter.MAX_TOKEN;
            this.responseFormat = anthropicRequest.getResponseFormat();
            this.temperature = anthropicRequest.getTemperature();
            this.messages = new ArrayList<AnthropicContent>();
            this.thinking = anthropicRequest.getThinking();
            this.stream = anthropicRequest.getStream();
            this.system = anthropicRequest.getPrompt();
            this.model = anthropicRequest.getModel();
            this.topP = anthropicRequest.getTopP();
            this.buildSystemPrompt(anthropicRequest);
            ///////////////////////////
            // System Prompt(top-level) -> History/FunCall/MimeQuery(sorted by created)
            //////////////////////////
            if (anthropicRequest.getMessage().hasHistory()) {
                this.buildHistories(anthropicRequest.getMessage().getHistories());
            }
            this.buildFunCallRequest(anthropicRequest);
            this.buildUserQuery(anthropicRequest);
            this.messages.sort(Comparator.comparing(AnthropicContent::getCreated, Comparator.nullsFirst(Long::compareTo)));
            this.buildFunCallResponse(anthropicRequest);
            if (anthropicRequest.hasFunCall()) {
                this.tools = new ArrayList<AnthropicTool>();
                for (ProviderFunCall each : anthropicRequest.getFunCalls()) {
                    this.tools.add(new AnthropicTool(each));
                }
            }
        }


        protected void buildSystemPrompt(AnthropicRequest anthropicRequest) throws Exception {
            Map<String, Object> cacheControl = anthropicRequest.getCacheControl();
            if (!MapUtils.isEmpty(cacheControl)) {
                this.system = Collections.singletonList(ImmutableMap.of(
                        "type", "text",
                        "text", anthropicRequest.getPrompt(),
                        "cache_control", cacheControl
                ));
            } else {
                this.system = anthropicRequest.getPrompt();
            }
        }

        protected void buildHistories(List<History> histories) throws Exception {
            if (!CollectionUtils.isEmpty(histories)) {
                for (History each : histories) {
                    try {
                        if (each.isFunction(History.FUN_CHAT)) {
                            this.buildChat(each);
                        } else {
                            // Fun Call
                            if (each.isType(History.TYPE_QUERY)) {
                                this.buildFunCallRequest(JsonUtils.read(each.getContent(), ProviderFunCallRequest.class));
                            }
                            if (each.isType(History.TYPE_ANSWER)) {
                                this.buildFunCallResponse(JsonUtils.read(each.getContent(), ProviderFunCallResponse.class));
                            }
                        }
                    } catch (Exception ex) {
                        if (log.isWarnEnabled()) {
                            log.warn(ex.getMessage(), ex);
                        }
                    }
                }
            }
        }

        protected void buildUserQuery(AnthropicRequest anthropicRequest) throws Exception {
            // Mime Query
            if (anthropicRequest.hasMimeContext()) {
                this.messages.add(new AnthropicContent(anthropicRequest.getMediaContext(), anthropicRequest.getAnthropicMedia(), anthropicRequest.getMimeType(), anthropicRequest.getMessage().getQuery(), anthropicRequest.getMessage().getCreated()));
            } else {
                // Text Query
                this.messages.add(new AnthropicContent(AnthropicContent.ROLE_USER, anthropicRequest.getMessage().getQuery(), anthropicRequest.getMessage().getCreated()));
            }
        }

        protected void buildFunCallResponse(AnthropicRequest anthropicRequest) throws Exception {
            if (!MapUtils.isEmpty(this.funCallResponses)) {
                ListIterator<AnthropicContent> iterator = this.messages.listIterator();
                while (iterator.hasNext()) {
                    AnthropicContent element = iterator.next();
                    if (!StringUtils.isEmpty(element.getRequestToolCallId())) {
                        LLMFunCallResponse funCallResponse = this.funCallResponses.get(element.getRequestToolCallId());
                        if (funCallResponse != null) {
                            iterator.add(new AnthropicContent(funCallResponse));
                        }
                    }
                }
            }
        }

        protected void buildFunCallRequest(AnthropicRequest anthropicRequest) throws Exception {
            if (anthropicRequest.hasFunCallData()) {
                this.funCallResponses = new HashMap<String, LLMFunCallResponse>();
                LLMFunCallData funCallData = anthropicRequest.getFunCallData();
                for (int index = 0; index < funCallData.getRequests().size() && index < funCallData.getResponses().size(); index++) {
                    String funCallId = this.buildFunCallId(funCallData.getRequests().get(index), funCallData.getResponses().get(index), index);
                    LLMFunCallRequest funCallRequest = this.normalizeFunCallRequest(funCallData.getRequests().get(index), funCallId);
                    LLMFunCallResponse funCallResponse = this.normalizeFunCallResponse(funCallData.getResponses().get(index), funCallId);
                    this.funCallResponses.put(funCallId, funCallResponse);
                    this.buildFunCallRequest(funCallRequest);
                }
            }
        }

        protected String buildFunCallId(LLMFunCallRequest funCallRequest, LLMFunCallResponse funCallResponse, int index) {
            String funCallId = StringUtils.defaultIfEmpty(funCallRequest.getId(), funCallResponse.getId());
            return StringUtils.defaultIfEmpty(funCallId, "fun_call_" + index);
        }

        protected LLMFunCallRequest normalizeFunCallRequest(LLMFunCallRequest funCallRequest, String funCallId) {
            return ProviderFunCallRequest.builder()
                    .created(funCallRequest.getCreated())
                    .metadata(funCallRequest.getMetadata())
                    .refer(funCallRequest.getRefer())
                    .reason(funCallRequest.getReason())
                    .args(funCallRequest.getArgs())
                    .name(funCallRequest.getName())
                    .id(funCallId)
                    .build();
        }

        protected LLMFunCallResponse normalizeFunCallResponse(LLMFunCallResponse funCallResponse, String funCallId) {
            return ProviderFunCallResponse.builder()
                    .created(funCallResponse.getCreated())
                    .metadata(funCallResponse.getMetadata())
                    .response(funCallResponse.getResponse())
                    .name(funCallResponse.getName())
                    .id(funCallId)
                    .build();
        }

        protected void buildFunCallResponse(LLMFunCallResponse response) throws Exception {
            this.messages.add(new AnthropicContent(response));
        }

        protected void buildFunCallRequest(LLMFunCallRequest request) throws Exception {
            this.messages.add(new AnthropicContent(request));
        }

        protected void buildChat(History history) throws Exception {
            this.messages.add(new AnthropicContent(history));
        }
    }

    @Getter
    public static class AnthropicContent {

        protected static final String ROLE_ASSISTANT = "assistant";

        protected static final String ROLE_USER = "user";

        protected static final String TOOL_RESULT = "tool_result";

        protected static final String TOOL_USE = "tool_use";

        @JsonIgnore
        protected final Long created;

        @JsonIgnore
        protected String requestToolCallId;

        protected Object content;

        protected String role;

        public AnthropicContent(List<MediaContext> mediaContext, AnthropicMedia anthropicMedia, String type, String part, Long created) throws Exception {
            List<Map<String, Object>> content = new ArrayList<Map<String, Object>>();
            content.add(ImmutableMap.of("type", "text", "text", part));
            for (MediaContext media : mediaContext) {
                String mediaType = media.getType(type);
                // https://platform.claude.com/docs/zh-CN/build-with-claude/vision
                if (MediaContext.isInline(mediaType)) {
                    // Inline
                    content.add(ImmutableMap.of("type", anthropicMedia.getType(mediaType), "source", ImmutableMap.of("type", "base64", "media_type", MediaContext.pureType(mediaType), "data", media.getData())));
                } else {
                    content.add(ImmutableMap.of("type", anthropicMedia.getType(mediaType), "source", ImmutableMap.of("type", "url", "url", media.getData())));
                }
            }
            this.role = AnthropicContent.ROLE_USER;
            this.content = content;
            this.created = created;
        }

        public AnthropicContent(LLMFunCallResponse llmFunCallResponse) throws Exception {
            Map<String, Object> response = new HashMap<String, Object>();
            response.put("content", llmFunCallResponse.getResponse());
            response.put("tool_use_id", llmFunCallResponse.getId());
            response.put("type", AnthropicContent.TOOL_RESULT);
            this.created = llmFunCallResponse.getCreated();
            this.role = AnthropicContent.ROLE_USER;
            this.content = new Object[]{response};
        }

        public AnthropicContent(LLMFunCallRequest llmFunCallRequest) throws Exception {
            Map<String, Object> request = new HashMap<String, Object>();
            // 强制格式，保证模型间兼容
            request.put("input", this.buildJson(llmFunCallRequest.getArgs()));
            request.put("name", llmFunCallRequest.getName());
            request.put("type", AnthropicContent.TOOL_USE);
            request.put("id", llmFunCallRequest.getId());
            this.requestToolCallId = llmFunCallRequest.getId();
            this.created = llmFunCallRequest.getCreated();
            this.role = AnthropicContent.ROLE_ASSISTANT;
            this.content = new Object[]{request};
        }

        public AnthropicContent(String role, String content, Long created) throws Exception {
            this.created = created;
            this.content = content;
            this.role = role;
            Assert.notNull(this.content, "Open AI content can not be empty");
            Assert.hasText(this.role, "Role can not be empty");
        }

        public AnthropicContent(History history) throws Exception {
            this(history.isRole(History.ROLE_USER) ? AnthropicContent.ROLE_USER : AnthropicContent.ROLE_ASSISTANT, history.getContent(), history.getCreated());
            Assert.hasText(history.getContent(), "History content can not be empty");
            Assert.notNull(history.getRole(), "History role can not be empty");
        }

        public AnthropicContent(Long created) throws Exception {
            this.created = created;
        }

        @JsonIgnore
        public String getRequestToolCallId() {
            return this.requestToolCallId;
        }

        protected Map<String, Object> buildJson(Object args) throws Exception {
            try {
                return Map.class.isAssignableFrom(args.getClass()) ? Map.class.cast(args) : JsonUtils.transfer(args, Map.class);
            } catch (Exception e) {
                log.warn(e.getMessage(), e);
                // 如果无法转换时的兜底
                return ImmutableMap.of("args", JsonUtils.write(args));
            }
        }
    }

    @Getter
    public static class AnthropicTool {

        protected final Map<String, Object> input_schema = new HashMap<String, Object>();

        protected final String description;

        protected final String name;

        public AnthropicTool(ProviderFunCall providerFunCall) throws Exception {
            this.input_schema.put("properties", providerFunCall.getProperties());
            this.input_schema.put("type", "object");
            this.description = providerFunCall.getDescription();
            this.name = providerFunCall.getName();
        }
    }

    @ConditionalOnProperty(name = "anthropic.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderRouterInitConfig {

        @Value("${anthropic.url:}")
        protected String url;

        @Bean(name = AnthropicRouter.NAME)
        @ConditionalOnMissingBean(name = AnthropicRouter.NAME)
        public AnthropicRouter anthropicRouter() throws Exception {
            AnthropicRouter anthropicRouter = new AnthropicRouter();
            BeanUtils.copyProperties(this, anthropicRouter);
            log.info("AnthropicRouter inited. url={}", anthropicRouter.getUrl());
            return anthropicRouter;
        }
    }
}
