package ai.open.right.integration;

import ai.open.right.context.RedirectContext;
import ai.open.right.context.UserContext;
import ai.open.right.protocol.Protocol;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.WorkflowWatcher;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.NotifierWriteBack;
import com.fasterxml.jackson.annotation.JsonIgnore;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.UUID;

@Getter
@Setter
@Slf4j
public class RightTask implements WorkflowTask {

    private final Long created = System.currentTimeMillis();

    private final WorkflowWatcher watcher = WorkflowWatcher.builder().build();

    protected final NotifierWriteBack notifierWriteBack;

    @JsonIgnore
    protected final RightConfig rightConfig;

    protected final String query;

    public RightTask(RightConfig rightConfig, NotifierWriteBack notifierWriteBack) {
        this.notifierWriteBack = notifierWriteBack;
        this.rightConfig = rightConfig;
        this.query = this.rightConfig.getQuery();
    }

    public RightTask init() {
        // 填充UserContext默认属性
        this.rightConfig.setUserContext(UserContext.setDefault(this.getUserContext()));
        // 指定Conversation、Chat、Protocol的默认值、
        if (StringUtils.isEmpty(this.getConversation())) {
            this.rightConfig.setConversation(String.valueOf(this.getCreated()));
        }
        if (StringUtils.isEmpty(this.getProtocol())) {
            this.rightConfig.setProtocol(Protocol.CHAT);
        }
        if (StringUtils.isEmpty(this.getChat())) {
            this.rightConfig.setChat(String.valueOf(this.getCreated()));
        }
        return this;
    }

    @Override
    public List<MediaContext> getMediaContext() {
        return this.rightConfig.getMediaContext();
    }

    @Override
    public Map<String, Object> getMetadata() {
        return this.rightConfig.getMetadata();
    }

    @Override
    public UserContext getUserContext() {
        return this.rightConfig.getUserContext();
    }

    @Override
    public List<History> getHistories() {
        return this.rightConfig.getHistories();
    }

    @Override
    public RightTask incrDeepness() {
        this.rightConfig.incrDeepness();
        return this;
    }

    @Override
    public void setDeepness(Integer deepness) {
        this.rightConfig.setDeepness(deepness);
    }

    @Override
    public Boolean isEntry() {
        return StringUtils.isEmpty(this.getUpstream()) && !this.isFromFunCall() && (this.getDeepness() == null || RedirectContext.DEEPNESS.equals(this.getDeepness()));
    }

    @Override
    public String getConversation() {
        return this.rightConfig.getConversation();
    }

    @Override
    public String getDimension() {
        return StringUtils.joinWith("-", this.getBiz(), this.getChat(), this.getDevice());
    }

    @Override
    public Integer getDeepness() {
        return this.rightConfig.getDeepness();
    }

    @Override
    public String getUpstream() {
        return this.rightConfig.getUpstream();
    }

    @Override
    public Long getConsuming() {
        return this.watcher.getConsuming();
    }

    @Override
    public String getTakeover() {
        return this.rightConfig.getTakeover();
    }

    @Override
    public String getNotifier() {
        return this.rightConfig.getNotifier();
    }

    @Override
    public String getProtocol() {
        return this.rightConfig.getProtocol();
    }

    @Override
    public String getWorkflow() {
        return this.rightConfig.getWorkflow();
    }

    @Override
    public String getOriginal() {
        return this.query;
    }

    @Override
    public String getPrevious() {
        return this.query;
    }

    @Override
    public String getInitial() {
        return this.query;
    }

    @Override
    public String getDevice() {
        return this.getUserContext().getDevice();
    }

    @Override
    public String getQuery() {
        return this.rightConfig.getQuery();
    }

    @Override
    public String getTrace() {
        return this.rightConfig.getTrace();
    }

    @Override
    public String getChat() {
        return this.rightConfig.getChat();
    }

    @Override
    public String getBiz() {
        return this.rightConfig.getBiz();
    }

    @Override
    public void setProviderAndToken(String provider, String token) {
        this.putMetadata(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN, token);
        this.putMetadata(ProviderRequestService.KEY_PROVIDER, provider);
    }

    @Override
    public void setMediaContext(List<MediaContext> mediaContext) {
        this.rightConfig.setMediaContext(mediaContext);
    }

    @Override
    public void addMediaContext(MediaContext mediaContext) {
        List<MediaContext> mediaContexts = this.rightConfig.getMediaContext();
        mediaContexts = mediaContexts != null ? mediaContexts : new ArrayList<MediaContext>();
        mediaContexts.add(mediaContext);
        this.rightConfig.setMediaContext(mediaContexts);
    }

    @Override
    public void setUserContext(UserContext userContext) {
        this.rightConfig.setUserContext(userContext);
    }

    @Override
    public void setHistories(List<History> histories) {
        this.rightConfig.setHistories(histories);
    }

    @Override
    public void addHistories(List<History> histories) {
        this.rightConfig.setHistories(histories);
    }

    @Override
    public void setWorkflow(String workflow) {
        this.rightConfig.setWorkflow(workflow);
    }

    @Override
    public void setNotifier(String notifier) {
        this.rightConfig.setNotifier(notifier);
    }

    @Override
    public void setProtocol(String protocol) {
        this.rightConfig.setProtocol(protocol);
    }

    @Override
    public void setUpstream(String upstream) {
        this.rightConfig.setUpstream(upstream);
    }

    @Override
    public void setTakeover(String takeover) {
        this.rightConfig.setTakeover(takeover);
    }

    public void setTrace(String trace) {
        this.rightConfig.setTrace(trace);
    }

    @Override
    public void setQuery(String query) {
        this.rightConfig.setQuery(query);
    }

    @Override
    public void setChat(String chat) {
        this.rightConfig.setChat(chat);
    }

    @Override
    public void setBiz(String biz) {
        this.rightConfig.setBiz(biz);
    }

    @Override
    public <T> T getMetadata(String key, Class<T> clazz) throws Exception {
        return !CollectionUtils.isEmpty(this.rightConfig.getMetadata()) ? clazz.cast(this.rightConfig.getMetadata().get(key)) : null;
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
        if (!CollectionUtils.isEmpty(this.rightConfig.getMetadata())) {
            this.rightConfig.getMetadata().remove(key);
        }
    }

    @Override
    public Boolean containMetadata(String key) {
        return MapUtils.getObject(this.getMetadata(), key) != null;
    }

    @Override
    public void beginFunCallTrack(String funCallTrack) {
        this.rightConfig.setFunCallTrack(funCallTrack);
    }

    @Override
    public void beginFunCallTrack() {
        // 默认FunCall的ID为UUID
        this.beginFunCallTrack(UUID.randomUUID().toString());
    }

    @Override
    public void beginChatTrack() {
        this.rightConfig.setChatTrack(true);
    }

    @Override
    public void closeFunCallTrack() {
        this.rightConfig.setFunCallTrack(null);
    }

    @Override
    public String getFunCallTrack() {
        return this.rightConfig.getFunCallTrack();
    }

    @Override
    public Boolean containFunCallTrack() {
        return !StringUtils.isEmpty(this.rightConfig.getFunCallTrack());
    }

    @Override
    public Boolean containHistories() {
        return !CollectionUtils.isEmpty(this.getHistories());
    }

    @Override
    public RightTask printQuery() {
        if (log.isInfoEnabled()) {
            log.info("The query={}", this.getQuery());
        }
        return this;
    }

    @Override
    public Boolean containChatTrack() {
        return this.rightConfig.getChatTrack();
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
    public void writeSource(Segment segment) throws Exception {
        this.notifierWriteBack.writeSource(segment);
    }

    @Override
    public void writeBack(Segment segment) throws Exception {
        this.notifierWriteBack.writeBack(segment);
    }

    @Override
    public void setObjectQuery(Object object) throws Exception {
        this.rightConfig.setObjectQuery(object);
    }

    @Override
    public <T> T getObjectQuery(Class<T> clazz) throws Exception {
        return this.rightConfig.getObjectQuery(clazz);
    }

    @Override
    public void resetQuery() {
        this.rightConfig.setQuery(this.rightConfig.getMarkQuery());
    }

    @Override
    public void markQuery() {
        this.rightConfig.setMarkQuery(this.rightConfig.getQuery());
    }

    @Override
    public void ignoreClosed() throws Exception {
        this.notifierWriteBack.ignoreClosed();
    }

    @Override
    public void checkClosed() throws Exception {
        this.notifierWriteBack.checkClosed();
    }

    @Override
    public Boolean isClosed() throws Exception {
        return this.notifierWriteBack.isClosed();
    }

    @Override
    public void close() throws Exception {
        this.notifierWriteBack.close();
    }

    public static class RightTaskChecker {
        public static void check(RightTask rightTask) {
            Assert.hasText(rightTask.getConversation(), "Conversation can not be empty");
            Assert.notNull(rightTask.getUserContext(), "User Context can not be empty");
            Assert.notNull(rightTask.getCreated(), "Timestamp can not be empty");
            Assert.notNull(rightTask.getProtocol(), "Protocol can not be empty");
            Assert.hasText(rightTask.getWorkflow(), "Workflow can not be empty");
            Assert.hasText(rightTask.getTrace(), "Trace can not be empty");
            Assert.hasText(rightTask.getChat(), "Chat can not be empty");
            Assert.hasText(rightTask.getBiz(), "Biz can not be empty");
            UserContext.UserContextChecker.check(rightTask.getUserContext());
        }
    }
}