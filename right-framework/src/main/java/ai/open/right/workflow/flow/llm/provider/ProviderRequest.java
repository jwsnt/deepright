package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.DumpUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.LLMFunCallData;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.LLMTakeover;
import ai.open.right.workflow.flow.media.MediaContext;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.CollectionUtils;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

@ToString
@Slf4j
@Setter
@Getter
public class ProviderRequest {

    public static final String REQUEST_ANTHROPIC = "anthropic";

    public static final String REQUEST_SEEDREAM = "seedream";

    public static final String REQUEST_GOOGLE = "google";

    public static final String REQUEST_OPENAI = "openai";

    public static final String REQUEST_COZE = "coze";

    // 默认，兼容
    public static final String REQUEST_DEF = "def";

    private final ProviderData providerData = new ProviderData();

    // 接管型Workflow FunCall
    protected Map<String, LLMTakeover> takeovers;

    protected Map<String, Object> extendedConfig;

    protected List<MediaContext> mediaContext;

    protected List<ProviderFunCall> funCalls;

    // 补充协议
    protected Map<String, Object> extraBody;

    protected Map<String, Object> metadata;

    protected LLMFunCallData funCallData;

    protected List<String> repositories;

    protected Boolean containHistories = false;

    protected Boolean clientDowngrade = true;

    protected Boolean clientHistories = false;

    // FunCall是否共享Metadata
    protected Boolean funCallHeritage = false;

    protected Boolean storeCompleted = true;

    protected Boolean recallFunCall = true;

    protected Boolean storeFunCall = false;

    protected Boolean printReason = false;

    protected Integer upstreamTimeout;

    protected Integer funCallTimeout;

    protected Integer recallOffset;

    protected Integer tokenBuffer;

    protected Integer tokenFirst;

    protected Integer histories;

    protected Boolean storeQuery = true;

    protected Boolean pureQuery = true;

    protected Boolean writeable;

    protected Integer maxError;

    protected Integer expired;

    // 是否兼容性丢弃
    protected Boolean discard;

    protected Integer timeout;

    protected Message message;

    protected String notifier;

    protected String autoDump;

    protected Boolean stream;

    protected String prefix = "";

    protected String suffix = "";

    protected String prompt;

    // 使用的模型
    protected String model;

    protected String scene;

    protected String chain;

    protected String token;

    protected String url;

    // API类型
    protected String api;

    public Boolean hasChain() {
        return !StringUtils.isEmpty(this.chain);
    }

    public Boolean isWriteable() {
        return (this.containHistories != null && this.containHistories) && (this.writeable != null ? this.writeable : true);
    }

    public void setFunCalls(List<ProviderFunCall> funCalls) {
        if (!CollectionUtils.isEmpty(funCalls)) {
            if (this.funCalls != null) {
                this.funCalls.addAll(funCalls);
            } else {
                this.funCalls = funCalls;
            }
        } else {
            if (log.isInfoEnabled()) {
                log.info("Skip empty Fun Calls");
            }
        }
    }

    public void setExtra(String key, Object value) {
        this.extraBody = this.extraBody != null ? this.extraBody : new HashMap<String, Object>();
        this.extraBody.put(key, value);
    }

    // 子类复写
    public Map<String, Object> getResponseSchema() {
        return null;
    }

    public List<String> getRepositories() {
        return LLMConfig.buildRepositories(this.repositories, this.getScene());
    }

    // Query会在执行过程中被修改（如Rag），如pureQuery=True则使用原始Query，否则使用更新后的Query
    public String getQuery4History() {
        return this.pureQuery ? this.message.getInitial() : this.message.getQuery();
    }

    public Boolean hasRecallOffset() {
        return this.recallOffset != null;
    }

    public Boolean hasMimeContext() {
        return !CollectionUtils.isEmpty(this.mediaContext);
    }

    public Boolean hasFunCallData() {
        return this.funCallData != null;
    }

    public Boolean hasFunCall() {
        return !CollectionUtils.isEmpty(this.funCalls);
    }

    public Boolean hasAutoDump() {
        return !StringUtils.isEmpty(this.autoDump);
    }

    public Boolean hasNotifier() {
        return !StringUtils.isEmpty(this.notifier);
    }

    public String getNotifier(String notifier) {
        return !StringUtils.isEmpty(this.notifier) ? this.notifier : notifier;
    }

    public void addTakeover(String name, LLMTakeover llmTakeover) {
        if (this.takeovers == null) {
            this.takeovers = new HashMap<String, LLMTakeover>();
        }
        this.takeovers.put(name, llmTakeover);
    }

    public LLMTakeover getTakeover(String name) {
        return LLMTakeover.class.cast(MapUtils.getObject(this.takeovers, name));
    }

    public Boolean isTakeover(String name) {
        return MapUtils.getObject(this.takeovers, name) != null;
    }

    public Boolean isApi(String name) {
        return StringUtils.equalsIgnoreCase(this.api, name);
    }

    public void appendResponse(String response) throws Exception {
        this.providerData.appendResponse(response);
    }

    public void appendRequest(String request) throws Exception {
        this.providerData.appendRequest(request);
    }

    public ProviderRequest putExtended(String key, Object value) {
        this.extendedConfig = this.extendedConfig != null ? this.extendedConfig : new HashMap<String, Object>();
        this.extendedConfig.put(key, value);
        return this;
    }

    public <T> T delExtended(String key, Class<T> clazz) {
        return this.extendedConfig != null ? clazz.cast(this.extendedConfig.remove(key)) : null;
    }

    public void autoDump(WorkflowException e) {
        try {
            // 400（请求错误） 914（报文残缺）MAX_TOKEN 或 开启全局或请求级别 autodump
            if ((ProtocolCode.C400.equals(e.getCode()) || (ProtocolCode.C914.equals(e.getCode())) || StringUtils.containsIgnoreCase(e.getMessage(), WorkflowException.MAX_TOKEN) || (this.hasAutoDump() && e.getCode() > ProtocolCode.C0))) {
                String dir = StringUtils.defaultIfEmpty(this.autoDump, "autodump");
                String res = StringUtils.join(new String[]{this.getModel(), String.valueOf(e.getCode()), "response"}, "_") + ".json";
                DumpUtils.dump(this.getMessage(), dir, res, JsonUtils.write(this.providerData.getResponse()));
                String req = StringUtils.join(new String[]{this.getModel(), String.valueOf(e.getCode()), "request"}, "_") + ".json";
                DumpUtils.dump(this.getMessage(), dir, req, JsonUtils.write(this.providerData.getRequest()));
                if (log.isInfoEnabled()) {
                    log.info("The exception autodump has been saved in dir={}, request={}, response={}", dir, req, res);
                }
            }
        } catch (Exception ex) {
            WorkflowException.dolog(ex);
        }
    }

    public void autoDump() {
        try {
            if (this.hasAutoDump()) {
                String dir = StringUtils.defaultIfEmpty(this.autoDump, "autodump");
                String res = StringUtils.join(new String[]{this.getModel(), "response"}, "_") + ".json";
                DumpUtils.dump(this.getMessage(), dir, res, JsonUtils.write(this.providerData.getResponse()));
                String req = StringUtils.join(new String[]{this.getModel(), "request"}, "_") + ".json";
                DumpUtils.dump(this.getMessage(), dir, req, JsonUtils.write(this.providerData.getRequest()));
                if (log.isInfoEnabled()) {
                    log.info("The success autodump has been saved in dir={}, request={}, response={}", dir, req, res);
                }
            }
        } catch (Exception ex) {
            WorkflowException.dolog(ex);
        }
    }
}
