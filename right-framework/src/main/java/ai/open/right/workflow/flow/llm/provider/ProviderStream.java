package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.TakeoverException;
import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.provider.reason.ProviderReason;
import ai.open.right.workflow.flow.llm.signal.SignalExecutor;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.llm.token.TokenStatistic;
import ai.open.right.workflow.flow.media.MediaInlineService;
import ai.open.right.workflow.flow.track.TrackDimension;
import ai.open.right.workflow.flow.track.TrackFunCall;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Slf4j
@Setter
@Getter
abstract public class ProviderStream<T extends ProviderRequest> implements SignalExecutor, LLMCallback {

    // LLM安全长度由模型控制
    protected final StringBuffer contentBuffer = new StringBuffer();

    protected final ProviderStorePolicy providerStorePolicy;

    protected final TrackFunCallService trackFunCallService;

    protected final MediaInlineService mediaInlineService;

    protected final NotifierService notifierService;

    protected final TokenStatistic tokenStatistic;

    protected final ProviderReason providerReason;

    protected final SignalStream signalStream;

    protected final HistoryStore historyStore;

    protected final NamesService namesService;

    protected final ProviderRequest request;

    protected final Segment segment;

    // Fun Call Request
    protected List<ProviderFunCallRequest> providerFunRequests;

    // 存储思考
    protected StringBuffer reasoning;

    protected Integer reasonIdx = 0;

    protected Boolean notify = true;

    protected Integer offset = 0;

    protected Integer index = 0;

    protected Integer seqid = 0;

    public ProviderStream(ProviderStreamConfig<T> providerRequestConfig) throws Exception {
        this.request = providerRequestConfig.getRequest();
        Segment.SegmentConfig segmentConfig = Segment.SegmentConfig.builder()
                // 指定Chain则使用Request配置的通知方式，如果没配置则使用Localhost。没有指定Chain则使用Endpoint
                // 如果是Fun Call则使用Endpoint
                .notifier(this.request.getMessage().isFromFunCall() ? this.request.getNotifier(Notifier.ENDPOINT) : (this.request.hasChain() ? this.request.getNotifier(Notifier.LOCALHOST) : Notifier.ENDPOINT))
                .workflow(this.request.hasChain() ? this.request.getChain() : this.request.getMessage().getWorkflow())
                .stream(this.request.getStream())
                .content(this.contentBuffer)
                .build();
        this.segment = Segment.build(this.request.getMessage(), segmentConfig);
        // 初始化LLM包装响应前缀
        this.contentBuffer.append(StringUtils.defaultIfEmpty((this.request).getPrefix(), ""));
        this.trackFunCallService = providerRequestConfig.getTrackFunCallService();
        this.providerStorePolicy = providerRequestConfig.getProviderStorePolicy();
        this.mediaInlineService = providerRequestConfig.getMediaInlineService();
        this.notifierService = providerRequestConfig.getNotifierService();
        this.providerReason = providerRequestConfig.getProviderReason();
        this.tokenStatistic = providerRequestConfig.getTokenStatistic();
        this.namesService = providerRequestConfig.getNamesService();
        this.historyStore = providerRequestConfig.getHistoryStore();
        this.signalStream = providerRequestConfig.getSignalStream();
    }

    abstract protected Boolean stream(String source) throws Exception;

    abstract protected Boolean atonce(String source) throws Exception;

    @Override
    public void callback(String message) throws Exception {
        // 由上游ProviderReaderCallback处理异常
        if (this.segment.getStream()) {
            if (this.stream(message)) {
                this.afterStream();
            }
        } else {
            if (this.atonce(message)) {
                this.afterAtOnce();
            }
        }
    }

    protected void notify(int seqid, boolean finished) throws Exception {
        // first token | finished ｜ Buffer ready
        if ((seqid <= 0 && this.contentBuffer.length() > this.request.getTokenFirst()) || finished || ((this.contentBuffer.length() - this.offset) > this.request.getTokenBuffer())) {
            if (this.signalStream != null) {
                this.signalStream.signal(this, this.request.getMessage());
            }
            if (this.notify || finished) {
                this.segment.reset(finished, this.index);
                this.notifierService.notify(this.segment, this.request.getMessage(), this.request.getMessage());
                this.offset = this.contentBuffer.length();
                this.index++;
            }
        }
    }

    // 追加LLM返回的Fun Call Request
    protected void addFunRequest(ProviderFunCallRequest providerFunRequest) throws Exception {
        // 绑定目标Workflow
        this.providerFunRequests = !CollectionUtils.isEmpty(this.providerFunRequests) ? this.providerFunRequests : new ArrayList<ProviderFunCallRequest>();
        this.providerFunRequests.add(providerFunRequest);
    }

    // 子类覆盖
    protected void addReason(Map<String, Object> message, Boolean finished) throws Exception {
    }

    // 写入文本消息
    protected String addContent(String text, Boolean finished) throws Exception {
        return this.contentBuffer.append(text).toString();
    }

    // 记录用量
    protected void tokenStatistic(Map<String, Object> body) throws Exception {

    }

    // 获取当前所有Fun Call Request的Response
    protected void getFunResponse() throws Exception {
        List<SyncWorkflowTask> syncWorkflowTasks = new ArrayList<SyncWorkflowTask>();
        for (ProviderFunCallRequest providerFunRequest : this.getFunRequest()) {
            SyncConfig syncConfig = this.getFunConfig(providerFunRequest);
            syncWorkflowTasks.add(SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig));
        }
        if (log.isDebugEnabled()) {
            log.debug("The number of Fun calls={}", syncWorkflowTasks.size());
        }
        this.addContent(this.getFunResponse(syncWorkflowTasks), false);
    }

    protected SyncConfig getFunConfig(ProviderFunCallRequest providerFunRequest) throws Exception {
        return SyncConfig.builder()
                // 如果是Takeover型需要切换Notifier
                .takeover(this.request.isTakeover(providerFunRequest.getName()) ? this.request.getTakeover(providerFunRequest.getName()).getNotifier() : null).reQuery(this.getFunRequest(providerFunRequest))
                // 如果是Takeover型不添加KEY_FUN_FETCH
                .metadata(this.request.isTakeover(providerFunRequest.getName()) ? this.getFunMetadata() : this.getFunMetadata(ProviderRequestService.KEY_FUN_FETCH, providerFunRequest))
                // 对应Name编码的MCP型Fun Call或Workflow型Fun Call
                .workflow(providerFunRequest.getName())
                .timeout(this.request.getFunCallTimeout())
                // False=独立Metadata，True=Copy Metadata
                .pure(!this.request.getFunCallHeritage())
                .workTask(this.request.getMessage())
                .build();
    }

    // 获取每个Fun Call Request的Response后聚合
    protected String getFunResponse(List<SyncWorkflowTask> syncWorkflowTasks) throws Exception {
        // KEY_FUN_SELECT为Fun Call List的内置思考链（Workflow），用于选择可用的FunCall List
        if (ProviderRequestService.KEY_FUN_SELECT.equalsIgnoreCase(this.request.getMessage().getWorkflow())) {
            // 选择可用的FunCall List
            return this.getFunSelect(syncWorkflowTasks);
        } else {
            // 获取的FunCall的结果再提交后获取真实响应
            return this.getFunData(syncWorkflowTasks);
        }
    }

    // 获取的FunCall的结果再提交后获取真实响应
    protected String getFunData(List<SyncWorkflowTask> syncWorkflowTasks) throws Exception {
        // Register FunCall
        ProviderFunCallData providerFunCall = this.request.getMessage().getMetadata(ProviderRequestService.KEY_FUN_MERGE, ProviderFunCallData.class);
        providerFunCall = providerFunCall != null ? providerFunCall : new ProviderFunCallData();
        for (int index = 0; index < syncWorkflowTasks.size(); index++) {
            ProviderFunCallRequest providerFunRequest = this.providerFunRequests.get(index);
            SyncWorkflowTask providerFunResponse = syncWorkflowTasks.get(index);
            FunCallResponse response = null;
            try {
                // Fail Fast, prevent Loop
                if (log.isDebugEnabled()) {
                    log.debug("The fun call success={}", providerFunRequest.getName());
                }
                if (this.request.isTakeover(providerFunRequest.getName())) {
                    // TakeOver Fun Call必须是当前Fun Call唯一的调用，否则会出现流程混乱
                    Assert.isTrue(syncWorkflowTasks.size() == 1, "The takeOver fun call must be the exclusive invocation of the current fun call");
                    throw TakeoverException.SIGNAL;
                } else {
                    // 正常返回
                    response = FunCallResponse.builder().data(providerFunResponse.get()).type(FunCallResponse.SUCCESS).build();
                }
            } catch (TakeoverException e) {
                throw e;
            } catch (Exception e) {
                // 异常返回
                WorkflowException.dolog(e);
                response = FunCallResponse.builder().data(WorkflowException.create(e).getMessage()).type(FunCallResponse.FAILED).build();
            }
            ProviderFunCallResponse providerFunCallResponse = ProviderFunCallResponse.builder().response(this.getFunResponse(providerFunRequest, response.getData())).name(providerFunRequest.getName()).id(providerFunRequest.getId()).build();
            providerFunCall.addFunCall(providerFunRequest, providerFunCallResponse);
            // 仅存储成功
            if (FunCallResponse.SUCCESS.equals(response.getType())) {
                // 记录FunCall Track(Request/Response)
                this.trackFunCall(providerFunRequest, providerFunResponse);
                // 存储Fun Call
                this.storeFunCallData(providerFunRequest, providerFunCallResponse);
            } else if (log.isDebugEnabled()) {
                log.debug("The fun call failed={},{}", providerFunRequest.getName(), response.getData());
            }
        }
        Assert.isTrue(providerFunCall.isValid(), "Fun Call Data is invalid");
        if (log.isDebugEnabled()) {
            log.debug("ProviderFunCall data={}", providerFunCall);
        }
        // 合并本次FunCall Response再次提交
        SyncConfig syncConfig = SyncConfig.builder().metadata(this.getFunMetadata(ProviderRequestService.KEY_FUN_MERGE, providerFunCall)).workflow(this.request.getMessage().getWorkflow()).reQuery(this.request.getMessage().getQuery()).workTask(this.request.getMessage()).timeout(this.request.getFunCallTimeout()).build();
        // 需要同步堵塞
        String funCallResponse = SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get();
        if (log.isDebugEnabled()) {
            log.debug("Fun call response={}", funCallResponse);
        }
        return funCallResponse;
    }

    // 获取FunCall的Response
    protected String getFunResponse(ProviderFunCallRequest providerFunRequest, String response) throws Exception {
        return JsonUtils.write(response);
    }

    // 获取FunCall的Request，如果Fun Call的Request（由模型返回）为空则使用原始Query
    protected String getFunRequest(ProviderFunCallRequest providerFunRequest) throws Exception {
        return StringUtils.defaultIfBlank(JsonUtils.write(providerFunRequest.getArgs()), this.request.getMessage().getQuery());
    }

    // 选择可用的FunCall List
    protected String getFunSelect(List<SyncWorkflowTask> syncWorkflowTasks) throws Exception {
        StringBuffer funCallResponse = new StringBuffer();
        for (SyncWorkflowTask syncWorkflowTask : syncWorkflowTasks) {
            // Fail Fast, prevent Loop
            try {
                funCallResponse.append(syncWorkflowTask.get());
            } catch (Exception e) {
                WorkflowException.dolog(e);
            }
        }
        if (log.isDebugEnabled()) {
            log.debug("Fun call list={}", funCallResponse);
        }
        return funCallResponse.toString();
    }

    protected Map<String, Object> getFunMetadata(String key, Object val) throws Exception {
        Map<String, Object> metadata = this.getFunMetadata();
        metadata.put(key, val);
        return metadata;
    }

    protected Map<String, Object> getFunMetadata() throws Exception {
        Map<String, Object> metadata = MapUtils.isEmpty(this.request.getMetadata()) ? new HashMap<String, Object>() : new HashMap<String, Object>(this.request.getMetadata());
        metadata.remove(ProviderRequestService.KEY_FUN_MERGE);
        metadata.remove(ProviderRequestService.KEY_FUN_FETCH);
        return metadata;
    }

    // 子类过滤
    protected List<ProviderFunCallRequest> getFunRequest() throws Exception {
        return this.providerFunRequests;
    }

    // 记录FunCall Track(Request/Response)
    protected void trackFunCall(ProviderFunCallRequest request, SyncWorkflowTask response) throws Exception {
        if (response.containFunCallTrack()) {
            try {
                TrackFunCall trackFunCall = new TrackFunCall();
                // Dimension + Track ID
                trackFunCall.setTrackDimension(new TrackDimension(response.getWorkTask(), response.getFunCallTrack()));
                trackFunCall.setResponse(response.get());
                trackFunCall.setRequest(request);
                Assert.notNull(this.trackFunCallService, "The track funcall can not be empty, please config `track.enable`");
                this.trackFunCallService.store(trackFunCall);
            } catch (Exception e) {
                WorkflowException.dolog(e);
            }
        }
    }

    protected void notifySegment() throws Exception {
        if (this.signalStream != null) {
            // 通知信号量
            this.signalStream.signal(this, this.request.getMessage());
        }
        // 重置Segment最终状态
        this.segment.reset(true, this.index);
        // 追加LLM包装后缀
        this.storeConversation(this.addContent(StringUtils.defaultIfEmpty(this.request.getSuffix(), ""), true));
        // 通知消息
        this.notifierService.notify(this.segment, this.request.getMessage(), this.request.getMessage());
    }

    // 处理入口
    protected void notifyProcess() throws Exception {
        try {
            // 处理Fun Call
            if (!CollectionUtils.isEmpty(this.providerFunRequests)) {
                this.getFunResponse();
            }
            // 信号量，记忆，通知
            this.notifySegment();
        } catch (TakeoverException e) {
            if (log.isDebugEnabled()) {
                log.debug("Takeover FunCall, terminate the process");
            }
        } catch (Exception e) {
            throw e;
        }
    }

    protected HistoryPair buildFunCallData(ProviderFunCallRequest request, ProviderFunCallResponse response) throws Exception {
        HistoryPair pair = new HistoryPair(this.request.getMessage(), request.getCreated());
        pair.setSource(SplitUtils.join(this.namesService.decode(request.getName())));
        pair.setAnswer(JsonUtils.write(response));
        pair.setQuery(JsonUtils.write(request));
        pair.setFunction(History.FUN_FUNCALL);
        pair.setModel(request.getModel());
        pair.setApi(request.getApi());
        return pair;
    }

    protected void storeFunCallData(ProviderFunCallRequest request, ProviderFunCallResponse response) throws Exception {
        if (ProviderStream.shouldStoreFunCallData(this.request, request, response) && (this.providerStorePolicy == null || this.providerStorePolicy.shouldStoreFunCallData(this.request, request, response))) {
            try {
                // 存储FunCall
                HistoryPair historyPairs = this.buildFunCallData(request, response);
                Assert.notNull(this.historyStore, "The history store can not be empty, please config `history.enable`");
                this.historyStore.store(this.request.getMessage(), this.request.getRepositories(), historyPairs, this.request.getExpired(), this.request.getHistories());
            } catch (Exception e) {
                // BuildFunCallData中NamesService可能幻觉错误
                WorkflowException.dolog(e);
            }
        }
    }

    protected List<HistoryPair> buildConversationHistories(String content) throws Exception {
        List<HistoryPair> histories = List.of(this.buildConversationRequest(), this.buildConversationResponse(content));
        return this.request.getStoreCompleted() && this.request.getStoreQuery() ? histories : histories.subList(1, histories.size());
    }

    protected HistoryPair buildConversationResponse(String content) throws Exception {
        // 使用当前时间
        HistoryPair response = new HistoryPair(this.request.getMessage(), System.currentTimeMillis());
        response.setReasoning(this.shouldStoreReasoning() ? this.reasoning.toString() : null);
        response.setModel(this.request.getModel());
        response.setApi(this.request.getApi());
        response.setAnswer(content);
        return response;
    }

    protected HistoryPair buildConversationRequest() throws Exception {
        // 使用创建时间 + 1
        HistoryPair request = new HistoryPair(this.request.getMessage(), this.request.getMessage().getCreated() + 1);
        request.setQuery(this.request.getQuery4History());
        request.setModel(this.request.getModel());
        request.setApi(this.request.getApi());
        return request;
    }

    protected String buildFailedMessage(Exception e) throws Exception {
        String prefix = "The error occurred, details are as follows: ";
        if (StringUtils.startsWithIgnoreCase(e.getMessage(), prefix)) {
            // 已添加前缀
            return e.getMessage();
        }
        return new StringBuffer().append(prefix).append("`").append(e.getMessage()).append("`").append(", please refrain from retrying similar errors.").toString();
    }

    protected List<HistoryPair> updateConversation(List<HistoryPair> historyPairs) throws Exception {
        for (HistoryPair historyPair : historyPairs) {
            historyPair.setFunction(History.FUN_CHAT);
        }
        return historyPairs;
    }

    // 子类可覆盖（逻辑需要与ProviderRequestService想同）
    protected Boolean shouldStoreConversation(String content) throws Exception {
        // Query/Answer分开存储时不判断ProviderStorePolicy，必须保存
        return ProviderStream.shouldStoreConversation(this.request, this.segment, content) && (!this.request.getStoreCompleted() && (this.providerStorePolicy == null || this.providerStorePolicy.shouldStoreConversation(this.request, this.segment, content)));
    }

    // 是否需要记录思考过程
    public Boolean shouldStoreReasoning() throws Exception {
        return !StringUtils.isEmpty(this.reasoning) && this.request.getPrintReason();
    }

    protected void storeConversation(String content) throws Exception {
        if (this.shouldStoreConversation(content)) {
            // 持久化记忆
            // 判断是否追加推理，判断是否仅保存Response
            List<HistoryPair> historyPairs = this.updateConversation(this.buildConversationHistories(content));
            this.historyStore.store(this.request.getMessage(), this.request.getRepositories(), historyPairs, this.request.getExpired(), this.request.getHistories());
        }
    }

    protected void storeConversation() throws Exception {
        this.storeConversation(this.contentBuffer.toString());
    }

    @Override
    // 用于Signal回调，处理响应
    public String getAndDelContentBuffer(int s, int e) {
        String content = this.contentBuffer.substring(s, e);
        this.contentBuffer.delete(s, e);
        return content;
    }

    @Override
    // 用于Signal回调，处理响应
    public Integer indexOfContentBuffer(String str, int s) {
        return this.contentBuffer.indexOf(str, s);
    }

    @Override
    // 用于Signal回调，处理响应
    public Integer indexOfContentBuffer(String str) {
        return this.indexOfContentBuffer(str, 0);
    }

    @Override
    // 用于Signal回调，处理响应
    public void setSignalMetadata(String signal) {
        List<String> signals = List.class.cast(this.segment.getMetadata().get(SignalExecutor.SIGNAL_KEY));
        signals = signals != null ? signals : new ArrayList<String>();
        signals.add(signal);
        this.segment.setMetadata(SignalExecutor.SIGNAL_KEY, signals);
    }

    // 报文完整度检查
    protected void responseCheck() throws Exception {
        // FunCall 或 响应内容 至少一个不能为空
        if (CollectionUtils.isEmpty(this.providerFunRequests) && this.contentBuffer.isEmpty()) {
            throw new WorkflowException("The response could not be parsed because it contains no content", ProtocolCode.C914);
        }
    }

    protected void afterStream() throws Exception {
        if (this.signalStream != null) {
            // 聚合信号
            this.signalStream.finish(this.request.getMessage());
        }
        if (log.isDebugEnabled()) {
            log.debug("The provider's streaming response has been processed completely");
        }
    }

    protected void afterAtOnce() throws Exception {
        if (this.signalStream != null) {
            // 聚合信号
            this.signalStream.finish(this.request.getMessage());
        }
        if (log.isDebugEnabled()) {
            log.debug("The provider's immediate response has been processed completely");
        }
    }

    @Override
    // 用于Signal回调，处理响应
    public void setWorkflow(String workflow) {
        this.segment.setWorkflow(workflow);
    }

    @Override
    // 用于Signal回调，处理响应
    public void setNotifier(String notifier) {
        this.segment.setNotifier(notifier);
    }

    @Override
    // 用于Signal回调，处理响应
    public void silent(Boolean silent) {
        this.segment.setSilent(silent);
    }

    @Override
    // 用于Signal回调，处理响应
    public void notify(Boolean notify) {
        this.notify = notify;
    }

    public static Boolean shouldStoreFunCallData(ProviderRequest request, ProviderFunCallRequest funCallRequest, ProviderFunCallResponse funCalResponse) throws Exception {
        return request.getContainHistories() && request.getStoreFunCall();
    }

    public static Boolean shouldStoreConversation(ProviderRequest request, Segment segment, String content) throws Exception {
        // 非记忆只读，非FunCall请求（FunCall可读不可写），非Segment静默（信号量)，Segment Finished
        // Fun Call由用户发起Query Request/模型返回FunCall Request/用户附带Fun Call Response再次Query Request...，仅保存最后一次不包含FunCall Request的响应（最终响应）
        return request.getContainHistories() && request.isWriteable() && !request.getMessage().isFromFunCall() && !segment.getSilent() && segment.isFinished();
    }

    //this.r.getContainHistories() && this.r.getStoreFunCall()

    @Builder
    @Getter
    public static class FunCallResponse {

        public static final Integer SUCCESS = 0;

        public static final Integer FAILED = 1;

        protected Integer type;

        protected String data;
    }
}