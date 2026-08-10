package ai.open.right.workflow.flow.llm.store.history;

import ai.open.right.WorkflowException;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.Markdown;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import com.fasterxml.jackson.annotation.JsonPropertyDescription;
import com.fasterxml.jackson.module.jsonSchema.factories.SchemaFactoryWrapper;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;

@Setter
@Getter
public class HistoryPair {

    public static String SCHEMA_HISTORIES_MARKDOWN;

    public static String SCHEMA_HISTORY_MARKDOWN;

    public static String SCHEMA_HISTORIES;

    public static String SCHEMA_HISTORY;

    static {
        try {
            SchemaFactoryWrapper wrapper4histories = new SchemaFactoryWrapper();
            SchemaFactoryWrapper wrapper4history = new SchemaFactoryWrapper();
            JsonUtils.instance().acceptJsonFormatVisitor(HistoryPair[].class, wrapper4histories);
            JsonUtils.instance().acceptJsonFormatVisitor(HistoryPair.class, wrapper4history);
            HistoryPair.SCHEMA_HISTORIES = JsonUtils.instance().writerWithDefaultPrettyPrinter().writeValueAsString(wrapper4histories.finalSchema());
            HistoryPair.SCHEMA_HISTORY = JsonUtils.instance().writerWithDefaultPrettyPrinter().writeValueAsString(wrapper4history.finalSchema());
            Assert.isTrue(!StringUtils.isEmpty(HistoryPair.SCHEMA_HISTORIES) && !StringUtils.isEmpty(HistoryPair.SCHEMA_HISTORY), "The history schema can not empty");
            HistoryPair.SCHEMA_HISTORIES_MARKDOWN = Markdown.array(HistoryPair.SCHEMA_HISTORIES);
            HistoryPair.SCHEMA_HISTORY_MARKDOWN = Markdown.object(HistoryPair.SCHEMA_HISTORY);
            Assert.isTrue(!StringUtils.isEmpty(HistoryPair.SCHEMA_HISTORIES_MARKDOWN) && !StringUtils.isEmpty(HistoryPair.SCHEMA_HISTORY_MARKDOWN), "The history schema can not empty");
        } catch (Exception e) {
            throw new WorkflowException(e);
        }
    }

    @JsonPropertyDescription("The more recent the Created timestamp is, the newer the memory, and the greater its value.")
    private Long created = System.currentTimeMillis();

    // 功能类型（会话、FunCall等）
    @JsonPropertyDescription("The function field: Chat=0, FunCall=1")
    protected Integer function = History.FUN_CHAT;

    // 产生的Conversation
    @JsonPropertyDescription("The event ID consists of multiple incrementing conversations. A larger conversation number indicates a newer message.")
    protected String conversation;

    protected String signature;

    // 推理原因
    @JsonPropertyDescription("The field is for internal use and records the model inference process.")
    protected String reasoning;

    // BIZ@WORKFLOW
    @JsonPropertyDescription("The format is BIZ@WORKFLOW, used to mark the internal source.")
    protected String source;

    @JsonPropertyDescription("The answer of the current interaction.")
    protected String answer;

    @JsonPropertyDescription("The query of the current interaction.")
    protected String query;

    @JsonPropertyDescription("The model version used in this chat, e.g.: gemini-3.0.")
    protected String model;

    @JsonPropertyDescription("The event ID identifies the same problem to be solved and requires integrated reasoning. If the user's latest query has the same event ID, historical memory shall be used as prior input for solving the problem. If not, historical memory only serves as best practice during subsequent execution. Focus on memories with the same event ID as the latest query; others are only for reference on problem-solving.")
    protected String chat;

    // 强制指定同一角色（默认为NULL）
    @JsonPropertyDescription("The role identifier: User=0, Assistant=1.")
    protected Integer role;

    @JsonPropertyDescription("The model api used in this chat, e.g.: anthropic、open-ai、google")
    protected String api;

    public HistoryPair(WorkflowTask workTask, Long created) {
        this.source = SplitUtils.join(workTask.getWorkflow(), workTask.getBiz());
        this.conversation = workTask.getConversation();
        this.chat = workTask.getChat();
        this.created = created;
    }

    public HistoryPair(History history) {
        this.role = history.isRole(History.ROLE_ASSISTANT) ? History.ROLE_ASSISTANT : History.ROLE_USER;
        this.function = history.isFunction(History.FUN_CHAT) ? History.FUN_CHAT : History.FUN_FUNCALL;
        this.conversation = history.getConversation();
        this.reasoning = history.getReason();
        this.created = history.getCreated();
        this.source = history.getSource();
        this.model = history.getModel();
        this.chat = history.getChat();
        this.api = history.getApi();
        if (history.isType(History.TYPE_ANSWER)) {
            this.answer = history.getContent();
        } else {
            this.query = history.getContent();
        }
    }

    public HistoryPair() {

    }

    public History[] buildHistories() {
        History[] histories = new History[2];
        if (!StringUtils.isEmpty(this.answer)) {
            History answer = new History();
            // 模型推理原因
            answer.setConversation(this.conversation);
            answer.setReason(this.reasoning);
            answer.setSignature(this.signature);
            answer.setContent(this.getAnswer());
            answer.setFunction(this.function);
            answer.setCreated(this.created);
            answer.setSource(this.source);
            answer.setModel(this.model);
            answer.setChat(this.chat);
            answer.setApi(this.api);
            answer.setAssistant();
            answer.setAnswer();
            histories[1] = this.changeRole(answer);
        }
        if (!StringUtils.isEmpty(this.query)) {
            History query = new History();
            query.setConversation(this.conversation);
            query.setContent(this.getQuery());
            query.setFunction(this.function);
            query.setCreated(this.created);
            query.setSource(this.source);
            query.setModel(this.model);
            query.setChat(this.chat);
            query.setApi(this.api);
            query.setQuery();
            query.setUser();
            histories[0] = this.changeRole(query);
        }
        return histories;
    }

    // 强制切换角色
    public History changeRole(History history) {
        if (this.role != null && !history.isRole(this.role)) {
            history.setRole(this.role);
        }
        return history;
    }
}