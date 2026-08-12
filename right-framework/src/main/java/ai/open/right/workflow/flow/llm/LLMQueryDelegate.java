package ai.open.right.workflow.flow.llm;

import ai.open.right.context.UserContext;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.Notifier;
import lombok.Getter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.springframework.util.CollectionUtils;

import java.util.List;
import java.util.Map;
import java.util.UUID;

@Slf4j
@Getter
public class LLMQueryDelegate implements LLMQuery {

    protected WorkflowTask workTask;

    public LLMQueryDelegate(WorkflowTask workTask, String workflow, String notifier) {
        this.workTask = workTask;
        this.workTask.setWorkflow(workflow);
        this.workTask.setNotifier(notifier);
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
    public LLMQueryDelegate incrDeepness() {
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
    public String getFunCallTrack() {
        return this.workTask.getFunCallTrack();
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
    public String getWorkflow() {
        return this.workTask.getWorkflow();
    }

    @Override
    public String getUpstream() {
        return this.workTask.getUpstream();
    }

    @Override
    public String getTakeover() {
        return this.workTask.getTakeover();
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
    public String getNotifier() {
        return this.workTask.getNotifier();
    }

    @Override
    public String getProtocol() {
        return this.workTask.getProtocol();
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
    public String getDevice() {
        return this.getUserContext().getDevice();
    }

    @Override
    public Long getCreated() {
        return this.workTask.getCreated();
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

    // Call回LocaHost
    @Override
    public void callToLocalHost() {
        this.workTask.setNotifier(Notifier.LOCALHOST);
    }

    // Call回Endpoint
    @Override
    public void callToEndpoint() {
        this.workTask.setNotifier(Notifier.ENDPOINT);
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
    public void setProtocol(String protocol) {
        this.workTask.setProtocol(protocol);
    }

    @Override
    public void setUpstream(String upstream) {
        this.workTask.setUpstream(upstream);
    }

    @Override
    public void setTakeover(String takeover) {
        this.workTask.setTakeover(takeover);
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
        T t = this.workTask.getMetadata(key, clazz);
        this.workTask.delMetadata(key);
        return t;
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
        return MapUtils.getObject(this.getMetadata(), key) != null;
    }

    @Override
    public Boolean containHistories() {
        return !CollectionUtils.isEmpty(this.getHistories());
    }

    @Override
    public LLMQueryDelegate printQuery() {
        if (log.isInfoEnabled()) {
            log.info("The query={}", this.getQuery());
        }
        return this;
    }

    @Override
    public LLMQueryDelegate emptyQuery() {
        this.workTask.emptyQuery();
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

    @Override
    public Boolean isEntry() {
        return this.workTask.isEntry();
    }

    @Override
    public void beginFunCallTrack(String funCallTrack) {
        this.workTask.beginFunCallTrack(funCallTrack);
    }

    @Override
    public void beginFunCallTrack() {
        this.beginFunCallTrack(UUID.randomUUID().toString());
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
    public Boolean containFunCallTrack() {
        return this.workTask.containFunCallTrack();
    }

    @Override
    public Boolean containChatTrack() {
        return this.workTask.containChatTrack();
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
    public void setObjectQuery(Object object) throws Exception {
        this.setQuery(JsonUtils.write(object));
    }

    @Override
    public <T> T getObjectQuery(Class<T> clazz) throws Exception {
        return JsonUtils.read(this.getQuery(), clazz);
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
}
