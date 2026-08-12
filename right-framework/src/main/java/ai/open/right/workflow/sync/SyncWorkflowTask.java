package ai.open.right.workflow.sync;

import ai.open.right.context.UserContext;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Setter
@Getter
@Slf4j
public class SyncWorkflowTask extends SyncWriteBack implements WorkflowTask {

    protected List<MediaContext> mediaContext;

    protected Map<String, Object> metadata;

    protected final WorkflowTask workTask;

    protected List<History> histories;

    protected String markQuery;

    protected String workflow;

    protected String upstream;

    // 接管型Fun Call的Notifier
    protected String takeover;

    protected String notifier;

    protected String protocol;

    protected String query;

    protected String chat;

    protected String biz;

    public SyncWorkflowTask(WorkflowTask workTask, SyncCallable syncCallable, String takeover, String notifier, Integer interval, Integer timeout, String chat, Boolean pure) {
        // Created用于计算超时，Timestamp用于获取WorkTask起始时间
        super(workTask, syncCallable, takeover, interval, timeout, workTask.getCreated());
        this.workTask = workTask;
        this.biz = this.workTask.getBiz();
        this.query = this.workTask.getQuery();
        // Upstream等于当前Workflow
        this.upstream = this.workTask.getWorkflow();
        this.workflow = this.workTask.getWorkflow();
        this.protocol = this.workTask.getProtocol();
        this.beginFunCallTrack(workTask.getFunCallTrack());
        this.chat = StringUtils.defaultString(chat, this.workTask.getChat());
        this.takeover = StringUtils.defaultIfBlank(takeover, this.workTask.getTakeover());
        this.notifier = StringUtils.defaultIfEmpty(notifier, this.workTask.getNotifier());
        // 保留客户端的History
        this.histories = History.getReferenceHistory(workTask.getHistories(), History.REFERENCE_CLIENT);
        // Pure表示使用纯净的Metadata
        if (!pure && !CollectionUtils.isEmpty(this.workTask.getMetadata())) {
            // Metadata存在内部使用，不能用于共享，需要拷贝隔离
            // 如果需要共享使用UserContext
            this.metadata = new HashMap<String, Object>(this.workTask.getMetadata());
            this.metadata.remove(ProviderRequestService.KEY_FUN_INTERNAL);
            this.metadata.remove(ProviderRequestService.KEY_FUN_SELECT);
            this.metadata.remove(ProviderRequestService.KEY_FUN_MEDIA);
            this.metadata.remove(ProviderRequestService.KEY_FUN_MERGE);
            this.metadata.remove(ProviderRequestService.KEY_FUN_FETCH);
        }
    }


    public SyncWorkflowTask(WorkflowTask workTask, SyncCallable syncCallable, String takeover, String notifier, Integer interval, Integer timeout, Boolean pure) {
        this(workTask, syncCallable, takeover, notifier, interval, timeout, workTask.getChat(), pure);
    }

    public SyncWorkflowTask(WorkflowTask workTask, SyncCallable syncCallable, String takeover, Integer interval, Integer timeout, Boolean pure) {
        this(workTask, syncCallable, takeover, null, interval, timeout, workTask.getChat(), pure);
    }

    public SyncWorkflowTask(WorkflowTask workTask, String takeover, String notifier, Integer timeout, String chat, Boolean pure) {
        this(workTask, null, takeover, notifier, null, timeout, chat, pure);
    }

    public SyncWorkflowTask(WorkflowTask workTask, String takeover, String notifier, Integer timeout, Boolean pure) {
        this(workTask, null, takeover, notifier, null, timeout, workTask.getChat(), pure);
    }

    public SyncWorkflowTask(WorkflowTask workTask, String takeover, String notifier, Integer timeout, String chat) {
        this(workTask, null, takeover, notifier, null, timeout, chat, false);
    }

    public SyncWorkflowTask(WorkflowTask workTask, String takeover, Integer timeout, String chat, Boolean pure) {
        this(workTask, null, takeover, null, null, timeout, chat, pure);
    }

    public SyncWorkflowTask(WorkflowTask workTask, String takeover, String notifier, Integer timeout) {
        this(workTask, null, takeover, notifier, null, timeout, workTask.getChat(), false);
    }

    public SyncWorkflowTask(WorkflowTask workTask, String takeover, Integer timeout, Boolean pure) {
        this(workTask, null, takeover, null, null, timeout, workTask.getChat(), pure);
    }

    public SyncWorkflowTask(WorkflowTask workTask, String takeover, Integer timeout, String chat) {
        this(workTask, null, takeover, null, null, timeout, chat, false);
    }

    public SyncWorkflowTask(WorkflowTask workTask, String takeover, Integer timeout) {
        this(workTask, null, takeover, null, null, timeout, workTask.getChat(), false);
    }

    @Override
    public void setUserContext(UserContext userContext) {
        this.workTask.setUserContext(userContext);
    }

    @Override
    public void addHistories(List<History> histories) {
        if (CollectionUtils.isEmpty(histories)) {
            return;
        }
        if (CollectionUtils.isEmpty(this.histories)) {
            this.histories = new ArrayList<History>();
        }
        this.histories.addAll(histories);
    }

    @Override
    public Map<String, Object> getMetadata() {
        if (this.metadata == null) {
            this.metadata = new HashMap<String, Object>();
        }
        return this.metadata;
    }

    @Override
    public List<History> getHistories() {
        // 被动创建
        this.histories = this.histories != null ? this.histories : new ArrayList<History>();
        return this.histories;
    }

    @Override
    public SyncWorkflowTask incrDeepness() {
        this.workTask.incrDeepness();
        return this;
    }

    @Override
    public void setDeepness(Integer deepness) {
        this.workTask.setDeepness(deepness);
    }

    @Override
    public UserContext getUserContext() {
        return this.workTask.getUserContext();
    }

    @Override
    public String getConversation() {
        return this.workTask.getConversation();
    }

    @Override
    public Integer getDeepness() {
        return this.workTask.getDeepness();
    }

    @Override
    public String getDimension() {
        return this.workTask.getDimension();
    }

    @Override
    public String getOriginal() {
        return this.workTask.getOriginal();
    }

    @Override
    public String getPrevious() {
        return this.workTask.getPrevious();
    }

    @Override
    public Long getCreated() {
        return this.workTask.getCreated();
    }

    @Override
    public Long getConsuming() {
        return this.workTask.getConsuming();
    }

    @Override
    public String getInitial() {
        return this.workTask.getInitial();
    }

    @Override
    public Boolean isEntry() {
        return this.workTask.isEntry();
    }

    @Override
    public String getDevice() {
        return this.getUserContext().getDevice();
    }

    @Override
    public String getTrace() {
        return this.workTask.getTrace();
    }

    @Override
    public void setProviderAndToken(String provider, String token) {
        this.putMetadata(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN, token);
        this.putMetadata(ProviderRequestService.KEY_PROVIDER, provider);
    }

    @Override
    public void addMediaContext(MediaContext mediaContext) {
        this.mediaContext = this.mediaContext != null ? this.mediaContext : new ArrayList<MediaContext>();
        this.mediaContext.add(mediaContext);
    }

    @Override
    public <T> T getMetadata(String key, Class<T> clazz) throws Exception {
        return clazz.cast(this.getMetadata().get(key));
    }

    @Override
    public <T> T delMetadata(String key, Class<T> clazz) throws Exception {
        T t = this.getMetadata(key, clazz);
        this.delMetadata(key);
        return t;
    }

    @Override
    public void putMetadata(String key, Object val) {
        this.getMetadata().put(key, val);
    }

    @Override
    public void delMetadata(String key) {
        this.getMetadata().remove(key);
    }


    @Override
    public Boolean containMetadata(String key) {
        return MapUtils.getObject(this.getMetadata(), key) != null;
    }


    @Override
    public Boolean containHistories() {
        return !CollectionUtils.isEmpty(this.getHistories());
    }

    @Override
    public SyncWorkflowTask printQuery() {
        if (log.isInfoEnabled()) {
            log.info("The query={}", this.getQuery());
        }
        return this;
    }

    @Override
    public SyncWorkflowTask emptyQuery() {
        this.query = null;
        return this;
    }

    @Override
    public Boolean isFromFunMerge() {
        return ProviderRequestService.isFromFunMerge(this.getMetadata());
    }

    @Override
    public Boolean isFromFunCall() {
        return ProviderRequestService.isFromFunCall(this.getMetadata());
    }

    @Override
    public void setObjectQuery(Object object) throws Exception {
        this.setQuery(JsonUtils.write(object));
    }

    @Override
    public <T> T getObjectQuery(Class<T> clazz) throws Exception {
        return JsonUtils.read(this.getQuery(), clazz);
    }

    @Override
    public void resetQuery() {
        this.query = this.markQuery;
    }

    @Override
    public void markQuery() {
        this.markQuery = this.query;
    }

    @Override
    public void checkClosed() throws Exception {
        this.workTask.checkClosed();
    }

    @Override
    public Boolean isClosed() throws Exception {
        return this.workTask.isClosed();
    }

    @Override
    public void close() throws Exception {
        this.workTask.close();
    }

    public static SyncWorkflowTask exeWorkflow(NotifierService notifierService, SyncCallable syncCallable, WorkflowTask workTask, String biz, String workflow, Map<String, Object> metadata, List<MediaContext> mediaContext, String reQuery, String takeover, String notifier, Integer interval, Integer timeout, String chat, Boolean pure) throws Exception {
        SyncWorkflowTask syncWorkflowTask = new SyncWorkflowTask(workTask, syncCallable != null ? syncCallable.setNotifierService(notifierService).setNotifierWriteBack(workTask).setRedirectContext(workTask) : null, takeover, notifier, interval, timeout, chat, pure);
        String[] pair = SplitUtils.split(workflow, biz);
        // 重置Workflow
        syncWorkflowTask.setWorkflow(pair[1]);
        syncWorkflowTask.setBiz(pair[0]);
        Assert.hasText(syncWorkflowTask.getWorkflow(), "Workflow can not be empty");
        Assert.hasText(syncWorkflowTask.getBiz(), "Biz can not be empty");
        Segment segment = Segment.build(syncWorkflowTask, Segment.SegmentConfig.builder()
                .upstream(SplitUtils.join(workTask.getWorkflow(), workTask.getBiz()))
                .content(reQuery != null ? new StringBuffer(reQuery) : null)
                .notifier(Notifier.LOCALHOST)
                .metadata(metadata)
                .build());
        notifierService.notify(segment, syncWorkflowTask, syncWorkflowTask, mediaContext);
        return syncWorkflowTask;
    }

    public static SyncWorkflowTask exeWorkflow(NotifierService notifierService, SyncConfig syncConfig) throws Exception {
        WorkflowTask workTask = syncConfig.getWorkTask();
        Assert.notNull(workTask, "WorkTask can not be empty");
        String workflow = StringUtils.defaultString(syncConfig.getWorkflow(), workTask.getWorkflow());
        String reQuery = StringUtils.defaultString(syncConfig.getReQuery(), workTask.getQuery());
        String biz = StringUtils.defaultString(syncConfig.getBiz(), workTask.getBiz());
        List<MediaContext> mediaContext = syncConfig.getMediaContext();
        SyncCallable syncCallable = syncConfig.getSyncCallable();
        Map<String, Object> metadata = syncConfig.getMetadata();
        Integer interval = syncConfig.getInterval();
        Integer timeout = syncConfig.getTimeout();
        String takeover = syncConfig.getTakeover();
        String notifier = syncConfig.getNotifier();
        Boolean pure = syncConfig.getPure();
        String chat = syncConfig.getChat();
        Assert.hasText(workflow, "Workflow can not be empty");
        Assert.hasText(biz, "Biz can not be empty");
        return SyncWorkflowTask.exeWorkflow(notifierService, syncCallable, workTask, biz, workflow, metadata, mediaContext, reQuery, takeover, notifier, interval, timeout, chat, pure);
    }
}
