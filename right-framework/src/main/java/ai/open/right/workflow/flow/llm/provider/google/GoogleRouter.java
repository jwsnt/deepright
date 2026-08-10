package ai.open.right.workflow.flow.llm.provider.google;

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
import lombok.Builder;
import lombok.Getter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.apache.http.client.methods.HttpPost;
import org.springframework.util.Assert;

import java.util.*;

@Slf4j
abstract public class GoogleRouter extends ProviderRouter<GoogleRequest> {

    @Override
    protected void reConfig(GoogleRequest request, LLMConfig llmConfig, HttpPost httpPost) throws Exception {
        super.reConfig(request, llmConfig, httpPost);
        this.addHeader(request, httpPost);
    }

    @Override
    protected GoogleReader reader(GoogleRequest request, LLMConfig llmConfig, LLMCallback llmCallback) throws Exception {
        return new GoogleReader(ProviderReaderConfig.<GoogleRequest>builder()
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

    protected void addHeader(GoogleRequest request, HttpPost httpPost) throws Exception {
        httpPost.addHeader("Authorization", request.getToken());
    }

    @Override
    protected Object body(GoogleRequest request) throws Exception {
        return new GoogleMessage(request);
    }

    @Getter
    public static class GoogleMessage {

        @JsonIgnore
        protected Map<String, LLMFunCallResponse> funCallResponses;

        @JsonProperty("safetySettings")
        protected List<Map<String, Object>> safetySettings;

        @JsonProperty("contents")
        private final List<GoogleContent> contents;

        @JsonProperty("tool_config")
        protected Map<String, Object> toolConfig;

        @JsonProperty("systemInstruction")
        protected GoogleInstruction instruction;

        @JsonProperty("generationConfig")
        protected Map<String, Object> configs;

        @JsonProperty("labels")
        protected Map<String, String> labels;

        @JsonProperty("tools")
        protected GoogleTools[] tools;

        public GoogleMessage(GoogleRequest googleRequest) throws Exception {
            this.instruction = new GoogleInstruction(googleRequest.getPrompt());
            this.safetySettings = googleRequest.getSafetySettings();
            this.toolConfig = googleRequest.getToolsConfig();
            this.contents = new ArrayList<GoogleContent>();
            this.labels = googleRequest.getLabels();
            this.configs = googleRequest.configs();
            ///////////////////////////
            // System Prompt(top-level) -> History/FunCall/MimeQuery(sorted by created)
            //////////////////////////
            if (googleRequest.getMessage().hasHistory()) {
                // Gemini compatibility: keep a synthetic empty user turn ahead of recalled user-first history.
                this.appendQuery(googleRequest);
                this.buildHistories(googleRequest, googleRequest.getMessage().getHistories());
            }
            this.buildFunCallRequest(googleRequest);
            this.buildUserQuery(googleRequest);
            this.contents.sort(Comparator.comparing(GoogleContent::getCreated, Comparator.nullsFirst(Long::compareTo)));
            this.buildFunCallResponse(googleRequest);
            // FunCall Config
            if (googleRequest.hasFunCall()) {
                this.tools = new GoogleTools[]{new GoogleTools(googleRequest.getFunCalls())};
            }
        }

        protected void buildHistories(GoogleRequest googleRequest, List<History> histories) {
            if (!CollectionUtils.isEmpty(histories)) {
                for (History each : histories) {
                    try {
                        if (each.isFunction(History.FUN_CHAT)) {
                            // 是否输出PrintReason
                            this.contents.add(new GoogleContent(googleRequest.getPrintReason() ? each.getReason() : null, each.getContent(), each.isRole(History.ROLE_USER) ? GoogleContent.ROLE_USER : GoogleContent.ROLE_ASSISTANT, each.getCreated()));
                        } else {
                            // Fun Call
                            if (each.isType(History.TYPE_ANSWER)) {
                                ProviderFunCallResponse response = JsonUtils.read(each.getContent(), ProviderFunCallResponse.class);
                                this.contents.add(new GoogleContent(response));
                            }
                            if (each.isType(History.TYPE_QUERY)) {
                                ProviderFunCallRequest request = JsonUtils.read(each.getContent(), ProviderFunCallRequest.class);
                                this.contents.add(new GoogleContent(request));
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

        protected void buildUserQuery(GoogleRequest googleRequest) throws Exception {
            // Mime Query
            if (googleRequest.hasMimeContext()) {
                this.contents.add(new GoogleContent(googleRequest.getMediaContext(), googleRequest.getMimeType(), googleRequest.getMessage().getQuery(), googleRequest.getMessage().getCreated()));
            } else {
                // Text Query
                this.contents.add(new GoogleContent(googleRequest.getMessage().getQuery(), GoogleContent.ROLE_USER, googleRequest.getMessage().getCreated()));
            }
        }

        protected void buildFunCallResponse(GoogleRequest googleRequest) throws Exception {
            if (!MapUtils.isEmpty(this.funCallResponses)) {
                ListIterator<GoogleContent> iterator = this.contents.listIterator();
                while (iterator.hasNext()) {
                    GoogleContent element = iterator.next();
                    if (!StringUtils.isEmpty(element.getRequestToolCallId())) {
                        LLMFunCallResponse funCallResponse = this.funCallResponses.get(element.getRequestToolCallId());
                        if (funCallResponse != null) {
                            iterator.add(new GoogleContent(funCallResponse));
                        }
                    }
                }
            }
        }

        protected void buildFunCallRequest(GoogleRequest googleRequest) throws Exception {
            if (googleRequest.hasFunCallData()) {
                this.funCallResponses = new HashMap<String, LLMFunCallResponse>();
                LLMFunCallData funCallData = googleRequest.getFunCallData();
                for (int index = 0; index < funCallData.getRequests().size() && index < funCallData.getResponses().size(); index++) {
                    String funCallId = this.buildFunCallId(funCallData.getRequests().get(index), funCallData.getResponses().get(index), index);
                    LLMFunCallRequest funCallRequest = this.normalizeFunCallRequest(funCallData.getRequests().get(index), funCallId);
                    LLMFunCallResponse funCallResponse = this.normalizeFunCallResponse(funCallData.getResponses().get(index), funCallId);
                    this.funCallResponses.put(funCallId, funCallResponse);
                    this.contents.add(new GoogleContent(funCallRequest));
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

        protected void appendQuery(GoogleRequest googleRequest) throws Exception {
            // https://discuss.ai.google.dev/t/about-the-gemini-api-400-please-ensure-that-function-call-turns-come-immediately-after-a-user-turn-or-after-a-function-response-turn-error/46213
            if (History.ROLE_USER.equals(googleRequest.getMessage().getHistories().getFirst().getRole())) {
                Long firstTimeline = History.buildFirstTimeline(googleRequest.getMessage().getHistories());
                Long created = googleRequest.getMessage().getCreated();
                if (firstTimeline != null) {
                    created = Math.min(created, firstTimeline);
                }
                created = Long.MIN_VALUE == created ? created : created - 1;
                // Empty Query
                this.contents.add(new GoogleContent("", GoogleContent.ROLE_USER, created));
            }
        }

        @Getter
        public static class GoogleContent {

            public static final String ROLE_ASSISTANT = "model";

            public static final String ROLE_USER = "user";

            protected final List<GooglePart> parts = new ArrayList<GooglePart>();

            @JsonIgnore
            protected final Long created;

            @JsonIgnore
            protected String requestToolCallId;

            protected String role;

            public GoogleContent(List<MediaContext> mediaContext, String type, String part, Long created) throws Exception {
                this(part, GoogleContent.ROLE_USER, created);
                for (MediaContext media : mediaContext) {
                    String mimeType = media.getType(type);
                    if (MediaContext.isInline(mimeType)) {
                        this.parts.add(new GooglePart(GoogleMime.builder().mimeType(MediaContext.pureType(mimeType)).data(media.getData()).build(), GooglePart.DATA_INLINE));
                    } else {
                        mimeType = !StringUtils.isEmpty(mimeType) ? mimeType : MediaContext.mimeType(media.getData());
                        this.parts.add(new GooglePart(GoogleMime.builder().uri(media.getData()).mimeType(mimeType).build(), GooglePart.DATA_FILE));
                    }
                }
            }

            public GoogleContent(String thoughtSignature, String part, String role, Long created) throws Exception {
                Assert.notNull(part, "Google content can not be empty");
                this.parts.add(new GooglePart(thoughtSignature, part));
                this.created = created;
                this.role = role;
            }

            public GoogleContent(LLMFunCallResponse llmFunCallResponse) throws Exception {
                this.parts.add(new GooglePart(llmFunCallResponse));
                this.created = llmFunCallResponse.getCreated();
                this.role = GoogleContent.ROLE_ASSISTANT;
            }

            public GoogleContent(LLMFunCallRequest llmFunCallRequest) throws Exception {
                this.parts.add(new GooglePart(llmFunCallRequest));
                this.requestToolCallId = llmFunCallRequest.getId();
                this.created = llmFunCallRequest.getCreated();
                this.role = GoogleContent.ROLE_ASSISTANT;
            }

            public GoogleContent(String part, String role, Long created) throws Exception {
                Assert.notNull(part, "Google content can not be empty");
                this.parts.add(new GooglePart(part));
                this.created = created;
                this.role = role;
            }

            public GoogleContent(Long created) throws Exception {
                this.created = created;
            }
        }

        @Getter
        public static class GoogleInstruction {

            @JsonProperty("parts")
            protected GooglePart part;

            public GoogleInstruction(String part) throws Exception {
                this.part = new GooglePart(part);
            }
        }

        @Getter
        public static class GooglePart {

            public static final String SIGNATURE = "c2lnbmF0dXJlLTQ3NzQ4MTdiLTFhNGItNDU5MC04MWRmLWY4ZjZkOWY0NzM3YQ==";

            public static final String DATA_INLINE = "inline";

            public static final String DATA_FILE = "file";

            protected Map<String, Object> functionResponse;

            protected Map<String, Object> functionCall;

            // Gemini的ThoughtSignature似乎存在时效限制，不可无限制存储
            // Gemini 3.0
            protected String thoughtSignature;

            @JsonProperty("inline_data")
            protected GoogleMime inline;

            @JsonProperty("file_data")
            protected GoogleMime file;

            protected String text;

            public GooglePart(GoogleMime googleMime, String type) throws Exception {
                if (GooglePart.DATA_INLINE.equalsIgnoreCase(type)) {
                    this.inline = googleMime;
                } else {
                    this.file = googleMime;
                }
            }

            public GooglePart(LLMFunCallResponse llmFunCallResponse) throws Exception {
                this.functionResponse = new HashMap<String, Object>();
                String body = llmFunCallResponse.getResponse();
                try {
                    if (JsonUtils.like(body)) {
                        this.functionResponse.put("response", JsonUtils.read(body, Map.class));
                    }
                } catch (Exception e) {
                    log.debug(e.getMessage(), e);
                }
                if (this.functionResponse.get("response") == null) {
                    this.functionResponse.put("response", ImmutableMap.of("content", StringUtils.defaultIfEmpty(llmFunCallResponse.getResponse(), "")));
                }
                this.functionResponse.put("name", llmFunCallResponse.getName());
            }

            public GooglePart(LLMFunCallRequest llmFunCallRequest) throws Exception {
                // 可以在思路签名字段中设置 "context_engineering_is_the_way_to_go" 或 "skip_thought_signature_validator" 虚拟签名，跳过验证
                // c2lnbmF0dXJlLTQ3NzQ4MTdiLTFhNGItNDU5MC04MWRmLWY4ZjZkOWY0NzM3YQ==
                this.thoughtSignature = StringUtils.defaultIfEmpty(MapUtils.getString(Map.class.cast(llmFunCallRequest.getRefer()), "thoughtSignature"), GooglePart.SIGNATURE);
                this.functionCall = new HashMap<String, Object>();
                this.functionCall.put("name", llmFunCallRequest.getName());
                this.functionCall.put("args", this.buildJson(llmFunCallRequest.getArgs()));
            }

            public GooglePart(String thoughtSignature, String text) throws Exception {
                this(text);
                this.thoughtSignature = thoughtSignature;
            }

            public GooglePart(String text) throws Exception {
                this.text = text;
            }

            protected Map<String, Object> buildJson(Object args) throws Exception {
                try {
                    return Map.class.isAssignableFrom(args.getClass()) ? Map.class.cast(args) : JsonUtils.transfer(args, Map.class);
                } catch (Exception e) {
                    log.warn(e.getMessage(), e);
                    // 如果无法转换时的兜底
                    return ImmutableMap.of("args", StringUtils.defaultIfEmpty(JsonUtils.write(args), "{}"));
                }
            }
        }

        @Getter
        @Builder
        public static class GoogleMime {

            @JsonProperty("mime_type")
            protected String mimeType;

            @JsonProperty("data")
            protected String data;

            @JsonProperty("file_uri")
            protected String uri;
        }

        @Getter
        public static class GoogleTools {

            @JsonProperty("function_declarations")
            protected final List<Map<String, Object>> functions = new ArrayList<Map<String, Object>>();

            public GoogleTools(List<ProviderFunCall> providerFunCall) throws Exception {
                for (ProviderFunCall tool : providerFunCall) {
                    String description = tool.getDescription();
                    String name = tool.getName();
                    Assert.hasText(description, "Description can not be empty: " + name);
                    Assert.hasText(name, "Name can not be empty: " + name);
                    Map<String, Object> eachFunction = new HashMap<String, Object>();
                    eachFunction.put("description", description);
                    eachFunction.put("name", name);
                    Map<String, Object> parameters = new HashMap<String, Object>();
                    parameters.put("properties", tool.getProperties());
                    parameters.put("required", tool.getRequired());
                    parameters.put("type", "object");
                    eachFunction.put("parameters", parameters);
                    this.functions.add(eachFunction);
                }
            }
        }
    }
}
