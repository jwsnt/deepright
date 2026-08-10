package ai.open.right.workflow.flow.llm.provider.openai;

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
import com.fasterxml.jackson.core.JsonParseException;
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
public class OpenAiRouter extends ProviderRouter<OpenAiRequest> {

    public static final String NAME = "OpenAiRouter";

    public static final String KEY_URL = "url";

    // Open API URL
    protected String url;

    @Override
    protected void reConfig(OpenAiRequest request, LLMConfig llmConfig, HttpPost httpPost) throws Exception {
        super.reConfig(request, llmConfig, httpPost);
        httpPost.addHeader("Authorization", request.getToken());
    }

    @Override
    protected OpenAiReader reader(OpenAiRequest request, LLMConfig llmConfig, LLMCallback llmCallback) throws Exception {
        return new OpenAiReader(ProviderReaderConfig.<OpenAiRequest>builder()
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
    public String url(OpenAiRequest request, LLMConfig llmConfig, String t) throws Exception {
        // 先从Metadata获取
        String url = MapUtils.getString(request.getMessage().getMetadata(), "__url", StringUtils.defaultIfEmpty(request.getUrl(), StringUtils.defaultIfEmpty(String.class.cast(llmConfig.getAdditional().get(OpenAiRouter.KEY_URL)), this.url)));
        Assert.hasText(url, "Url can not be empty");
        return url;
    }

    @Override
    public Object body(OpenAiRequest request) throws Exception {
        return new OpenAiMessage(request);
    }

    @Getter
    public static class OpenAiMessage {

        @JsonIgnore
        protected Map<String, LLMFunCallResponse> funCallResponses;

        @JsonProperty("response_format")
        protected Map<String, Object> responseFormat;

        @JsonProperty("reasoning")
        protected Map<String, Object> reasoning;

        @JsonProperty("messages")
        protected List<OpenAiContent> messages;

        @JsonProperty("thinking")
        protected Map<String, Object> thinking;

        @JsonProperty("parallel_tool_calls")
        protected Boolean parallelToolCalls;

        @JsonProperty("frequency_penalty")
        protected Double frequencyPenalty;

        @JsonProperty("presence_penalty")
        protected Double presencePenalty;

        @JsonProperty("reasoning_effort")
        protected String reasoningEffort;

        @JsonProperty("tools")
        protected List<OpenAiTool> tools;

        @JsonProperty("temperature")
        protected Double temperature;

        @JsonProperty("max_tokens")
        protected Integer maxTokens;

        @JsonProperty("stream")
        protected Boolean stream;

        @JsonProperty("model")
        protected String model;

        // 采样时考虑的词元累计概率上限
        @JsonProperty("top_p")
        protected Double topP;

        public OpenAiMessage(OpenAiRequest openAiRequest) throws Exception {
            this.messages = new ArrayList<OpenAiContent>();
            // 仅在存在FunCall时读取配置
            this.parallelToolCalls = !CollectionUtils.isEmpty(openAiRequest.getFunCalls()) ? MapUtils.getBoolean(openAiRequest.getExtraBody(), OpenAiRequestService.KEY_PARALLEL_TOOL) : null;
            this.reasoning = MapUtils.getMap(openAiRequest.getExtraBody(), OpenAiRequestService.KEY_REASONING);
            this.thinking = MapUtils.getMap(openAiRequest.getExtraBody(), ProviderRequestService.KEY_THINKING);
            this.frequencyPenalty = openAiRequest.getFrequencyPenalty();
            this.presencePenalty = openAiRequest.getPresencePenalty();
            this.reasoningEffort = openAiRequest.getReasoningEffort();
            this.responseFormat = openAiRequest.getResponseFormat();
            this.temperature = openAiRequest.getTemperature();
            this.maxTokens = openAiRequest.getMaxTokens();
            this.stream = openAiRequest.getStream();
            this.model = openAiRequest.getModel();
            this.topP = openAiRequest.getTopP();
            this.tools = null;
            ///////////////////////////
            // History -> FunCall -> System Prompt -> Mime/Query
            //////////////////////////
            // History
            this.buildFunCallRequest(openAiRequest);
            this.buildHistories(openAiRequest.getMessage().getHistories());
            this.buildUserQuery(openAiRequest);
            this.messages.sort(Comparator.comparing(OpenAiContent::getCreated, Comparator.nullsFirst(Long::compareTo)));
            this.buildFunCallResponse(openAiRequest);
            if (openAiRequest.hasFunCall()) {
                this.tools = new ArrayList<OpenAiTool>();
                for (ProviderFunCall each : openAiRequest.getFunCalls()) {
                    this.tools.add(new OpenAiTool(each));
                }
            }
        }

        protected void buildFunCallResponse(OpenAiRequest openAiRequest) throws Exception {
            if (!MapUtils.isEmpty(this.funCallResponses)) {
                ListIterator<OpenAiContent> iterator = this.messages.listIterator();
                while (iterator.hasNext()) {
                    OpenAiContent element = iterator.next();
                    if (!StringUtils.isEmpty(element.getRequestToolCallId())) {
                        LLMFunCallResponse funCallResponse = this.funCallResponses.get(element.getRequestToolCallId());
                        if (funCallResponse != null) {
                            iterator.add(new OpenAiContent(funCallResponse));
                        }
                    }
                }
            }
        }

        protected void buildFunCallRequest(OpenAiRequest openAiRequest) throws Exception {
            if (openAiRequest.hasFunCallData()) {
                this.funCallResponses = new HashMap<String, LLMFunCallResponse>();
                LLMFunCallData funCallData = openAiRequest.getFunCallData();
                for (int index = 0; index < funCallData.getRequests().size() && index < funCallData.getResponses().size(); index++) {
                    LLMFunCallRequest funCallRequest = funCallData.getRequests().get(index);
                    LLMFunCallResponse funCallResponse = funCallData.getResponses().get(index);
                    String funCallId = StringUtils.defaultIfEmpty(funCallRequest.getId(), funCallResponse.getId());
                    funCallId = StringUtils.defaultIfEmpty(funCallId, "fun_call_" + index);
                    funCallRequest.setId(funCallId);
                    funCallResponse.setId(funCallId);
                    this.funCallResponses.put(funCallId, funCallResponse);
                    this.messages.add(new OpenAiContent(funCallRequest));
                }
            }
        }

        protected void buildUserQuery(OpenAiRequest openAiRequest) throws Exception {
            if (!StringUtils.isEmpty(openAiRequest.getPrompt())) {
                this.messages.add(new OpenAiContent(OpenAiContent.ROLE_SYSTEM, openAiRequest.getPrompt(), openAiRequest.getMessage().getCreated()));
            }
            // Mime Query
            if (openAiRequest.hasMimeContext()) {
                this.messages.add(new OpenAiContent(openAiRequest.getMediaContext(), openAiRequest.getOpenAiMedia(), openAiRequest.getMimeType(), openAiRequest.getMessage().getQuery(), openAiRequest.getMessage().getCreated()));
            } else {
                // Text Query
                this.messages.add(new OpenAiContent(OpenAiContent.ROLE_USER, openAiRequest.getMessage().getQuery(), openAiRequest.getMessage().getCreated()));
            }
        }

        protected void buildFunCallResponse(LLMFunCallResponse response) throws Exception {
            this.messages.add(new OpenAiContent(response));
        }

        protected void buildFunCallRequest(LLMFunCallRequest request) throws Exception {
            this.messages.add(new OpenAiContent(request));
        }

        protected void buildHistories(List<History> histories) throws Exception {
            if (!CollectionUtils.isEmpty(histories)) {
                ProviderFunCallRequest lastRequest = null;
                for (History each : histories) {
                    try {
                        if (each.isFunction(History.FUN_CHAT)) {
                            this.messages.add(new OpenAiContent(each));
                        } else {
                            // Fun Call
                            if (each.isType(History.TYPE_QUERY)) {
                                this.buildFunCallRequest((lastRequest = JsonUtils.read(each.getContent(), ProviderFunCallRequest.class)));
                            }
                            if (each.isType(History.TYPE_ANSWER)) {
                                try {
                                    this.buildFunCallResponse(JsonUtils.read(each.getContent(), ProviderFunCallResponse.class));
                                } catch (JsonParseException e) {
                                    ProviderFunCallResponse funCallResponse = new ProviderFunCallResponse();
                                    if (lastRequest != null) {
                                        if (log.isInfoEnabled()) {
                                            log.info("The function call response will be built based on the previous request={}", lastRequest);
                                        }
                                        // Fallback需要保证在lastRequest之后
                                        funCallResponse.setCreated(lastRequest.getCreated() + 1);
                                        funCallResponse.setModel(lastRequest.getModel());
                                        funCallResponse.setName(lastRequest.getName());
                                        funCallResponse.setResponse(each.getContent());
                                        funCallResponse.setId(lastRequest.getId());
                                        this.buildFunCallResponse(funCallResponse);
                                    } else {
                                        throw e;
                                    }
                                }
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
    }

    @Getter
    public static class OpenAiContent {

        protected static final String ROLE_ASSISTANT = "assistant";

        protected static final String ROLE_SYSTEM = "system";

        protected static final String ROLE_TOOL = "tool";

        protected static final String ROLE_USER = "user";

        protected static final String TOOL_TYPE = "function";

        @JsonIgnore
        protected final Long created;

        @JsonIgnore
        protected String requestToolCallId;

        @JsonProperty("tool_calls")
        protected Object[] toolCalls;

        @JsonProperty("tool_call_id")
        protected String toolCallId;

        @JsonProperty("reasoning_content")
        protected String reasoning;

        protected Object content;

        protected String role;

        public OpenAiContent(List<MediaContext> mediaContext, OpenAiMedia openAiMedia, String type, String part, Long created) throws Exception {
            List<Map<String, Object>> content = new ArrayList<Map<String, Object>>();
            content.add(ImmutableMap.of("type", "text", "text", part));
            for (MediaContext media : mediaContext) {
                String mediaType = media.getType(type);
                String keyUrl = openAiMedia.getKeyUrl(mediaType);
                if (MediaContext.isInline(mediaType)) {
                    // Inline
                    content.add(ImmutableMap.of("type", keyUrl, keyUrl, ImmutableMap.of("url", openAiMedia.getPrefix(MediaContext.pureType(mediaType)) + media.getData())));
                } else {
                    content.add(ImmutableMap.of("type", keyUrl, keyUrl, ImmutableMap.of("url", media.getData())));
                }
            }
            this.role = OpenAiContent.ROLE_USER;
            this.content = content;
            this.created = created;
        }

        public OpenAiContent(LLMFunCallResponse llmFunCallResponse) throws Exception {
            this.content = llmFunCallResponse.getResponse();
            this.created = llmFunCallResponse.getCreated();
            this.toolCallId = llmFunCallResponse.getId();
            this.role = OpenAiContent.ROLE_TOOL;
        }

        public OpenAiContent(LLMFunCallRequest llmFunCallRequest) throws Exception {
            // 默认为空" "（不是空字符）
            this.reasoning = StringUtils.defaultIfEmpty(llmFunCallRequest.getReason(), "");
            this.toolCalls = this.buildToolCalls(llmFunCallRequest);
            this.created = llmFunCallRequest.getCreated();
            this.role = OpenAiContent.ROLE_ASSISTANT;
        }

        public OpenAiContent(String role, String content, Long created) throws Exception {
            this.created = created;
            this.content = content;
            this.role = role;
            Assert.notNull(this.content, "Open AI content can not be empty");
            Assert.hasText(this.role, "Role can not be empty");
        }

        public OpenAiContent(History history) throws Exception {
            this(history.isRole(History.ROLE_USER) ? OpenAiContent.ROLE_USER : OpenAiContent.ROLE_ASSISTANT, history.getContent(), history.getCreated());
            this.reasoning = history.getReason();
            Assert.hasText(history.getContent(), "History content can not be empty");
            Assert.notNull(history.getRole(), "History role can not be empty");
        }

        public OpenAiContent(Long created) throws Exception {
            this.created = created;
        }

        protected Object[] buildToolCalls(LLMFunCallRequest llmFunCallRequest) throws Exception {
            Map<String, Object> request = new HashMap<String, Object>();
            // 强制格式，保证模型间兼容
            request.put("function", ImmutableMap.of("arguments", StringUtils.defaultIfEmpty(JsonUtils.write(llmFunCallRequest.getArgs()), "{}"), "name", llmFunCallRequest.getName()));
            request.put("index", MapUtils.getIntValue(llmFunCallRequest.getRefer(), "index", 0));
            request.put("type", OpenAiContent.TOOL_TYPE);
            request.put("id", llmFunCallRequest.getId());
            this.requestToolCallId = llmFunCallRequest.getId();
            return new Object[]{request};
        }
    }

    @Getter
    public static class OpenAiTool {

        protected final Map<String, Object> function = new HashMap<String, Object>();

        protected final String type = "function";

        public OpenAiTool(ProviderFunCall providerFunCall) throws Exception {
            this.function.put("description", providerFunCall.getDescription());
            this.function.put("name", providerFunCall.getName());
            if (!MapUtils.isEmpty(providerFunCall.getProperties())) {
                this.function.put("parameters", ImmutableMap.of("properties", providerFunCall.getProperties(), "type", "object"));
            }
        }
    }

    @ConditionalOnProperty(name = "openai.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderRouterInitConfig {

        @Value("${openai.url:}")
        // Open API URL
        protected String url;

        @Bean(name = OpenAiRouter.NAME)
        @ConditionalOnMissingBean(name = OpenAiRouter.NAME)
        public OpenAiRouter openAiRouter() throws Exception {
            OpenAiRouter openAiRouter = new OpenAiRouter();
            BeanUtils.copyProperties(this, openAiRouter);
            log.info("OpenAiRouter inited");
            return openAiRouter;
        }
    }
}
