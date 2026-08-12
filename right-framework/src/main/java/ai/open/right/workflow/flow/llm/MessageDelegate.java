package ai.open.right.workflow.flow.llm;

import ai.open.right.context.UserContext;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import lombok.Getter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.springframework.util.CollectionUtils;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.UUID;

@Getter
@Slf4j
public class MessageDelegate implements Message {

    protected final Long created = System.currentTimeMillis();

    protected final LLMQuery llmQuery;

    protected List<History> messages;

    public MessageDelegate(LLMQuery llmQuery) {
        if ((this.llmQuery = llmQuery).containHistories()) {
            this.messages = this.llmQuery.getHistories();
        }
    }

    @Override
    public List<MediaContext> getMediaContext() {
        return this.llmQuery.getMediaContext();
    }

    @Override
    public Map<String, Object> getMetadata() {
        return this.llmQuery.getMetadata();
    }

    @Override
    public MessageDelegate incrDeepness() {
        this.llmQuery.incrDeepness();
        return this;
    }

    @Override
    public void setDeepness(Integer deepness) {
        this.llmQuery.setDeepness(deepness);
    }

    @Override
    public UserContext getUserContext() {
        return this.llmQuery.getUserContext();
    }

    @Override
    public String getConversation() {
        return this.llmQuery.getConversation();
    }

    @Override
    public String getFunCallTrack() {
        return this.llmQuery.getFunCallTrack();
    }

    @Override
    public Integer getDeepness() {
        return this.llmQuery.getDeepness();
    }

    @Override
    public String getDimension() {
        return this.llmQuery.getDimension();
    }

    @Override
    public String getWorkflow() {
        return this.llmQuery.getWorkflow();
    }

    @Override
    public String getUpstream() {
        return this.llmQuery.getUpstream();
    }

    @Override
    public String getTakeover() {
        return this.llmQuery.getTakeover();
    }

    @Override
    public String getOriginal() {
        return this.llmQuery.getOriginal();
    }

    @Override
    public String getPrevious() {
        return this.llmQuery.getPrevious();
    }

    @Override
    public String getNotifier() {
        return this.llmQuery.getNotifier();
    }

    @Override
    public String getProtocol() {
        return this.llmQuery.getProtocol();
    }

    @Override
    public Long getCreated() {
        return this.llmQuery.getCreated();
    }

    @Override
    public Long getConsuming() {
        return this.llmQuery.getConsuming();
    }

    @Override
    public String getInitial() {
        return this.llmQuery.getInitial();
    }

    @Override
    public String getDevice() {
        return this.getUserContext().getDevice();
    }

    @Override
    public String getTrace() {
        return this.llmQuery.getTrace();
    }

    public String getQuery() {
        return this.llmQuery.getQuery();
    }

    @Override
    public String getChat() {
        return this.llmQuery.getChat();
    }

    @Override
    public String getBiz() {
        return this.llmQuery.getBiz();
    }

    @Override
    public void setProviderAndToken(String provider, String token) {
        this.llmQuery.setProviderAndToken(provider, token);
    }

    @Override
    public void replaceHistories(List<History> histories) {
        this.messages = histories;
    }

    @Override
    public void addHistories(List<History> histories) {
        if (CollectionUtils.isEmpty(histories)) {
            return;
        }
        if (!CollectionUtils.isEmpty(this.messages)) {
            this.messages.addAll(histories);
        } else {
            this.messages = histories;
        }
    }

    @Override
    public void setHistories(List<History> histories) {
        this.messages = histories != null ? new ArrayList<>(histories) : null;
    }

    @Override
    public void addHistory(History history) {
        if (CollectionUtils.isEmpty(this.messages)) {
            this.messages = new ArrayList<History>();
        }
        this.messages.add(history);
    }

    @Override
    public void delHistories() {
        if (this.messages != null) {
            this.messages.clear();
        }
    }

    @Override
    public List<History> getHistories() {
        // 被动创建
        this.messages = this.messages != null ? this.messages : new ArrayList<History>();
        return this.messages;
    }

    @Override
    public Boolean hasHistory() {
        return !CollectionUtils.isEmpty(this.messages);
    }

    @Override
    public void setMediaContext(List<MediaContext> mediaContext) {
        this.llmQuery.setMediaContext(mediaContext);
    }

    @Override
    public void addMediaContext(MediaContext mediaContext) {
        this.llmQuery.addMediaContext(mediaContext);
    }

    @Override
    public void setUserContext(UserContext userContext) {
        this.llmQuery.setUserContext(userContext);
    }

    @Override
    public void setWorkflow(String workflow) {
        this.llmQuery.setWorkflow(workflow);
    }

    @Override
    public void setNotifier(String notifier) {
        this.llmQuery.setNotifier(notifier);
    }

    @Override
    public void setProtocol(String protocol) {
        this.llmQuery.setProtocol(protocol);
    }

    @Override
    public void setUpstream(String upstream) {
        this.llmQuery.setUpstream(upstream);
    }

    @Override
    public void setTakeover(String takeover) {
        this.llmQuery.setTakeover(takeover);
    }

    @Override
    public void setQuery(String query) {
        this.llmQuery.setQuery(query);
    }

    @Override
    public void setChat(String chat) {
        this.llmQuery.setChat(chat);
    }

    @Override
    public void setBiz(String biz) {
        this.llmQuery.setBiz(biz);
    }

    @Override
    public void putMetadata(String key, Object val) {
        this.llmQuery.putMetadata(key, val);
    }

    @Override
    public <T> T getMetadata(String key, Class<T> clazz) throws Exception {
        return this.llmQuery.getMetadata(key, clazz);
    }

    @Override
    public <T> T delMetadata(String key, Class<T> clazz) throws Exception {
        T t = this.llmQuery.getMetadata(key, clazz);
        this.llmQuery.delMetadata(key);
        return t;
    }

    @Override
    public void delMetadata(String key) {
        this.llmQuery.delMetadata(key);
    }

    @Override
    public Boolean containMetadata(String key) {
        return MapUtils.getObject(this.getMetadata(), key) != null;
    }

    @Override
    public void beginFunCallTrack(String funCallTrack) {
        this.llmQuery.beginFunCallTrack(funCallTrack);
    }

    @Override
    public void beginFunCallTrack() {
        this.beginFunCallTrack(UUID.randomUUID().toString());
    }

    @Override
    public void beginChatTrack() {
        this.llmQuery.beginChatTrack();
    }

    @Override
    public void closeFunCallTrack() {
        this.llmQuery.closeFunCallTrack();
    }

    @Override
    public Boolean containFunCallTrack() {
        return this.llmQuery.containFunCallTrack();
    }

    @Override
    public Boolean containChatTrack() {
        return this.llmQuery.containChatTrack();
    }

    @Override
    public Boolean containHistories() {
        return !CollectionUtils.isEmpty(this.getHistories());
    }

    @Override
    public MessageDelegate printQuery() {
        if (log.isInfoEnabled()) {
            log.info("The query={}", this.getQuery());
        }
        return this;
    }

    @Override
    public MessageDelegate emptyQuery() {
        this.llmQuery.emptyQuery();
        return this;
    }

    @Override
    public Boolean isFromFunMerge() {
        return this.llmQuery.isFromFunMerge();
    }

    @Override
    public Boolean isFromFunCall() {
        return this.llmQuery.isFromFunCall();
    }

    @Override
    public Boolean isEntry() {
        return this.llmQuery.isEntry();
    }

    @Override
    public void callToLocalHost() {
        this.llmQuery.callToLocalHost();
    }

    @Override
    public void callToEndpoint() {
        this.llmQuery.callToEndpoint();
    }

    @Override
    public void writeSource(Segment segment) throws Exception {
        this.llmQuery.writeSource(segment);
    }

    @Override
    public void writeBack(Segment segment) throws Exception {
        this.llmQuery.writeBack(segment);
    }

    @Override
    public void setObjectQuery(Object object) throws Exception {
        this.llmQuery.setObjectQuery(object);
    }

    @Override
    public <T> T getObjectQuery(Class<T> clazz) throws Exception {
        return this.llmQuery.getObjectQuery(clazz);
    }

    @Override
    public void resetQuery() {
        this.llmQuery.resetQuery();
    }

    @Override
    public void markQuery() {
        this.llmQuery.markQuery();
    }

    @Override
    public void ignoreClosed() throws Exception {
        this.llmQuery.ignoreClosed();
    }

    @Override
    public void checkClosed() throws Exception {
        this.llmQuery.checkClosed();
    }

    @Override
    public Boolean isClosed() throws Exception {
        return this.llmQuery.isClosed();
    }

    @Override
    public void close() throws Exception {
        this.llmQuery.close();
    }
}
