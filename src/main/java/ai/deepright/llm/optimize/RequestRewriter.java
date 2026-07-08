package ai.deepright.llm.optimize;

import static org.springframework.util.ObjectUtils.isEmpty;

import ai.deepright.cli.CliPrinter;
import ai.deepright.cli.CliSubFetcher;
import ai.deepright.complex.ComplexityMode;
import ai.deepright.complex.ComplexityUtils;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.lang.XmlResourceLang;
import ai.deepright.llm.provider.RequestContextUtils;
import ai.deepright.llm.provider.RequestFunCallStore;
import ai.deepright.memory.MemoryService;
import ai.open.right.WorkflowException;
import ai.open.right.utils.BytesUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.*;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestRewriter;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.summary.SummaryConfig;
import ai.open.right.workflow.flow.summary.SummaryPart;
import ai.open.right.workflow.flow.summary.SummaryService;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierService;
import com.google.common.collect.ImmutableMap;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.*;
import java.util.stream.Collectors;

@Slf4j
@Getter
@Setter
// 压缩策略
public class RequestRewriter implements ProviderRequestRewriter {

    public static final String LANG_KEY_REWRITE_MESSAGE = "rewrite.message";

    public static final String NAME = "request_rewriter";

    public static final String KEY = "__summary__";

    protected Map<String, FunCallCompressor> funCallCompressor;

    protected NotifierService notifierService;

    protected SummaryService summaryService;

    protected RequestDiscard requestDiscard;

    protected RequestFunCall requestFunCall;

    protected MemoryService memoryService;

    protected CliSubFetcher cliSubFetcher;

    protected HistoryStore historyStore;

    protected Double funCallRemove4rate;

    protected Integer funCallSafePoint;

    protected Integer funCallLargeData;

    protected Double funCall4oversize;

    protected Double histories4oversize;

    protected Integer histories4offset;

    protected Double histories4rate;

    protected Boolean rewriterHistories;

    protected Boolean rewriterFunCall;

    protected Boolean rewriterBase64;

    protected Double dropOnFailed;

    @Override
    public void rewrite(ProviderRequest providerRequest, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // 始终需要调用
        this.requestFunCall.allowed(providerRequest, llmConfig, llmQuery);
        this.requestDiscard.discard(providerRequest, llmConfig, llmQuery);
        // LLMConfig配置当前节点是否需要总结，默认True（Summary节点及特殊节点需要跳过）
        if (this.shouldRewrite(providerRequest, llmConfig, llmQuery)) {
            // 先FunCall，可能会添加额外History
            this.compressFunCall(providerRequest, llmConfig, llmQuery);
            this.compressHistory(providerRequest, llmConfig, llmQuery);
        } else if (log.isDebugEnabled()) {
            log.debug("The request will skip compression={}", SplitUtils.join(providerRequest.getMessage()));
        }
    }

    protected Boolean shouldRewrite(ProviderRequest providerRequest, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // 是否要触发重写
        // Global 优先，Global 未配置时再用 Metadata 都未配置则为 true
        return MapUtils.getBooleanValue(llmConfig.getGlobalConfig(), RequestRewriter.KEY, MapUtils.getBoolean(llmQuery.getMetadata(), RequestRewriter.KEY, true));
    }

    protected void compressHistory(ProviderRequest providerRequest, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // 开启持久化，且存在上下文，且超过压缩限制
        List<History> histories = providerRequest.getMessage().getHistories();
        if (this.rewriterHistories && llmConfig.getContainHistories() && !CollectionUtils.isEmpty(histories) && (histories.size() > (Double.valueOf(llmConfig.getHistories() * this.histories4rate)).intValue())) {
            // 超过需要压缩的大小
            int bytes = BytesUtils.utf8Bytes((JsonUtils.write(histories)));
            // 当前上下文（总量）与模型允许上下文（总量）的比例上限
            if ((bytes / (double) RequestContextUtils.limit(providerRequest.getMessage(), providerRequest.getModel())) > this.histories4oversize) {
                SummaryHistories summaryHistories = this.buildHistories(providerRequest, llmConfig);
                if (summaryHistories.shouldSummary()) {
                    this.notify(providerRequest.getMessage());
                    // 有先后顺序
                    this.summaryMemory(providerRequest, llmConfig, llmQuery, summaryHistories.getChat());
                    this.summaryHistory(providerRequest, llmConfig, llmQuery, summaryHistories.getTotal(), bytes);
                }
            }
        }
    }

    protected void summaryHistory(ProviderRequest providerRequest, LLMConfig llmConfig, LLMQuery llmQuery, List<History> histories, int bytes) throws Exception {
        try {
            SummaryPart part = this.summaryService.summarize(this.buildSummaryConfig(providerRequest, llmConfig, bytes), providerRequest.getMessage(), histories);
            if (part != null && part.hasPairs()) {
                List<History> summary = new ArrayList<History>();
                for (HistoryPair pair : part.getPairs()) {
                    // 压缩，取出归纳为Null的上下文
                    summary.addAll(Arrays.stream(pair.buildHistories())
                            .filter(Objects::nonNull)
                            .toList());
                }
                providerRequest.getMessage().setHistories(summary);
                if (log.isInfoEnabled()) {
                    log.info("The request was compressed, device={}, history={}, summary={}", providerRequest.getMessage().getDevice(), histories.size(), summary.size());
                }
            } else if (log.isWarnEnabled()) {
                // 空压缩，可能为Bug
                log.warn("The request was compressed empty, device={}", providerRequest.getMessage().getDevice());
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
    }

    protected void summaryMemory(ProviderRequest providerRequest, LLMConfig llmConfig, LLMQuery llmQuery, List<History> histories) throws Exception {
        try {
            this.memoryService.refresh(providerRequest.getMessage(), histories);
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
    }

    protected void compressFunCall(ProviderRequest providerRequest, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        LLMFunCallData funCallData = providerRequest.getFunCallData();
        if (funCallData != null) {
            funCallData = this.compressFunCalls(providerRequest, funCallData, llmConfig);
            // 循环裁剪FunCall直到满足指定大小
            // 首轮删除为0，每轮递增，直到最坏情况为删除完
            double removeRate = 0D;
            // 当前FunCall(总量)大于模型允许总量的比例上限
            while (this.rewriterFunCall && (BytesUtils.utf8Bytes(JsonUtils.write(funCallData)) / (double) RequestContextUtils.limit(providerRequest.getMessage(), providerRequest.getModel()) > this.funCall4oversize)) {
                try {
                    // 删除最旧s
                    funCallData = this.removeOldestCalls(providerRequest, funCallData, llmConfig, removeRate);
                    // 删除信息熵不在范围内的（不处理最近 funCallSafePoint 条）
                    funCallData = this.removeChaosCalls(providerRequest, funCallData, llmConfig);
                    removeRate += this.funCallRemove4rate;
                } catch (Exception e) {
                    WorkflowException.dolog(e);
                }
            }
            // 回写
            providerRequest.setFunCallData(funCallData);
            // 补充召回消息时间到[当前]第一条FunCall时间间的History（已丢弃数据的Reason）
            // @see RequestFunCallStore
            LLMFunCallRequest llmFunCallRequest = !CollectionUtils.isEmpty(funCallData.getRequests()) ? funCallData.getRequests().getFirst() : null;
            if (llmFunCallRequest != null) {
                List<History> histories = this.historyStore.restore(providerRequest.getMessage(), providerRequest.getScene(), llmConfig.getRecallNums(), llmConfig.getRecallDesc(), -llmFunCallRequest.getCreated(), -providerRequest.getMessage().getCreated());
                // 保留source相等的数据
                histories = histories.stream().filter(h -> StringUtils.equalsIgnoreCase(RequestFunCallStore.SOURCE, h.getSource())).collect(Collectors.toList());
                if (!CollectionUtils.isEmpty(histories)) {
                    providerRequest.getMessage().getHistories().addAll(histories);
                    if (log.isInfoEnabled()) {
                        log.info("The request will be append additional histories={}", histories.size());
                    }
                }
            }
        }
    }

    // 删除信息熵不在范围的FunCall
    protected LLMFunCallData removeChaosCalls(ProviderRequest providerRequest, LLMFunCallData funCallData, LLMConfig llmConfig) throws Exception {
        List<Integer> removed = new ArrayList<Integer>();
        // 倒序，防止索引错位，排除funCallSafePoint轮（最新轮）
        for (int index = funCallData.getRequests().size() - this.funCallSafePoint - 1; index >= 0; index--) {
            if (ComplexityUtils.score(JsonUtils.write(funCallData.getRequests().get(index).getRefer()) + funCallData.getResponses().get(index).getResponse()).is(ComplexityMode.FAST_REPLY)) {
                removed.add(index);
            }
        }
        if (log.isInfoEnabled()) {
            log.info("The request funcall call will be removed by entropy removed={} / total={}", removed.size(), funCallData.getRequests().size());
        }
        // 从大到小添加索引，迭代时也要相同（从大到小，防止索引错位）
        for (Integer index : removed) {
            funCallData.getResponses().remove((int) index);
            funCallData.getRequests().remove((int) index);
        }
        return funCallData;
    }

    // 按比例删除末尾
    protected LLMFunCallData removeOldestCalls(ProviderRequest providerRequest, LLMFunCallData funCallData, LLMConfig llmConfig, Double removeRate) throws Exception {
        // 需要删除的数量
        int remove = Double.valueOf(funCallData.getRequests().size() * removeRate).intValue();
        if (remove > 0) {
            // 删除最早出现的FunCall
            int close = funCallData.getRequests().size();
            if (log.isInfoEnabled()) {
                log.info("The request funcall will be removed, device={}, start={}, close={}", providerRequest.getMessage().getDevice(), remove, close);
            }
            if (close > remove) {
                // 保留末尾（最新）
                // subList 返回原 List 的视图，不是独立副本，需要new新List加速GC
                funCallData.setResponses(new ArrayList<LLMFunCallResponse>(funCallData.getResponses().subList(remove, close)));
                funCallData.setRequests(new ArrayList<LLMFunCallRequest>(funCallData.getRequests().subList(remove, close)));
            } else {
                // 完全清除
                funCallData.setResponses(new ArrayList<LLMFunCallResponse>());
                funCallData.setRequests(new ArrayList<LLMFunCallRequest>());
            }
        }
        return funCallData;
    }

    protected LLMFunCallData compressFunCalls(ProviderRequest providerRequest, LLMFunCallData funCallData, LLMConfig llmConfig) throws Exception {
        // 需要压缩的数量（全部，没有保护范围）
        for (int index = funCallData.getRequests().size() - 1; index >= 0; index--) {
            LLMFunCallResponse response = funCallData.getResponses().get(index);
            LLMFunCallRequest request = funCallData.getRequests().get(index);
            // 所有Funcall的大小检查
            if (BytesUtils.utf8Bytes(JsonUtils.write(ImmutableMap.of("req", request.getRefer(), "rep", response.getResponse()))) > this.funCallLargeData) {
                String key = FunCallCompressor.FLAG + providerRequest.getApi();
                // 获取不同服务商的压缩方法
                FunCallCompressor funCallCompressor = this.funCallCompressor.get(key);
                if (funCallCompressor != null) {
                    this.compressResponse(funCallCompressor, providerRequest, funCallData.getResponses().get(index));
                    this.compressRequest(funCallCompressor, providerRequest, funCallData.getRequests().get(index));
                } else if (log.isInfoEnabled()) {
                    log.info("The request funcall can not find compressor={}", key);
                }
            }
        }
        return funCallData;
    }

    protected void compressResponse(FunCallCompressor funCallCompressor, ProviderRequest providerRequest, LLMFunCallResponse funCall) throws Exception {
        funCallCompressor.compress(providerRequest, funCall);
    }

    protected void compressRequest(FunCallCompressor funCallCompressor, ProviderRequest providerRequest, LLMFunCallRequest funCall) throws Exception {
        funCallCompressor.compress(providerRequest, funCall);
    }

    protected SummaryHistories buildHistories(ProviderRequest providerRequest, LLMConfig llmConfig) throws Exception {
        List<LLMFunCallRequest> requests = providerRequest.hasFunCallData() ? providerRequest.getFunCallData().getRequests() : null;
        // Offset，如果有Funcall就取第一个（获取截止至Funcall），如果没有就取消息（截止至起始消息）
        Long lastTimestamp = !CollectionUtils.isEmpty(requests) ? requests.getFirst().getCreated() : providerRequest.getMessage().getCreated();
        // 召回消息时间到第一条FunCall时间间的History
        // 默认Desc=false
        return new SummaryHistories(this.historyStore.restore(providerRequest.getMessage(), providerRequest.getScene(), llmConfig.getRecallNums(), -lastTimestamp));
    }

    protected SummaryConfig buildSummaryConfig(ProviderRequest providerRequest, LLMConfig llmConfig, Integer bytes) throws Exception {
        SummaryConfig summaryConfig = new SummaryConfig();
        // 溢出临界值时启动DropOnFailed
        summaryConfig.setDropOnFailed((Double.valueOf(bytes) / RequestContextUtils.limit(providerRequest.getMessage(), providerRequest.getModel())) > this.dropOnFailed);
        // 指定需要召回上下文Scene和上下文数量
        summaryConfig.setMaxsize(Math.max(1, Double.valueOf(llmConfig.getHistories() * this.histories4rate).intValue()));
        // 需要召回的记忆池
        summaryConfig.setScene(providerRequest.getScene());
        // 转为MediaContext时使用Base64
        summaryConfig.setBase64(this.rewriterBase64);
        // 归纳调用的Workflow（summary.json）
        summaryConfig.setDynamic("summary@compress");
        // 压缩FunCall
        summaryConfig.setIncludeFunCall(true);
        // 直接更新用户上下文记忆（下次加载有效）
        summaryConfig.setStore(true);
        // 倒序归纳（更新最旧n条）
        summaryConfig.setDesc(true);
        return summaryConfig;
    }

    public void notify(WorkflowTask workTask) throws Exception {
        if (!FeatureFlag.isSilent(workTask)) {
            Segment.SegmentConfig segmentConfig = Segment.SegmentConfig.builder()
                    .content(new StringBuffer(XmlResourceLang.get(RequestRewriter.LANG_KEY_REWRITE_MESSAGE)))
                    .metadata(CliPrinter.process(RequestRewriter.KEY))
                    .workflow(workTask.getWorkflow())
                    .notifier(Notifier.SOURCE)
                    .build();
            this.notifierService.notify(Segment.build(workTask, segmentConfig), workTask, workTask);
        }
    }

    @Getter
    // SummaryHistories会对现有的上下文做过滤，然后LLM归纳后重新Clear->Store
    // 超长上下文或上下文复杂度会在过滤阶段被清理，然后通过Clear->Store清除
    public static class SummaryHistories {

        protected final List<History> total = new ArrayList<History>();

        protected final List<History> chat = new ArrayList<History>();

        public SummaryHistories(List<History> histories) throws Exception {
            for (History history : histories) {
                if (!ComplexityUtils.score(history.getContent() + (!history.isEncrypt() ? history.getReasoning() : "")).is(ComplexityMode.FAST_REPLY)) {
                    if (history.isFunction(History.FUN_CHAT)) {
                        this.chat.add(history);
                    }
                    this.total.add(history);
                } else if (log.isDebugEnabled()) {
                    log.debug("The summary history will be ignored for complexity");
                }
            }
        }

        public Boolean shouldSummary() {
            return !CollectionUtils.isEmpty(this.total);
        }
    }

    @Configuration
    @Getter
    @Setter
    public static class InitConfig {

        @Autowired
        protected Map<String, FunCallCompressor> funCallCompressor;

        @Autowired
        protected NotifierService notifierService;

        @Autowired
        protected SummaryService summaryService;

        @Autowired
        protected RequestDiscard requestDiscard;

        @Autowired
        protected RequestFunCall requestFunCall;

        @Autowired
        @Qualifier(MemoryService.NAME)
        protected MemoryService memoryService;

        @Autowired
        protected CliSubFetcher cliSubFetcher;

        @Autowired
        protected HistoryStore historyStore;

        // 直接删除的比例
        @Value("${optimize.rewriter.funcall.remove.rate:0.2}")
        protected Double funCallRemove4rate;

        // 需要压缩的阈值，10K
        @Value("${optimize.rewriter.funcall.largeData:10240}")
        protected Integer funCallLargeData;

        // 最近N轮不检查复杂度
        @Value("${optimize.rewriter.funcall.safePoint:3}")
        protected Integer funCallSafePoint;

        // 触发压缩的比例（当前Funcall总量与模型允许Funcall的比例上限）
        @Value("${optimize.rewriter.funcall.oversize:0.3}")
        protected Double funCall4oversize;

        // @see buildSummaryConfig
        @Value("${optimize.rewriter.histories.oversize:0.3}")
        protected Double histories4oversize;

        // @see buildSummaryConfig
        @Value("${optimize.rewriter.histories.offset:300000}")
        protected Integer histories4offset;

        // 触发上下文压缩的比例
        @Value("${optimize.rewriter.histories.rate:0.005}")
        protected Double histories4rate;

        @Value("${optimize.rewriter.histories:true}")
        protected Boolean rewriterHistories;

        @Value("${optimize.rewriter.funcall:true}")
        protected Boolean rewriterFunCall;

        @Value("${optimize.rewriter.base64:true}")
        protected Boolean rewriterBase64;

        // @see buildSummaryConfig
        @Value("${optimize.rewriter.dropOnFailed:0.8}")
        protected Double dropOnFailed;

        @Bean(RequestRewriter.NAME)
        @ConditionalOnMissingBean(name = RequestRewriter.NAME)
        public RequestRewriter requestRewriter() throws Exception {
            RequestRewriter requestRewriter = new RequestRewriter();
            BeanUtils.copyProperties(this, requestRewriter);
            log.info("RequestRewriter inited");
            return requestRewriter;
        }
    }
}
