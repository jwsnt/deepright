package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.WorkflowException;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.config.WorkflowConfigService;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.LLMFunCall;
import ai.open.right.workflow.flow.llm.config.LLMMcpCall;
import ai.open.right.workflow.flow.llm.config.LLMPromptService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.mcp.client.McpClientService;
import ai.open.right.workflow.mcp.client.McpResult;
import ai.open.right.workflow.mcp.client.McpRuntime;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.dimension.McpDimensionService;
import ai.open.right.workflow.mcp.client.rewrtier.McpRewriteService;
import ai.open.right.workflow.mcp.client.trigger.McpTriggerService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.util.Assert;

import java.util.*;
import java.util.concurrent.TimeUnit;

@Slf4j
@Setter
@Getter
abstract public class ProviderRequestService<T extends ProviderRequest> {

    public static final String KEY_REASONING_EFFORT = "reasoning_effort";

    public static final String KEY_RESPONSE_SCHEMA = "response_schema";

    public static final String KEY_EXTRA_BODY = "extra_body";

    public static final String KEY_THINKING = "thinking";

    public static final String KEY_MIMETYPE = "mimeType";

    public static final String KEY_INTERNAL = "__";

    public static final String KEY_MODEL = "model";

    public static final String KEY_SEED = "seed";

    // 获取FunCall结果时的内部思考链（Workflow）
    public static final String KEY_FUN_INTERNAL = ProviderRequestService.KEY_INTERNAL + "fun__internal";

    // 选择Fun Call List最优结果的内部思考链（Workflow）
    public static final String KEY_FUN_SELECT = ProviderRequestService.KEY_INTERNAL + "fun__select";

    // 处理多媒体的内部思考链（Workflow）
    public static final String KEY_FUN_MEDIA = ProviderRequestService.KEY_INTERNAL + "fun__media";

    // 获取每个FunCall结果
    public static final String KEY_FUN_FETCH = ProviderRequestService.KEY_INTERNAL + "fun__fetch";

    // 获取所有FunCall结果后合并再次提交的内部思考链（Workflow）
    public static final String KEY_FUN_MERGE = ProviderRequestService.KEY_INTERNAL + "fun__merge";

    // 调用方指定模型服务商
    public static final String KEY_PROVIDER = ProviderRequestService.KEY_INTERNAL + "provider";

    public static final String KEY_PREFIX = "Bearer ";

    public static final String KEY_TOKEN = "token";

    protected ProviderRequestRewriter providerRequestRewriter;

    protected WorkflowConfigService workflowConfigService;

    protected McpDimensionService mcpDimensionService;

    protected McpRewriteService mcpRewriteService;

    protected McpTriggerService mcpTriggerService;

    protected McpClientService mcpClientService;

    protected LLMPromptService llmPromptService;

    protected ProviderToken providerToken;

    protected NamesService namesService;

    protected HistoryStore historyStore;

    protected Integer upstreamTimeout;

    protected Integer funCallTimeout;

    protected Integer funCallWaiting;

    // LLM请求失败时自动Dump的目录
    protected String autoDump;

    abstract protected T build() throws Exception;

    public T config(LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        T request = this.build();
        // 配置LLM请求
        this.request(request, llmConfig, llmQuery);
        // 配置多论会话记忆
        this.buildHistory(request, llmConfig, llmQuery);
        this.buildPrompt(request, llmConfig, request.getMessage());
        if (log.isDebugEnabled()) {
            log.debug("The request took {} milliseconds to fetch histories", llmQuery.getConsuming());
        }
        this.checkCompleted(request, llmConfig, llmQuery);
        this.providerRequestRewriter.rewrite(request, llmConfig, llmQuery);
        ProviderRequestChecker.check(request);
        return request;
    }

    protected void prepare(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // LLM请求失败时自动Dump的目录
        request.setAutoDump(this.buildAutodump(request, llmConfig, llmQuery));
        // 加载System Prompt/Rag并更新LLMQuery（需要放第一）
        request.setMessage(Message.build(llmQuery));
    }

    // 切换模型时FunCall不兼容时的处理
    protected void discard(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        if (request.getRecallFunCall() && request.getDiscard() && !CollectionUtils.isEmpty(request.getMessage().getHistories())) {
            Iterator<History> iterator = request.getMessage().getHistories().iterator();
            while (iterator.hasNext()) {
                History history = iterator.next();
                // 过滤非FUN CHAT且API不是默认，也不兼容则过滤
                if (!history.isFunction(History.FUN_CHAT) && !StringUtils.equalsIgnoreCase(history.getApi(), ProviderRequest.REQUEST_DEF) && !StringUtils.equalsIgnoreCase(history.getApi(), request.getApi())) {
                    iterator.remove();
                    if (log.isDebugEnabled()) {
                        log.debug("The request FunCall={}/{} is incompatible and has been removed.", history.getApi(), request.getApi());
                    }
                }
            }
        }
    }

    protected void request(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        this.prepare(request, llmConfig, llmQuery);
        if (log.isDebugEnabled()) {
            log.debug("The request took {} milliseconds to prepare rags", llmQuery.getConsuming());
        }
        // 加载Fun Call的配置
        request.setFunCallData(!MapUtils.isEmpty(llmQuery.getMetadata()) ? ProviderFunCallData.class.cast(llmQuery.getMetadata().get(ProviderRequestService.KEY_FUN_MERGE)) : null);
        request.setPrefix(llmConfig.hasDecoration() ? llmConfig.getDecoration().getPrefix() : "");
        request.setSuffix(llmConfig.hasDecoration() ? llmConfig.getDecoration().getSuffix() : "");
        // 下游FunCall调用超时
        request.setFunCallHeritage(this.buildFunCallHeritage(request, llmConfig, llmQuery));
        request.setUpstreamTimeout(this.buildUpstreamTimeout(request, llmConfig, llmQuery));
        request.setFunCallTimeout(this.buildFunCallTimeout(request, llmConfig, llmQuery));
        request.setRecallOffset(this.buildRecallOffset(request, llmConfig, llmQuery));
        request.setTimeout(this.buildTimeout(request, llmConfig, llmQuery));
        // 多论会话记忆恢复配置
        request.setContainHistories(this.buildContainHistories(request, llmConfig, llmQuery));
        request.setClientHistories(this.buildClientHistories(request, llmConfig, llmQuery));
        request.setClientDowngrade(this.buildClientDowngrade(request, llmConfig, llmQuery));
        request.setStoreQuery(this.buildStoreQuery(request, llmConfig, llmQuery));
        request.setDiscard(this.buildDiscard(request, llmConfig, llmQuery));
        request.setScene(this.buildScene(request, llmConfig, llmQuery));
        // 多论会话记忆保存配置
        request.setStoreCompleted(this.buildStoreCompleted(request, llmConfig, llmQuery));
        request.setRecallFunCall(this.buildRecallFunCall(request, llmConfig, llmQuery));
        request.setRepositories(this.buildRepositories(request, llmConfig, llmQuery));
        request.setStoreFunCall(this.buildStoreFunCall(request, llmConfig, llmQuery));
        request.setPrintReason(this.buildPrintReason(request, llmConfig, llmQuery));
        // 多论会话记忆可写配置
        request.setWriteable(this.buildWriteable(request, llmConfig, llmQuery));
        // 多论会话记忆记忆条数
        request.setHistories(this.buildHistories(request, llmConfig, llmQuery));
        // 截取最大错误长度
        request.setMaxError(this.buildMaxError(request, llmConfig, llmQuery));
        // 通知机制
        request.setNotifier(this.buildNotifier(request, llmConfig, llmQuery));
        // 多论会话记忆记忆时间
        request.setExpired(this.buildExpired(request, llmConfig, llmQuery));
        /////////////////////
        request.setMetadata(this.buildMetadata(request, llmConfig, llmQuery));
        // 加载Token和Model
        request.setToken(this.buildToken(request, llmConfig, llmQuery));
        request.setModel(this.buildModel(request, llmConfig, llmQuery));
        request.setUrl(this.buildUrl(request, llmConfig, llmQuery));
        // 多媒体请求
        request.setMediaContext(llmQuery.getMediaContext());
        request.setTokenBuffer(llmConfig.getTokenBuffer());
        request.setTokenFirst(llmConfig.getTokenFirst());
        request.setPureQuery(llmConfig.getPureQuery());
        request.setStream(llmConfig.getStream());
        request.setChain(llmConfig.getChain());
        // 加载Fun Call
        this.buildFunCall(request, llmConfig);
        if (log.isDebugEnabled()) {
            log.debug("The request took {} milliseconds to assemble", llmQuery.getConsuming());
        }
    }

    protected Boolean buildRecallFunCall(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getBoolean(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "recallFunCall", llmConfig.getRecallFunCall());
    }

    protected Integer buildFunCallTimeout(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // 如果没有Upstream且Deepness为1则使用FunCallWaiting（主线程等待）
        return MapUtils.getInteger(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "funCallTimeout", request.getMessage().isEntry() ? llmConfig.getFunCallWaiting(this.funCallWaiting) : llmConfig.getFunCallTimeout(this.funCallTimeout != null ? this.funCallTimeout : llmConfig.getTimeout()));
    }

    protected List<String> buildRepositories(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // 附加已经获得的Scene
        return List.class.cast(MapUtils.getObject(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "repositories", llmConfig.getRepositories()));
    }

    protected Boolean buildContainHistories(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getBoolean(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "containHistories", llmConfig.getContainHistories());
    }

    protected Boolean buildClientDowngrade(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getBoolean(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "clientDowngrade", llmConfig.getClientDowngrade());
    }

    protected Boolean buildClientHistories(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getBoolean(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "clientHistories", llmConfig.getClientHistories());
    }

    protected Boolean buildStoreCompleted(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getBoolean(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "storeCompleted", llmConfig.getStoreCompleted());
    }

    protected Integer buildUpstreamTimeout(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getInteger(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "upstreamTimeout", llmConfig.getUpstreamTimeout(this.upstreamTimeout != null ? this.upstreamTimeout : llmConfig.getTimeout()));
    }

    protected Boolean buildFunCallHeritage(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getBoolean(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "funCallHeritage", llmConfig.getFunCallHeritage());
    }

    protected Boolean buildStoreFunCall(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getBoolean(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "storeFunCall", llmConfig.getStoreFunCall());
    }

    protected Integer buildRecallOffset(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getInteger(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "recallOffset", llmConfig.getRecallOffset());
    }

    protected Boolean buildPrintReason(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getBoolean(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "printReason", llmConfig.getPrintReason());
    }

    protected Integer buildRecallNums(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getInteger(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "recallNums", llmConfig.getRecallNums());
    }

    protected Boolean buildStoreQuery(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getBoolean(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "storeQuery", llmConfig.getStoreQuery());
    }

    protected Boolean buildWriteable(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getBoolean(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "writeable", llmConfig.getWriteable());
    }

    protected Integer buildHistories(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getInteger(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "histories", llmConfig.getHistories());
    }

    protected Integer buildMaxError(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getInteger(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "maxError", llmConfig.getMaxError());
    }

    protected String buildNotifier(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getString(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "notifier", llmConfig.getNotifier());
    }

    protected Integer buildExpired(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getInteger(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "expired", llmConfig.getExpired());
    }

    protected Boolean buildDiscard(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getBoolean(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "discard", llmConfig.getDiscard());
    }

    protected Integer buildTimeout(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getInteger(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "timeout", llmConfig.getTimeout());
    }

    protected String buildAutodump(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // 需要先开启autodump
        return !StringUtils.isEmpty(this.autoDump) ? MapUtils.getString(llmQuery.getMetadata(), "__autodump", this.autoDump) : this.autoDump;
    }

    // LLMConfig -> WorkflowTask -> Default
    protected String buildModel(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return StringUtils.defaultIfEmpty(StringUtils.defaultIfEmpty(StringUtils.defaultIfEmpty(MapUtils.getString(llmConfig.getAdditional(), ProviderRequestService.KEY_MODEL), MapUtils.getString(llmQuery.getMetadata(), "__model")), this.defModel(request.getMessage())), "");
    }

    // WorkflowTask -> LLMConfig -> Default
    protected String buildToken(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        String token = MapUtils.getString(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN, MapUtils.getString(llmConfig.getAdditional(), ProviderRequestService.KEY_TOKEN));
        return StringUtils.defaultIfEmpty(this.providerToken.select(request, llmConfig, llmQuery, StringUtils.defaultIfEmpty(token, this.defToken(request.getMessage()))), "");
    }

    protected String buildScene(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getString(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "scene", llmConfig.getScene(llmQuery.getWorkflow()));
    }

    protected void buildPrompt(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // 包括Rag Chian
        String prompt = MapUtils.getString(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "prompt");
        prompt = prompt != null ? prompt : this.llmPromptService.prompt(request, llmConfig, llmQuery);
        request.setPrompt(prompt);
    }

    protected String buildUrl(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        return MapUtils.getString(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + "url");
    }

    protected void buildHistory(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // 恢复内部记忆
        this.internalHistory(request, llmConfig, llmQuery);
        // 外部（MCP）记忆
        this.externalHistory(request, llmConfig, llmQuery);
        // 独立存储Request
        this.storeHistoryQuery(request, llmConfig, llmQuery);
        // 丢弃不兼容记忆
        this.discard(request, llmConfig, llmQuery);
    }

    protected void buildFunCall(T request, LLMConfig llmConfig) throws Exception {
        // 内部Fun Call
        List<?> internalFunCall = request.getMessage().getMetadata(ProviderRequestService.KEY_FUN_INTERNAL, List.class);
        if (!CollectionUtils.isEmpty(internalFunCall)) {
            this.addInternalFunCall(request, llmConfig, Arrays.asList(JsonUtils.transfer(internalFunCall, LLMFunCall[].class)));
        } else {
            // 否则加载Config的Fun Call
            List<LLMFunCall> funCalls = llmConfig.getFunCalls();
            if (CollectionUtils.isEmpty(funCalls)) {
                return;
            }
            this.recallFunCall(request, llmConfig, funCalls);
        }
    }

    protected McpDimension buildMcpDimension(T request, LLMConfig llmConfig) throws Exception {
        return this.buildMcpDimension(request, llmConfig, LLMMcpCall.class.cast(null));
    }

    protected List<HistoryPair> buildHistoryQuery(T request, LLMConfig llmConfig) throws Exception {
        HistoryPair history = new HistoryPair(request.getMessage(), request.getMessage().getCreated() + 1);
        history.setQuery(request.getQuery4History());
        history.setModel(request.getModel());
        history.setApi(request.getApi());
        return List.of(history);
    }

    protected McpRuntime buildMcpRuntime(T request, LLMConfig llmConfig, LLMMcpCall llmMcpCall) throws Exception {
        return McpRuntime.builder().dynamic(llmMcpCall.getDynamic()).timeout(llmMcpCall.getTimeout()).workTask(request.getMessage()).build();
    }

    protected McpDimension buildMcpDimension(T request, LLMConfig llmConfig, LLMMcpCall llmMcpCall) throws Exception {
        McpDimension mcpDimension = McpDimension.builder().device(request.getMessage().getUserContext().getDevice()).workflow(request.getMessage().getWorkflow()).chat(request.getMessage().getChat()).biz(request.getMessage().getBiz()).mcpConfig(llmMcpCall).build();
        return this.buildMcpDimension(request, llmConfig, mcpDimension);
    }

    protected McpDimension buildMcpDimension(T request, LLMConfig llmConfig, McpDimension mcpDimension) throws Exception {
        Assert.notNull(this.mcpDimensionService, "The mcp dimensionService can not be empty, please config `mcp.enable`");
        return this.mcpDimensionService.buildDimension(mcpDimension, request.getMessage());
    }

    // 加载Metadata
    protected Map<String, Object> buildMetadata(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        if (llmConfig.getBridged()) {
            // 如果开启了Right Open AI桥接则ReWrite Metadata
            llmQuery.putMetadata("userContext", llmQuery.getUserContext());
            llmQuery.putMetadata("chat", llmQuery.getChat());
            return llmQuery.getMetadata();
        } else {
            // Refer
            return llmQuery.getMetadata();
        }
    }

    // 内部Fun Call
    protected void addInternalFunCall(T request, LLMConfig llmConfig, List<LLMFunCall> funCalls) throws Exception {
        List<ProviderFunCall> replaceFunCalls = new ArrayList<ProviderFunCall>();
        for (LLMFunCall funCall : funCalls) {
            // 有Refer且允许曝光
            if (!funCall.getRefer()) {
                replaceFunCalls.add(funCall);
            } else if (log.isWarnEnabled()) {
                log.warn("Internal Fun Call can not support MCP");
            }
        }
        if (!CollectionUtils.isEmpty(replaceFunCalls)) {
            request.setFunCalls(replaceFunCalls);
        }
    }

    // 召回并配置FunCall
    protected void recallFunCall(T request, LLMConfig llmConfig, List<LLMFunCall> funCalls) throws Exception {
        List<ProviderFunCall> replaceFunCalls = new ArrayList<ProviderFunCall>();
        for (LLMFunCall funCall : funCalls) {
            // 有Refer且允许曝光
            if (funCall.getRefer()) {
                // MCP型Fun Call
                List<ProviderFunCall> mcpFunCalls = this.recallMcpFunCalls(request, llmConfig, funCall);
                if (!CollectionUtils.isEmpty(mcpFunCalls)) {
                    replaceFunCalls.addAll(mcpFunCalls);
                }
            } else {
                // Workflow型Fun Call的Allowed入参为BIZ@WORKFLOW
                LLMFunCall llmFunCall = this.recallWorkflowFunCall(request, llmConfig, funCall);
                // 允许曝光
                if (llmFunCall != null) {
                    replaceFunCalls.add(llmFunCall);
                }
            }
        }
        if (log.isDebugEnabled()) {
            log.debug("ReCall Fun Call={}", replaceFunCalls.size());
        }
        // 追加FunCall(Upsert)
        request.setFunCalls(replaceFunCalls);
    }

    // 召回并配置Workflow型FunCall
    protected LLMFunCall recallWorkflowFunCall(T request, LLMConfig llmConfig, LLMFunCall funCall) throws Exception {
        try {
            // 查找对应Workflow的配置
            String[] part = SplitUtils.split(funCall.getName(), request.getMessage().getBiz());
            WorkflowConfig workflowConfig = this.workflowConfigService.config(part[0], part[1]);
            // 如果配置了FunCall则merge
            LLMFunCall llmFunCall = workflowConfig.getLlmFunCall();
            if (llmFunCall != null) {
                // 过滤指定Workflow FunCall
                String name = SplitUtils.join(funCall.getName(), request.getMessage().getBiz());
                if (llmConfig.hasMcpCall() && !llmConfig.getMcpCall().allowed(name)) {
                    if (log.isInfoEnabled()) {
                        log.info("Workflow fun call is filtered by the mcp configuration, name={}", name);
                    }
                    return null;
                }
                // 过滤回路
                if (llmFunCall.isLooped(request.getMessage().getBiz(), request.getMessage().getWorkflow())) {
                    if (log.isInfoEnabled()) {
                        log.info("Workflow fun call is filtered by the loop operation");
                    }
                    return null;
                }
                // 从Workflow读取配置，并使用当前FunCall覆盖
                funCall = llmFunCall.merge(funCall);
                funCall.setDescription((funCall.hasPrefix() ? StringUtils.defaultIfEmpty(funCall.getPrefix(), "") : "") + StringUtils.defaultIfEmpty(funCall.getDescription(), "") + (funCall.hasSuffix() ? StringUtils.defaultIfEmpty(funCall.getSuffix(), "") : ""));
            }
            // 编码名称
            funCall.setName(this.encodeWorkflow(request, llmConfig, funCall.getName(), request.getMessage().getBiz()));
            // 追加TakeOver配置
            if (funCall.hasTakeover()) {
                request.addTakeover(funCall.getName(), funCall.getTakeover());
            }
            return funCall;
        } catch (Exception e) {
            WorkflowException.dolog(e);
            // 任意异常返回Null，由调用方从Tools列表去除
            return null;
        }
    }

    // 召回并配置MCP型FunCall
    protected List<ProviderFunCall> recallMcpFunCalls(T request, LLMConfig llmConfig, LLMFunCall funCall) throws Exception {
        try {
            // Mcp型Fun Call
            Assert.notNull(this.mcpClientService, "The mcp client can not be empty, please config `mcp enable`");
            List<ProviderFunCall> mcpFunCalls = this.mcpClientService.toolsList(funCall.getName(), this.buildMcpDimension(request, llmConfig));
            Assert.notEmpty(mcpFunCalls, "Mcp tools can not be empty");
            Iterator<ProviderFunCall> mcpIterator = mcpFunCalls.iterator();
            while (mcpIterator.hasNext()) {
                ProviderFunCall each = mcpIterator.next();
                // 解码检查
                String name = this.namesService.decode(each.getName())[1];
                if (funCall.allowed(name)) {
                    // 符合则Rewrite MCP配置
                    this.reConfigMcpFunCall(each, llmConfig, funCall, name);
                } else {
                    // 不符合则删除
                    if (log.isInfoEnabled()) {
                        // 不符合则删除
                        log.info("Mcp Fun Call can not be allowed={}", each.getName());
                    }
                    mcpIterator.remove();
                }
            }
            return mcpFunCalls;
        } catch (Exception e) {
            WorkflowException.dolog(e);
            return null;
        }
    }

    // 对MCP型Fun Call进行重配置
    protected void reConfigMcpFunCall(ProviderFunCall mcpCall, LLMConfig llmConfig, LLMFunCall funCall, String name) throws Exception {
        // Rewrite Description
        mcpCall.setDescription(funCall.hasDescriptions(name) ? funCall.getDescriptions().get(name) : mcpCall.getDescription());
        mcpCall.setDescription((funCall.hasPrefix() ? funCall.getPrefix() : "") + mcpCall.getDescription() + (funCall.hasSuffix() ? funCall.getSuffix() : ""));
        // Rewrite Properties
        if (funCall.hasProperties(name)) {
            mcpCall.setProperties(Map.class.cast(funCall.getProperties().get(name)));
        }
    }

    // 对Workflow型Fun Call编码
    protected String encodeWorkflow(T request, LLMConfig llmConfig, String name, String biz) throws Exception {
        String[] pair = SplitUtils.split(name, biz);
        return this.namesService.encode(NamesService.PREFIX_WORKFLOW, pair[0], pair[1]);
    }

    // 恢复MCP记忆
    protected void externalHistory(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        if (llmConfig.hasMcpCall() && llmConfig.getMcpCall().hasClient()) {
            LLMMcpCall llmMcpCall = llmConfig.getMcpCall();
            McpDimension mcpDimension = this.buildMcpDimension(request, llmConfig, llmConfig.getMcpCall());
            // 触发MCP Prompt Get
            Map<String, Object> arguments = llmMcpCall.arguments(request.getMessage().getQuery());
            Assert.notNull(this.mcpTriggerService, "The mcp trigger can not be empty, please config `mcp.enable`");
            this.mcpTriggerService.beforePromptGet(mcpDimension, arguments, request.getMessage());
            Assert.notNull(this.mcpClientService, "The mcp client can not be empty, please config `mcp enable`");
            McpResult<List<History>> result = this.mcpClientService.promptGet(mcpDimension.getClient(), mcpDimension.getName(), arguments, this.buildMcpRuntime(request, llmConfig, llmMcpCall), mcpDimension);
            // MCP Listener Rewrite
            Assert.notNull(this.mcpRewriteService, "The mcp rewrite can not be empty, please config `mcp.enable`");
            List<History> histories = this.mcpRewriteService.promptGet(mcpDimension, arguments, request.getMessage(), result).getResult();
            if (!CollectionUtils.isEmpty(histories)) {
                // 当Replace=True时，开启MCP Prompt返回结果仅一条记忆且记忆Role=User则替换Query
                if (llmMcpCall.getReplace() && histories.size() == 1 && histories.getFirst().isRole(History.ROLE_USER)) {
                    String query = histories.getFirst().getContent();
                    if (log.isInfoEnabled()) {
                        log.info("Replace query from mcp={}", query);
                    }
                    request.getMessage().setQuery(query);
                } else {
                    if (log.isInfoEnabled()) {
                        log.info("Append history from mcp={}", histories.size());
                    }
                    request.getMessage().addHistories(histories);
                }
            }
        }
    }

    // 从内部记忆恢复
    protected void internalHistory(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // 备份
        List<History> clientHistories = request.getClientDowngrade() ? History.getReferenceHistory(request.getMessage().getHistories(), History.REFERENCE_CLIENT) : null;
        if (!request.getClientHistories()) {
            // 清空调用端记忆
            request.getMessage().delHistories();
            if (log.isDebugEnabled()) {
                log.debug("Cleaning client-side history");
            }
        }
        if (request.getContainHistories()) {
            // 开启多轮会话（调用端记忆优先）
            // Offset为时间偏移，如果为正数表示过去时间（遭遇当前时间），负数表示未来时间（会被过滤）
            // Timestamp 最终为负数，数值越大（越接近 0）表示消息越旧，越负表示越新
            // 如果当前时间是T，end传入为-(T-90秒)，desc=false，取过去 90 秒到现在的数据
            Assert.notNull(this.historyStore, "The history store can not be empty, please config `history.enable`");
            long lastTime = request.hasRecallOffset() ? request.getMessage().getCreated() - TimeUnit.MILLISECONDS.convert(request.getRecallOffset(), TimeUnit.SECONDS) : 0;
            List<History> histories = new ArrayList<History>(this.historyStore.restore(request.getMessage(), request.getScene(), this.buildRecallNums(request, llmConfig, llmQuery), llmConfig.getRecallDesc(), -request.getMessage().getCreated(), -lastTime));
            // 如果Histories为空，并且开启了ClientDowngrade，并且端侧记忆不为空，则替换
            histories = !CollectionUtils.isEmpty(histories) ? histories : (request.getClientDowngrade() && !CollectionUtils.isEmpty(clientHistories) ? clientHistories : histories);
            if (!CollectionUtils.isEmpty(histories)) {
                this.discardHistory(request, llmConfig, histories);
            }
            request.getMessage().replaceHistories(histories);
            if (log.isDebugEnabled()) {
                log.debug("History size={}", histories.size());
            }
        }
    }

    protected void discardHistory(T request, LLMConfig llmConfig, List<History> histories) throws Exception {
        Iterator<History> iterator = histories.iterator();
        while (iterator.hasNext()) {
            History history = iterator.next();
            // 剔除Function类型，只保留非FunCall的会话
            if (!request.getRecallFunCall() && history.isFunction(History.FUN_FUNCALL)) {
                iterator.remove();
                continue;
            }
            // 创建时间在Message.timestamp之后的（一般已经存在与FunCall，避免重复加载）
            // 用于指定了负数Offset且使用Desc=True（获取过去时间之后的历史记录）与FunCall重叠部分
            if (history.getCreated() >= request.getMessage().getCreated()) {
                iterator.remove();
                continue;
            }
        }
    }

    // 如果需要单独存储Request，逻辑需要与ProviderStream一致
    protected void storeHistoryQuery(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        if (ProviderRequestService.shouldStoreHistoryQuery(request, llmConfig)) {
            Assert.notNull(this.historyStore, "The history store can not be empty, please config `history.enable`");
            this.historyStore.store(request.getMessage(), request.getRepositories(), this.buildHistoryQuery(request, llmConfig), request.getExpired(), request.getHistories());
        }
    }

    protected void checkCompleted(T request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        Assert.hasText(request.getToken(), "The token can not be empty");
    }

    protected String defToken(WorkflowTask workTask) throws Exception {
        return "";
    }

    protected String defModel(WorkflowTask workTask) throws Exception {
        return "";
    }

    public static Boolean shouldStoreHistoryQuery(ProviderRequest request, LLMConfig llmConfig) throws Exception {
        return request.getStoreQuery() && ((request.getContainHistories() && request.isWriteable() && !request.getMessage().isFromFunCall()) && !request.getStoreCompleted());
    }

    public static Boolean isFromFunMerge(Map<String, Object> metadata) {
        return MapUtils.getObject(metadata, ProviderRequestService.KEY_FUN_MERGE) != null;
    }

    public static Boolean isFromFunCall(Map<String, Object> metadata) {
        return MapUtils.getObject(metadata, ProviderRequestService.KEY_FUN_FETCH) != null || ProviderRequestService.isFromFunMerge(metadata);
    }

    public static class ProviderRequestChecker {

        public static void check(ProviderRequest request) {
            Assert.notNull(request.getContainHistories(), "ContainHistories can not be empty");
            Assert.notNull(request.getTokenBuffer(), "Token Buffer can not be empty");
            Assert.notNull(request.getTokenFirst(), "Token First can not be empty");
            Assert.notNull(request.getMessage(), "Message can not be empty");
            Assert.notNull(request.getStream(), "Stream can not be empty");
            Assert.hasText(request.getToken(), "Token can not be empty");
            Message.MessageChecker.check(request.getMessage());
        }
    }

    @Setter
    @Getter
    public static class ProviderRequestInitConfig {

        @Autowired
        protected ProviderRequestRewriter providerRequestRewriter;

        @Autowired
        protected WorkflowConfigService workflowConfigService;

        @Autowired(required = false)
        protected McpDimensionService mcpDimensionService;

        @Autowired(required = false)
        protected McpRewriteService mcpRewriteService;

        @Autowired(required = false)
        protected McpTriggerService mcpTriggerService;

        @Autowired(required = false)
        protected McpClientService mcpClientService;

        @Autowired
        protected LLMPromptService llmPromptService;

        @Autowired
        protected ProviderToken providerToken;

        @Autowired
        protected NamesService namesService;

        @Autowired(required = false)
        protected HistoryStore historyStore;

        @Value("${upstream.timeout:}")
        protected Integer upstreamTimeout;

        // 主线程等待FunCall时的超时，默认等于FunCallWaiting
        // 当UpStream为空时且Deepness = 1时表示主线程
        @Value("${funcall.waiting:}")
        protected Integer funCallWaiting;

        @Value("${funcall.timeout:}")
        protected Integer funCallTimeout;

        @Value("${request.timeout:60000}")
        // LLM Socket Timeout默认时间
        protected Integer timeout;

        @Value("${autodump.llm:}")
        // LLM请求失败时自动Dump的目录
        protected String autoDump;
    }
}
