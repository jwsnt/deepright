package ai.open.right.workflow.flow.llm.store.history;

import ai.open.right.WorkflowException;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.Markdown;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import com.fasterxml.jackson.annotation.JsonIgnore;
import com.fasterxml.jackson.annotation.JsonPropertyDescription;
import com.fasterxml.jackson.module.jsonSchema.factories.SchemaFactoryWrapper;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.lang3.StringEscapeUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;

import java.util.*;

@Setter
@Getter
@ToString
public class History {

    public static final List<History>[] EMPTY_SPLIT = (List<History>[]) new List<?>[]{Collections.unmodifiableList(Collections.EMPTY_LIST), Collections.unmodifiableList(Collections.EMPTY_LIST)};

    public static String SCHEMA_HISTORIES_MARKDOWN;

    public static String SCHEMA_HISTORY_MARKDOWN;

    public static String SCHEMA_HISTORIES;

    public static String SCHEMA_HISTORY;

    static {
        try {
            SchemaFactoryWrapper wrapper4histories = new SchemaFactoryWrapper();
            SchemaFactoryWrapper wrapper4history = new SchemaFactoryWrapper();
            JsonUtils.instance().acceptJsonFormatVisitor(History[].class, wrapper4histories);
            JsonUtils.instance().acceptJsonFormatVisitor(History.class, wrapper4history);
            History.SCHEMA_HISTORIES = JsonUtils.instance().writerWithDefaultPrettyPrinter().writeValueAsString(wrapper4histories.finalSchema());
            History.SCHEMA_HISTORY = JsonUtils.instance().writerWithDefaultPrettyPrinter().writeValueAsString(wrapper4history.finalSchema());
            Assert.isTrue(!StringUtils.isEmpty(History.SCHEMA_HISTORIES) && !StringUtils.isEmpty(History.SCHEMA_HISTORY), "The history schema can not empty");
            History.SCHEMA_HISTORIES_MARKDOWN = Markdown.array(History.SCHEMA_HISTORIES);
            History.SCHEMA_HISTORY_MARKDOWN = Markdown.object(History.SCHEMA_HISTORY);
            Assert.isTrue(!StringUtils.isEmpty(History.SCHEMA_HISTORIES_MARKDOWN) && !StringUtils.isEmpty(History.SCHEMA_HISTORY_MARKDOWN), "The history schema can not empty");
        } catch (Exception e) {
            throw new WorkflowException(e);
        }
    }

    public static final Integer REFERENCE_SERVER = 0;

    public static final Integer REFERENCE_CLIENT = 1;

    public static final Integer ROLE_ASSISTANT = 1;

    public static final Integer ROLE_USER = 0;

    public static final Integer TYPE_ANSWER = 1;

    public static final Integer TYPE_QUERY = 0;

    public static final Integer FUN_FUNCALL = 1;

    public static final Integer FUN_CHAT = 0;

    // 创建时间，默认当前
    @JsonPropertyDescription("The more recent the Created timestamp is, the newer the memory, and the greater its value.")
    protected Long created = System.currentTimeMillis();

    @JsonIgnore
    protected Integer reference = History.REFERENCE_SERVER;

    // 功能类型（会话、FunCall等）
    @JsonPropertyDescription("The function field: Chat=0, FunCall=1")
    protected Integer function = History.FUN_CHAT;

    // 产生的Conversation
    @JsonPropertyDescription("The event ID consists of multiple incrementing conversations. A larger conversation number indicates a newer message.")
    protected String conversation;

    protected String signature;

    @JsonPropertyDescription("The content of the current interaction.")
    protected String content;

    // BIZ@WORKFLOW
    @JsonPropertyDescription("The format is BIZ@WORKFLOW, used to mark the internal source.")
    protected String source;

    // 推理原因
    @JsonPropertyDescription("The field is for internal use and records the model inference process.")
    protected String reason;

    @JsonPropertyDescription("The model version used in this chat, e.g.: gemini-3.0.")
    protected String model;

    // 产生的Chat
    @JsonPropertyDescription("The event ID identifies the same problem to be solved and requires integrated reasoning. If the user's latest query has the same event ID, historical memory shall be used as prior input for solving the problem. If not, historical memory only serves as best practice during subsequent execution. Focus on memories with the same event ID as the latest query; others are only for reference on problem-solving.")
    protected String chat;

    @JsonPropertyDescription("The role identifier: User=0, Assistant=1.")
    // user 0 / assistant 1
    protected Integer role;

    @JsonPropertyDescription("The type field: Query=0, Answer=1.")
    // query 0 | answer 1
    protected Integer type;

    @JsonPropertyDescription("The model api used in this chat, e.g.: anthropic、open-ai、google")
    protected String api;

    public History(WorkflowTask workTask) {
        this.setConversation(workTask.getConversation());
        this.setReference(History.REFERENCE_SERVER);
        this.setSource(SplitUtils.join(workTask));
        this.setCreated(workTask.getCreated());
        this.setChat(workTask.getChat());
    }

    public History() {

    }

    public void setAssistant() {
        this.role = History.ROLE_ASSISTANT;
    }

    public void setAnswer() {
        this.type = History.TYPE_ANSWER;
    }

    public void setQuery() {
        this.type = History.TYPE_QUERY;
    }

    public void setUser() {
        this.role = History.ROLE_USER;
    }

    public String getFunctionAsString() {
        if (History.FUN_FUNCALL.equals(this.function)) {
            return "FUN_FUNCALL";
        }
        if (History.FUN_CHAT.equals(this.function)) {
            return "FUN_CHAT";
        }
        return "";
    }

    public Boolean isFunction(Integer function) {
        return this.function != null && this.function.equals(function);
    }

    public Boolean isType(Integer type) {
        return this.type != null && this.type.equals(type);
    }

    public Boolean isRole(Integer role) {
        return this.role != null && this.role.equals(role);
    }

    public Boolean isApi(String... api) {
        return StringUtils.equalsAnyIgnoreCase(this.api, api);
    }

    // Reason是否人类可读
    public Boolean isEncrypt() {
        return this.isApi(ProviderRequest.REQUEST_GOOGLE, ProviderRequest.REQUEST_ANTHROPIC);
    }

    public History copy() {
        History history = new History();
        history.setConversation(this.getConversation());
        history.setReference(this.getReference());
        history.setReason(this.getReason());
        history.setSignature(this.getSignature());
        history.setFunction(this.getFunction());
        history.setCreated(this.getCreated());
        history.setContent(this.getContent());
        history.setSource(this.getSource());
        history.setModel(this.getModel());
        history.setChat(this.getChat());
        history.setType(this.getType());
        history.setRole(this.getRole());
        history.setApi(this.getApi());
        return history;
    }

    // 当前Content转为指定对象
    public <T> T getObjectContent(Class<T> clazz) throws Exception {
        return JsonUtils.transfer(this.content, clazz);
    }

    public static List<History> getReferenceHistory(List<History> histories, Integer reference) {
        List<History> result = null;
        if (!CollectionUtils.isEmpty(histories)) {
            for (History history : histories) {
                if (reference != null && reference.equals(history.getReference())) {
                    result = result != null ? result : new ArrayList<History>();
                    result.add(history);
                }
            }
        }
        return result;
    }

    // 基础Markdown
    public static String buildMarkdown(List<History> histories, HistoryTruncate truncate) throws Exception {
        StringBuffer buffer = new StringBuffer();
        buffer.append("|The type field|The content of the current interaction|How is the content generated|The more recent the Created timestamp is, the newer the memory, and the greater its value|");
        buffer.append(System.lineSeparator());
        buffer.append("|---|---|---|---|");
        buffer.append(System.lineSeparator());
        // 最新排最上，Copy，不影响外层的顺序
        List<History> snapshot = new ArrayList<History>(histories);
        snapshot.sort(Comparator.comparing(History::getCreated).reversed());
        StringBuffer content = new StringBuffer();
        for (History history : snapshot) {
            content.append("|").append(history.isType(History.TYPE_QUERY) ? "QUERY" : "ANSWER");
            content.append("|").append(StringEscapeUtils.unescapeJava(history.getContent()));
            // 不为Google API且不为空
            content.append("|").append(!history.isEncrypt() ? StringUtils.defaultIfEmpty(history.getReason(), "") : "");
            content.append("|").append(history.getCreated());
            content.append("|").append(System.lineSeparator());
        }
        // 可能产生截断造成不完整的MD，符合预期
        buffer.append(truncate != null ? truncate.truncate(content.toString()) : content);
        buffer.append(System.lineSeparator());
        return buffer.toString();
    }

    // 最后一条消息（最新）的时间
    public static Long buildLastTimeline(List<History> histories) {
        return !CollectionUtils.isEmpty(histories) ? histories.stream().map(History::getCreated).filter(Objects::nonNull).max(Long::compare).orElse(null) : null;
    }

    // 第一条消息（最旧）的时间
    public static Long buildFirstTimeline(List<History> histories) {
        return !CollectionUtils.isEmpty(histories) ? histories.stream().map(History::getCreated).filter(Objects::nonNull).min(Long::compare).orElse(null) : null;
    }
}
