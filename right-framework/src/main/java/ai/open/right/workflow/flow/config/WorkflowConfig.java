package ai.open.right.workflow.flow.config;

import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.CollectionsUtils;
import ai.open.right.workflow.flow.assistant.DefaultAssistant;
import ai.open.right.workflow.flow.competition.CompetitionConfig;
import ai.open.right.workflow.flow.fork.ForkConfig;
import ai.open.right.workflow.flow.function.FunctionConfig;
import ai.open.right.workflow.flow.iteration.IterationConfig;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.LLMFunCall;
import ai.open.right.workflow.flow.llm.signal.SignalConfig;
import ai.open.right.workflow.flow.llm.store.history.HistoryConfig;
import ai.open.right.workflow.flow.mapcombine.MapCombineConfig;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.flow.parallel.ParallelConfig;
import ai.open.right.workflow.flow.plan.PlanConfig;
import ai.open.right.workflow.flow.pubsub.PubSubConfig;
import ai.open.right.workflow.flow.resource.ResourceConfig;
import ai.open.right.workflow.flow.script.ScriptConfig;
import ai.open.right.workflow.flow.select.ChainSelectConfig;
import ai.open.right.workflow.flow.tools.ToolsConfig;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.CollectionUtils;

@Setter
@Getter
@ToString
public class WorkflowConfig extends GlobalConfig {

    public static final String REQUEST_JSON = "JSON";

    // unboxed
    public static final String UNBOXED = "__query";

    public static final String EXTENDED = ",";

    @JsonProperty("selector")
    // 动态下一个思考链（Workflow）配置
    protected ChainSelectConfig chainSelectConfig;

    @JsonProperty("competition")
    // 竞争思考链（Workflow）配置
    protected CompetitionConfig competitionConfig;

    @JsonProperty("mapCombine")
    // Map Combine思考链（Workflow）配置
    protected MapCombineConfig mapCombineConfig;

    @JsonProperty("iteration")
    // 迭代思考链（Workflow）配置
    protected IterationConfig iterationConfig;

    @JsonProperty("parallel")
    // 并行思考链（Workflow）配置
    protected ParallelConfig parallelConfig;

    @JsonProperty("resource")
    protected ResourceConfig resourceConfig;

    @JsonProperty("function")
    // 自定义功能思考链（Workflow）配置
    protected FunctionConfig functionConfig;

    @JsonProperty("history")
    // 短记忆处理思考链（Workflow）配置
    protected HistoryConfig historyConfig;

    @JsonProperty("signal")
    // 信号量配置
    protected SignalConfig signalConfig;

    @JsonProperty("script")
    // 脚本思考链（Workflow）配置
    protected ScriptConfig scriptConfig;

    @JsonProperty("pubsub")
    // PubSub思考链（Workflow）配置
    protected PubSubConfig pubSubConfig;

    @JsonProperty("media")
    // Media/Mime多媒体解析思考链（Workflow）配置
    protected MediaConfig mediaConfig;

    @JsonProperty("tools")
    // 自定义工具思考链（Workflow）配置
    protected ToolsConfig toolsConfig;

    @JsonProperty("plan")
    // 规划思考链（Workflow）配置
    protected PlanConfig planConfig;

    @JsonProperty("fork")
    // 分叉思考链（Workflow）配置
    protected ForkConfig forkConfig;

    @JsonProperty("funCall")
    protected LLMFunCall llmFunCall;

    @JsonProperty("llm")
    // LLM配置
    protected LLMConfig llmConfig;

    @JsonProperty("mcp")
    // MCP配置
    protected McpConfig mcpConfig;

    // A2A发布时的描述
    protected String description;

    // 是否保存用来回放的终端响应
    protected Boolean chatTrack;

    // 思考链（Workflow）使用的助手类型，默认为DefaultAssistant
    protected String assistant;

    protected Integer deepness;

    // 思考结果使用的通知方式（Localhost/Endpoint/Source）
    protected String notifier;

    // 继承自的四思考链配置
    protected String extended;

    // 用于Passage，静默结果
    protected Boolean discard;

    // 思考链（Workflow）触发监听器，WorkflowTrigger的实现
    protected String trigger;

    // 请求类型: Text, JSON
    protected String request;

    // 思考链（Workflow）型Fun Call的请求提取属性
    protected String unboxed;

    // 用于Passage，主动关闭通道
    protected Boolean close;

    // COT模式下一个思考链（Workflow）
    protected String chain;

    // 用于Passage，指定响应码
    protected Integer code;

    public WorkflowConfig merge(WorkflowConfig workflowConfig) throws Exception {
        super.merge(workflowConfig);
        if (workflowConfig != null) {
            this.chainSelectConfig = this.chainSelectConfig != null ? this.chainSelectConfig.merge(workflowConfig.chainSelectConfig) : workflowConfig.chainSelectConfig;
            this.competitionConfig = this.competitionConfig != null ? this.competitionConfig.merge(workflowConfig.competitionConfig) : workflowConfig.competitionConfig;
            this.mapCombineConfig = this.mapCombineConfig != null ? this.mapCombineConfig.merge(workflowConfig.mapCombineConfig) : workflowConfig.mapCombineConfig;
            this.iterationConfig = this.iterationConfig != null ? this.iterationConfig.merge(workflowConfig.iterationConfig) : workflowConfig.iterationConfig;
            this.parallelConfig = this.parallelConfig != null ? this.parallelConfig.merge(workflowConfig.parallelConfig) : workflowConfig.parallelConfig;
            this.functionConfig = this.functionConfig != null ? this.functionConfig.merge(workflowConfig.functionConfig) : workflowConfig.functionConfig;
            this.resourceConfig = this.resourceConfig != null ? this.resourceConfig.merge(workflowConfig.resourceConfig) : workflowConfig.resourceConfig;
            this.historyConfig = this.historyConfig != null ? this.historyConfig.merge(workflowConfig.historyConfig) : workflowConfig.historyConfig;
            this.signalConfig = this.signalConfig != null ? this.signalConfig.merge(workflowConfig.signalConfig) : workflowConfig.signalConfig;
            this.scriptConfig = this.scriptConfig != null ? this.scriptConfig.merge(workflowConfig.scriptConfig) : workflowConfig.scriptConfig;
            this.pubSubConfig = this.pubSubConfig != null ? this.pubSubConfig.merge(workflowConfig.pubSubConfig) : workflowConfig.pubSubConfig;
            this.mediaConfig = this.mediaConfig != null ? this.mediaConfig.merge(workflowConfig.mediaConfig) : workflowConfig.mediaConfig;
            this.toolsConfig = this.toolsConfig != null ? this.toolsConfig.merge(workflowConfig.toolsConfig) : workflowConfig.toolsConfig;
            this.planConfig = this.planConfig != null ? this.planConfig.merge(workflowConfig.planConfig) : workflowConfig.planConfig;
            this.forkConfig = this.forkConfig != null ? this.forkConfig.merge(workflowConfig.forkConfig) : workflowConfig.forkConfig;
            this.llmFunCall = this.llmFunCall != null ? this.llmFunCall.merge(workflowConfig.llmFunCall) : workflowConfig.llmFunCall;
            this.llmConfig = this.llmConfig != null ? this.llmConfig.merge(workflowConfig.llmConfig) : workflowConfig.llmConfig;
            this.mcpConfig = this.mcpConfig != null ? this.mcpConfig.merge(workflowConfig.mcpConfig) : workflowConfig.mcpConfig;
            this.description = StringUtils.defaultIfBlank(this.description, workflowConfig.description);
            this.assistant = StringUtils.defaultIfBlank(this.assistant, workflowConfig.assistant);
            this.chatTrack = this.chatTrack != null ? this.chatTrack : workflowConfig.chatTrack;
            this.notifier = StringUtils.defaultIfBlank(this.notifier, workflowConfig.notifier);
            this.extended = StringUtils.defaultIfBlank(this.extended, workflowConfig.extended);
            this.deepness = this.deepness != null ? this.deepness : workflowConfig.deepness;
            this.trigger = StringUtils.defaultIfBlank(this.trigger, workflowConfig.trigger);
            this.unboxed = StringUtils.defaultIfBlank(this.unboxed, workflowConfig.unboxed);
            this.discard = this.discard != null ? this.discard : workflowConfig.discard;
            this.chain = StringUtils.defaultIfBlank(this.chain, workflowConfig.chain);
            this.close = this.close != null ? this.close : workflowConfig.close;
            this.code = this.code != null ? this.code : workflowConfig.code;
        }
        return this;
    }

    public WorkflowConfig init() {
        // 传递初始化LLMConfig的Chain/Notifier/Rewriter
        String rewriter = this.hasMcp() ? this.mcpConfig.getRewriter() : null;
        String trigger = this.hasMcp() ? this.mcpConfig.getTrigger() : null;
        this.llmConfig.init(this.chain, this.notifier, trigger, rewriter);
        if (this.hasIteration()) {
            this.iterationConfig.init(this.llmConfig);
        }
        if (this.hasPlan()) {
            this.planConfig.init(this.llmConfig);
        }
        if (!StringUtils.isEmpty(this.notifier)) {
            if (this.hasIteration()) {
                this.iterationConfig.init(this.notifier);
            }
            if (this.hasMapCombine()) {
                this.mapCombineConfig.init(this.notifier);
            }
            if (this.hasParallel()) {
                this.parallelConfig.init(this.notifier);
            }
            if (this.hasFunCall()) {
                this.llmFunCall.init(this.notifier);
            }
            if (this.hasPubSub()) {
                this.pubSubConfig.init(this.notifier);
            }
            if (this.hasScript()) {
                this.scriptConfig.init(this.notifier);
            }
            if (this.hasPlan()) {
                this.planConfig.init(this.notifier);
            }
        }
        return this;
    }

    public Boolean hasCompetition() {
        return this.competitionConfig != null;
    }

    public Boolean hasMapCombine() {
        return this.mapCombineConfig != null;
    }

    public Boolean hasIteration() {
        return this.iterationConfig != null;
    }

    public Boolean hasChatTrack() {
        return this.getChatTrack();
    }

    public Boolean hasParallel() {
        return this.parallelConfig != null;
    }

    public Boolean hasExtended() {
        return !StringUtils.isEmpty(this.extended);
    }

    public Boolean hasSignals() {
        return this.signalConfig != null && !CollectionUtils.isEmpty(this.signalConfig.getConfigs());
    }

    public Boolean hasFunction() {
        return this.functionConfig != null;
    }

    public Boolean hasNotifier() {
        return !StringUtils.isEmpty(this.notifier);
    }

    public Boolean hasSelector() {
        // 配置了ChainSelectConfig.Selector或者Function
        return this.chainSelectConfig != null && (this.chainSelectConfig.hasDynamic() || this.chainSelectConfig.hasFunction() || this.chainSelectConfig.hasChain());
    }

    public Boolean hasFunCall() {
        return this.llmFunCall != null;
    }

    public Boolean hasHistory() {
        return this.historyConfig != null;
    }

    public Boolean hasTrigger() {
        return !StringUtils.isEmpty(this.trigger);
    }

    public Boolean hasPubSub() {
        return this.pubSubConfig != null;
    }

    public Boolean hasScript() {
        return this.scriptConfig != null;
    }

    public Boolean hasGlobal() {
        return !CollectionUtils.isEmpty(this.globalConfig);
    }

    public Boolean hasMedia() {
        return this.mediaConfig != null;
    }

    public Boolean hasChain() {
        return !StringUtils.isEmpty(this.chain);
    }

    public Boolean hasTools() {
        return this.toolsConfig != null;
    }

    public Boolean hasPlan() {
        return this.planConfig != null;
    }

    public Boolean hasFork() {
        return this.forkConfig != null;
    }

    public Boolean hasMcp() {
        return this.mcpConfig != null;
    }

    public Boolean hasLlm() {
        return this.llmConfig != null;
    }

    public String getNotifier(String notifier) {
        return this.hasNotifier() ? this.notifier : notifier;
    }

    public Boolean getChatTrack() {
        return this.chatTrack != null ? this.chatTrack : false;
    }

    public String getAssistant() {
        return this.assistant != null ? this.assistant : DefaultAssistant.WORKFLOW_NAME;
    }

    public Boolean getDiscard() {
        return this.discard != null ? this.discard : false;
    }

    public String getUnboxed() {
        return this.unboxed != null ? this.unboxed : WorkflowConfig.UNBOXED;
    }

    public Boolean getClose() {
        return this.close != null ? this.close : false;
    }

    public Integer getCode() {
        return this.code != null ? this.code : ProtocolCode.C200;
    }
}
