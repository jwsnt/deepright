package ai.open.right.workflow.flow;

import ai.open.right.context.UserContext;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;

import java.util.List;
import java.util.Map;

@Getter
@Setter
@Slf4j
public class WorkflowTaskWrap implements WorkflowTask {

    protected final WorkflowTask workTask;

    protected final Boolean closeable;

    protected final Long created;

    public WorkflowTaskWrap(WorkflowTask workTask, Boolean closeable, Long created) {
        this.closeable = closeable;
        this.workTask = workTask;
        this.created = created;
    }

    public WorkflowTaskWrap(WorkflowTask workTask, Boolean closeable) {
        this(workTask, closeable, workTask.getCreated());
    }

    public WorkflowTaskWrap(WorkflowTask workTask) {
        this(workTask, true);
    }

    @Override
    public List<MediaContext> getMediaContext() {
        return this.workTask.getMediaContext();
    }

    @Override
    public Map<String, Object> getMetadata() {
        return this.workTask.getMetadata();
    }

    @Override
    public WorkflowTaskWrap incrDeepness() {
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
    public List<History> getHistories() {
        return this.workTask.getHistories();
    }

    @Override
    public String getConversation() {
        return this.workTask.getConversation();
    }

    @Override
    public String getNotifier() {
        return this.workTask.getNotifier();
    }

    @Override
    public String getProtocol() {
        return this.workTask.getProtocol();
    }

    @Override
    public String getWorkflow() {
        return this.workTask.getWorkflow();
    }

    @Override
    public String getUpstream() {
        return this.workTask.getUpstream();
    }

    @Override
    public Long getCreated() {
        return this.created;
    }

    @Override
    public Long getConsuming() {
        return this.workTask.getConsuming();
    }

    @Override
    public String getTrace() {
        return this.workTask.getTrace();
    }

    @Override
    public String getQuery() {
        return this.workTask.getQuery();
    }

    @Override
    public String getChat() {
        return this.workTask.getChat();
    }

    @Override
    public String getBiz() {
        return this.workTask.getBiz();
    }

    @Override
    public void setProviderAndToken(String provider, String token) {
        this.workTask.setProviderAndToken(provider, token);
    }

    @Override
    public void setMediaContext(List<MediaContext> mediaContext) {
        this.workTask.setMediaContext(mediaContext);
    }

    @Override
    public void addMediaContext(MediaContext mediaContext) {
        this.workTask.addMediaContext(mediaContext);
    }

    @Override
    public void setUserContext(UserContext userContext) {
        this.workTask.setUserContext(userContext);
    }

    @Override
    public void setHistories(List<History> histories) {
        this.workTask.setHistories(histories);
    }

    @Override
    public void addHistories(List<History> histories) {
        this.workTask.addHistories(histories);
    }

    @Override
    public void setWorkflow(String workflow) {
        this.workTask.setWorkflow(workflow);
    }

    @Override
    public void setNotifier(String notifier) {
        this.workTask.setNotifier(notifier);
    }

    @Override
    public void setUpstream(String upstream) {
        this.workTask.setUpstream(upstream);
    }

    @Override
    public void setProtocol(String protocol) {
        this.workTask.setProtocol(protocol);
    }

    @Override
    public void setQuery(String query) {
        this.workTask.setQuery(query);
    }

    @Override
    public void setChat(String chat) {
        this.workTask.setChat(chat);
    }

    @Override
    public void setBiz(String biz) {
        this.workTask.setBiz(biz);
    }

    @Override
    public <T> T getMetadata(String key, Class<T> clazz) throws Exception {
        return this.workTask.getMetadata(key, clazz);
    }

    @Override
    public <T> T delMetadata(String key, Class<T> clazz) throws Exception {
        return this.workTask.delMetadata(key, clazz);
    }

    @Override
    public void putMetadata(String key, Object val) {
        this.workTask.putMetadata(key, val);
    }

    @Override
    public void delMetadata(String key) {
        this.workTask.delMetadata(key);
    }

    @Override
    public Boolean containMetadata(String key) {
        return this.workTask.containMetadata(key);
    }

    @Override
    public Boolean containHistories() {
        return this.workTask.containHistories();
    }

    @Override
    public WorkflowTaskWrap printQuery() {
        if (log.isInfoEnabled()) {
            log.info("The query={}", this.getQuery());
        }
        return this;
    }

    @Override
    public Boolean isFromFunMerge() {
        return this.workTask.isFromFunMerge();
    }

    @Override
    public Boolean isFromFunCall() {
        return this.workTask.isFromFunCall();
    }

    // NotifierTrack
    @Override
    public void beginFunCallTrack(String funCallTrack) {
        this.workTask.beginFunCallTrack(funCallTrack);
    }

    @Override
    public void beginFunCallTrack() {
        this.workTask.beginFunCallTrack();
    }

    @Override
    public void beginChatTrack() {
        this.workTask.beginChatTrack();
    }

    @Override
    public void closeFunCallTrack() {
        this.workTask.closeFunCallTrack();
    }

    @Override
    public String getFunCallTrack() {
        return this.workTask.getFunCallTrack();
    }

    @Override
    public Boolean containFunCallTrack() {
        return this.workTask.containFunCallTrack();
    }

    @Override
    public Boolean containChatTrack() {
        return this.workTask.containChatTrack();
    }

    @Override
    public String getTakeover() {
        return this.workTask.getTakeover();
    }

    @Override
    public void setTakeover(String takeover) {
        this.workTask.setTakeover(takeover);
    }

    @Override
    public void writeSource(Segment segment) throws Exception {
        this.workTask.writeSource(segment);
    }

    @Override
    public void writeBack(Segment segment) throws Exception {
        this.workTask.writeBack(segment);
    }

    @Override
    public void resetQuery() {
        this.workTask.resetQuery();
    }

    @Override
    public void markQuery() {
        this.workTask.markQuery();
    }

    @Override
    public void ignoreClosed() throws Exception {
        this.workTask.ignoreClosed();
    }

    @Override
    public void checkClosed() throws Exception {
        if (this.closeable) {
            this.workTask.checkClosed();
        }
    }

    @Override
    public Boolean isClosed() throws Exception {
        return this.closeable && this.workTask.isClosed();
    }

    @Override
    public void close() throws Exception {
        if (this.closeable) {
            this.workTask.close();
        }
    }

    @Override
    public String getDimension() {
        return this.workTask.getDimension();
    }

    @Override
    public Integer getDeepness() {
        return this.workTask.getDeepness();
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
    public String getInitial() {
        return this.workTask.getInitial();
    }

    @Override
    public String getDevice() {
        return this.workTask.getDevice();
    }

    @Override
    public void setObjectQuery(Object object) throws Exception {
        this.workTask.setObjectQuery(object);
    }

    @Override
    public <T> T getObjectQuery(Class<T> clazz) throws Exception {
        return this.workTask.getObjectQuery(clazz);
    }

    @Override
    public Boolean isEntry() {
        return this.workTask.isEntry();
    }
}
