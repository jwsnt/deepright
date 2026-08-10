package ai.open.right.workflow.flow.llm.config;

import ai.open.right.utils.CollectionsUtils;
import ai.open.right.workflow.flow.config.GlobalConfig;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.mcp.RagMcpConfig;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.util.*;

@Setter
@Getter
@Slf4j
public class LLMConfig extends GlobalConfig {

    public static final Integer MAX_ERROR = 200;

    public static final Integer HISTORIES = 5;

    public static final Boolean STREAM = true;

    // LLM厂商相关的配置
    private Map<String, Object> additional = new HashMap<String, Object>();

    @JsonProperty("rag")
    // Rag配置
    private List<RagConfig> ragConfig = new ArrayList<RagConfig>();

    // LLM厂商API静态Headers
    protected Map<String, String> headers;

    // 开启多论会话后，模型响应写入的会话仓库（可以为多个），不配置则默认为当前思考链（Workflow）同名会话仓库
    protected List<String> repositories;

    @JsonProperty("funCall")
    // 需要使用的Fun Call配置（含Workflow FunCall、MCP FunCall）
    protected List<LLMFunCall> funCalls;

    // 响应结果静态修饰
    protected LLMDecoration decoration;

    // 动态System Prompt（DyPromptService）
    protected LLMDynamic dynamic;

    @JsonProperty("mcp")
    // MCP配置
    protected LLMMcpCall mcpCall;

    // 是否开启多论会话记忆
    protected Boolean containHistories;

    // 是否包含调用端记忆
    protected Boolean clientHistories;

    // 是否客户端记忆降级（默认为True，及时clientHistories=False，在无法从云端获取记忆是会使用客户端记忆）
    protected Boolean clientDowngrade;

    // 是否固定使用默认服务商（终端不可指定）
    protected Boolean regularProvider;

    // LLM调用来自Upstream时的调用超时，默认等于Timeout
    protected Integer upstreamTimeout;

    // FunCall是否继承/共享Metadata
    protected Boolean funCallHeritage;

    // LLM调用来自FunCall时的调用超时，默认等于Timeout
    // Upstream在FunCall的Sync Timeout
    protected Integer funCallTimeout;

    // 主线程等待FunCall时的超时，默认等于FunCallTimeout
    // 当UpStream为空时且Deepness = 1时表示主线程
    protected Integer funCallWaiting;

    // 获取Request/Response同时保存（默认True）
    protected Boolean storeCompleted;

    // Stream时的HTTP Buffer
    protected Integer networkBuffer;

    // 多论会话模式时的时间偏移（秒），默认0
    protected Integer recallOffset;

    // 是否召回FunCall（默认等于GetStoreFunCall）
    protected Boolean recallFunCall;

    // 是否存储FunCall
    protected Boolean storeFunCall;

    // OPEN AI系列：是否输出Reason到Segment.Content
    // Google 系列： 是否将Reason加入到多轮会话
    protected Boolean printReason;

    // Stream模式时的缓存
    protected Integer tokenBuffer;

    // Stream模式时的首包
    protected Integer tokenFirst;

    // 多论会话模式倒序召回（默认TRUE）
    protected Boolean recallDesc;

    // 是否存储Query
    protected Boolean storeQuery;

    // 多论会话模式召回数量（默认等于histories）
    protected Integer recallNums;

    // 多论会话模式时的记忆条数
    protected Integer histories;

    // 多论会话记忆存储时是否使用原始Query（Query在执行过程中会被ReQuery改写，比如Rag）
    protected Boolean pureQuery;

    // 多论会话模式时是否可写记忆，默认为True（开启False并配置Scene共享记忆时可以可读不可写，防止记忆污染）
    protected Boolean writeable;

    // 截取最大错误长度（String）
    protected Integer maxError;

    // 当下游为Right包装Open AI协议服务时，重组Metadata信息（ProviderRequestService.bridge）
    protected Boolean bridged;

    // 兼容性丢弃
    protected Boolean discard;

    // 多论会话模式时的记忆时间
    protected Integer expired;

    // LLM调用超时（覆盖默认超时）
    protected Integer timeout;

    // LLM响应通知方式（Localhost/Endpoint/Source）
    protected String notifier;

    // LLM供应商
    protected String provider;

    // 是否为流式响应
    protected Boolean stream;

    // System Prompt加载名称
    protected String prompt;

    // 共享记忆Key，配合Repositories允许跨思考链共享记忆
    protected String scene;

    // CoT思考链（Workflow）
    protected String chain;

    public LLMConfig merge(LLMConfig llmConfig) throws Exception {
        super.merge(llmConfig);
        if (llmConfig != null) {
            // this.prompt 不做Merge
            this.decoration = this.decoration != null ? this.decoration.merge(llmConfig.decoration) : llmConfig.decoration;
            this.containHistories = this.containHistories != null ? this.containHistories : llmConfig.containHistories;
            this.clientDowngrade = this.clientDowngrade != null ? this.clientDowngrade : llmConfig.clientDowngrade;
            this.clientHistories = this.clientHistories != null ? this.clientHistories : llmConfig.clientHistories;
            this.regularProvider = this.regularProvider != null ? this.regularProvider : llmConfig.regularProvider;
            this.upstreamTimeout = this.upstreamTimeout != null ? this.upstreamTimeout : llmConfig.upstreamTimeout;
            this.funCallHeritage = this.funCallHeritage != null ? this.funCallHeritage : llmConfig.funCallHeritage;
            this.funCallWaiting = this.funCallWaiting != null ? this.funCallWaiting : llmConfig.funCallWaiting;
            this.funCallTimeout = this.funCallTimeout != null ? this.funCallTimeout : llmConfig.funCallTimeout;
            this.storeCompleted = this.storeCompleted != null ? this.storeCompleted : llmConfig.storeCompleted;
            this.dynamic = this.dynamic != null ? this.dynamic.merge(llmConfig.dynamic) : llmConfig.dynamic;
            this.mcpCall = this.mcpCall != null ? this.mcpCall.merge(llmConfig.mcpCall) : llmConfig.mcpCall;
            this.networkBuffer = this.networkBuffer != null ? this.networkBuffer : llmConfig.networkBuffer;
            this.recallFunCall = this.recallFunCall != null ? this.recallFunCall : llmConfig.recallFunCall;
            this.storeFunCall = this.storeFunCall != null ? this.storeFunCall : llmConfig.storeFunCall;
            this.recallOffset = this.recallOffset != null ? this.recallOffset : llmConfig.recallOffset;
            this.printReason = this.printReason != null ? this.printReason : llmConfig.printReason;
            this.tokenBuffer = this.tokenBuffer != null ? this.tokenBuffer : llmConfig.tokenBuffer;
            this.repositories = CollectionsUtils.merge(this.repositories, llmConfig.repositories);
            this.storeQuery = this.storeQuery != null ? this.storeQuery : llmConfig.storeQuery;
            this.recallDesc = this.recallDesc != null ? this.recallDesc : llmConfig.recallDesc;
            this.tokenFirst = this.tokenFirst != null ? this.tokenFirst : llmConfig.tokenFirst;
            this.additional = CollectionsUtils.merge(this.additional, llmConfig.additional);
            this.histories = this.histories != null ? this.histories : llmConfig.histories;
            // GetRecallNums默认值为History，需要直接使用属性
            this.recallNums = this.recallNums != null ? this.recallNums : llmConfig.recallNums;
            this.pureQuery = this.pureQuery != null ? this.pureQuery : llmConfig.pureQuery;
            this.writeable = this.writeable != null ? this.writeable : llmConfig.writeable;
            this.notifier = StringUtils.defaultIfBlank(this.notifier, llmConfig.notifier);
            this.provider = StringUtils.defaultIfBlank(this.provider, llmConfig.provider);
            this.ragConfig = CollectionsUtils.merge(this.ragConfig, llmConfig.ragConfig);
            this.funCalls = CollectionsUtils.merge(this.funCalls, llmConfig.funCalls);
            this.maxError = this.maxError != null ? this.maxError : llmConfig.maxError;
            this.discard = this.discard != null ? this.discard : llmConfig.discard;
            this.bridged = this.bridged != null ? this.bridged : llmConfig.bridged;
            this.expired = this.expired != null ? this.expired : llmConfig.expired;
            this.timeout = this.timeout != null ? this.timeout : llmConfig.timeout;
            this.headers = CollectionsUtils.merge(this.headers, llmConfig.headers);
            this.scene = StringUtils.defaultIfBlank(this.scene, llmConfig.scene);
            this.chain = StringUtils.defaultIfBlank(this.chain, llmConfig.chain);
            this.stream = this.stream != null ? this.stream : llmConfig.stream;
        }
        return this;
    }

    public LLMConfig init(String chain, String notifier, String trigger, String rewriter) {
        // Chain && Stream and not open notifier
        this.chain = chain;
        this.notifier = StringUtils.defaultString(this.notifier, notifier);
        if (!StringUtils.isEmpty(this.notifier) && this.hasDynamicPrompt()) {
            this.getDynamic().init(this.notifier);
        }
        this.initRewriter(rewriter);
        this.initTrigger(trigger);
        this.initRag();
        return this;
    }

    public LLMConfig init(String chain) {
        return this.init(chain, null, null, null);
    }

    protected void initTrigger(String trigger) {
        if (!StringUtils.isEmpty(trigger)) {
            LLMMcpCall llmMcp = this.hasMcpCall() ? this.getMcpCall() : new LLMMcpCall();
            llmMcp.setTrigger(StringUtils.defaultString(llmMcp.getTrigger(), trigger));
            this.setMcpCall(llmMcp);
            // 为每个Rag更新Trigger
            for (RagConfig ragConfig : this.ragConfig) {
                // 没有则创建
                RagMcpConfig ragMcp = ragConfig.hasRagMcp() ? ragConfig.getRagMcpConfig() : new RagMcpConfig();
                ragMcp.setTrigger(StringUtils.defaultString(ragMcp.getTrigger(), trigger));
                ragConfig.setRagMcpConfig(ragMcp);
            }
        }
    }

    protected void initRewriter(String rewriter) {
        if (!StringUtils.isEmpty(rewriter)) {
            LLMMcpCall llmMcp = this.hasMcpCall() ? this.getMcpCall() : new LLMMcpCall();
            llmMcp.setRewriter(StringUtils.defaultString(llmMcp.getRewriter(), rewriter));
            this.setMcpCall(llmMcp);
            // 为每个Rag更新Rewriter
            for (RagConfig ragConfig : this.ragConfig) {
                RagMcpConfig ragMcp = ragConfig.hasRagMcp() ? ragConfig.getRagMcpConfig() : new RagMcpConfig();
                ragMcp.setRewriter(StringUtils.defaultString(ragMcp.getRewriter(), rewriter));
                ragConfig.setRagMcpConfig(ragMcp);
            }
        }
    }


    public void initRag() {
        if (!CollectionUtils.isEmpty(this.ragConfig)) {
            for (RagConfig each : this.ragConfig) {
                each.init(this);
            }
        }
    }

    public List<String> buildRepositories(String repository) {
        return LLMConfig.buildRepositories(this.getRepositories(), repository);
    }

    public List<String> buildRepositories() {
        return this.buildRepositories(this.getScene());
    }

    public Integer getUpstreamTimeout(Integer upstreamTimeout) {
        return this.upstreamTimeout != null ? this.upstreamTimeout : (upstreamTimeout != null ? upstreamTimeout : this.getTimeout());
    }

    public Integer getFunCallWaiting(Integer funCallWaiting) {
        return this.funCallWaiting != null ? this.funCallWaiting : (funCallWaiting != null ? funCallWaiting : this.getTimeout());
    }

    public Integer getFunCallTimeout(Integer funCallTimeout) {
        return this.funCallTimeout != null ? this.funCallTimeout : (funCallTimeout != null ? funCallTimeout : this.getTimeout());
    }

    public Integer getTimeout(Integer timeout) {
        return this.timeout != null ? this.timeout : timeout;
    }

    public String getScene(String scene) {
        return StringUtils.defaultString(this.scene, scene);
    }

    public Boolean getFunCallHeritage() {
        return this.funCallHeritage != null ? this.funCallHeritage : false;
    }

    public Boolean getContainHistories() {
        return this.containHistories != null ? this.containHistories : false;
    }

    public Boolean getClientDowngrade() {
        return this.clientDowngrade != null ? this.clientDowngrade : true;
    }

    public Boolean getClientHistories() {
        return this.clientHistories != null ? this.clientHistories : true;
    }

    public Boolean getRegularProvider() {
        return this.regularProvider != null ? this.regularProvider : false;
    }

    public Boolean getStoreCompleted() {
        return this.storeCompleted != null ? this.storeCompleted : true;
    }

    public Boolean getRecallFunCall() {
        return this.recallFunCall != null ? this.recallFunCall : this.getStoreFunCall();
    }

    public Boolean getStoreFunCall() {
        return this.storeFunCall != null ? this.storeFunCall : false;
    }

    public Integer getRecallOffset() {
        return this.recallOffset != null ? this.recallOffset : 0;
    }

    public Boolean getPrintReason() {
        return this.printReason != null ? this.printReason : false;
    }

    public Integer getTokenBuffer() {
        return this.tokenBuffer != null ? this.tokenBuffer : 5;
    }

    public Integer getTokenFirst() {
        return this.tokenFirst != null ? this.tokenFirst : 5;
    }

    public Boolean getRecallDesc() {
        return this.recallDesc != null ? this.recallDesc : false;
    }

    public Boolean getStoreQuery() {
        return this.storeQuery != null ? this.storeQuery : true;
    }

    public Integer getRecallNums() {
        return this.recallNums != null ? this.recallNums : this.getHistories();
    }

    public Integer getHistories() {
        return this.histories != null ? this.histories : LLMConfig.HISTORIES;
    }

    public Boolean getPureQuery() {
        return this.pureQuery != null ? this.pureQuery : true;
    }

    public Integer getMaxError() {
        return this.maxError != null ? this.maxError : LLMConfig.MAX_ERROR;
    }

    public Boolean getBridged() {
        return this.bridged != null ? this.bridged : true;
    }

    public Boolean getDiscard() {
        return this.discard != null ? this.discard : true;
    }

    public Boolean getStream() {
        return this.stream != null ? this.stream : LLMConfig.STREAM;
    }

    public Boolean hasDynamicPrompt() {
        return this.dynamic != null;
    }

    public Boolean hasNetworkBuffer() {
        return this.networkBuffer != null;
    }

    public Boolean hasRecallOffset() {
        return this.recallOffset != null;
    }

    public Boolean hasDecoration() {
        return this.decoration != null;
    }

    public Boolean hasNotifier() {
        return !StringUtils.isEmpty(this.notifier);
    }

    public Boolean hasHeaders() {
        return !CollectionUtils.isEmpty(this.headers);
    }

    public Boolean hasMcpCall() {
        return this.getMcpCall() != null;
    }

    public Boolean hasChain() {
        return !StringUtils.isEmpty(this.chain);
    }

    // 替换非常驻供应商
    public void replaceProvider(String provider) {
        if (StringUtils.isEmpty(this.getProvider()) || !this.getRegularProvider()) {
            this.provider = provider;
        }
    }

    public static List<String> buildRepositories(List<String> repositories, String scene) {
        if (CollectionUtils.isEmpty(repositories)) {
            Assert.hasText(scene, "The scene can not be empty");
            return Collections.singletonList(scene);
        }
        // Copy和Merge
        List<String> result = new ArrayList<String>(repositories);
        if (!StringUtils.isEmpty(scene) && !result.contains(scene)) {
            result.add(scene);
            return result;
        }
        return result;
    }

}
