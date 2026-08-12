package ai.open.right.workflow.notify.impl;

import ai.open.right.context.RedirectContext;
import ai.open.right.context.UserContext;
import ai.open.right.listener.EventListenerService;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.WorkflowQueue;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.WorkflowWatcher;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierWriteBack;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.UUID;

@Slf4j
@Setter
@Getter
// 推送至下一层思考链（Workflow）
public class LocalhostNotifier implements Notifier {

    protected EventListenerService eventListenerService;

    protected WorkflowQueue workflowQueue;

    @Override
    public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
        if (segment.isFinished()) {
            if (this.eventListenerService != null) {
                this.eventListenerService.listen(new LocalhostEvent(segment));
            }
            this.workflowQueue.put(new WorkflowTaskImpl(segment, notifierWriteBack, mediaContext));
        }
    }

    @Override
    public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
        this.notify(segment, redirectContext, notifierWriteBack, null);
    }

    @Override
    // 特殊用途，不传递Deepness
    public void notify(Segment segment, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
        this.notify(segment, RedirectContext.EMPTY, notifierWriteBack, mediaContext);
    }

    @Override
    // 特殊用途，不传递Deepness
    public void notify(Segment segment, NotifierWriteBack notifierWriteBack) throws Exception {
        this.notify(segment, RedirectContext.EMPTY, notifierWriteBack, null);
    }

    @Slf4j
    public static class WorkflowTaskImpl implements WorkflowTask {

        protected final WorkflowWatcher watcher = WorkflowWatcher.builder().build();

        @Getter
        protected final Long created = System.currentTimeMillis();

        protected final NotifierWriteBack notifierWriteBack;

        @Getter
        @Setter
        protected List<MediaContext> mediaContext;

        @Setter
        protected List<History> histories;

        protected final Segment segment;

        @Getter
        protected final String original;

        @Getter
        protected final String previous;

        @Getter
        protected final String initial;

        @Getter
        protected String funCallTrack;

        protected Boolean chatTrack = false;

        @Getter
        @Setter
        protected String markQuery;

        @Getter
        @Setter
        protected String workflow;

        @Getter
        @Setter
        protected String takeover;

        @Getter
        @Setter
        protected String notifier;

        @Getter
        @Setter
        protected String chat;

        @Getter
        @Setter
        protected String biz;

        public WorkflowTaskImpl(Segment segment, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) {
            this(segment, notifierWriteBack);
            this.mediaContext = mediaContext;
        }

        public WorkflowTaskImpl(Segment segment, NotifierWriteBack notifierWriteBack) {
            String[] pair = SplitUtils.split(segment.getWorkflow(), segment.getBiz());
            // 指定Takeover则覆盖Notifier
            this.notifier = StringUtils.defaultIfBlank((this.notifierWriteBack = notifierWriteBack).getTakeover(), null);
            this.funCallTrack = this.notifierWriteBack.getFunCallTrack();
            // 追加历史记录，通常为客户端传递的
            this.histories = segment.getHistories();
            this.original = segment.getOriginal();
            this.previous = segment.getInitial();
            this.initial = segment.getContent();
            this.chat = segment.getChat();
            this.workflow = pair[1];
            this.biz = pair[0];
            this.segment = segment.copy();
            this.segment.init();
        }

        @Override
        public Map<String, Object> getMetadata() {
            return this.segment.getMetadata();
        }

        @Override
        public WorkflowTaskImpl incrDeepness() {
            this.segment.incrDeepness();
            return this;
        }

        @Override
        public void setDeepness(Integer deepness) {
            this.segment.setDeepness(deepness);
        }

        @Override
        public Integer getDeepness() {
            return this.segment.getDeepness();
        }

        @Override
        public List<History> getHistories() {
            // 被动创建
            this.histories = this.histories != null ? this.histories : new ArrayList<History>();
            return this.histories;
        }

        @Override
        public UserContext getUserContext() {
            return this.segment.getUserContext();
        }

        @Override
        public String getConversation() {
            return this.segment.getConversation();
        }

        @Override
        public String getDimension() {
            return StringUtils.joinWith("-", this.getBiz(), this.getChat(), this.getDevice());
        }

        @Override
        public String getUpstream() {
            return this.segment.getUpstream();
        }

        @Override
        public String getProtocol() {
            return this.segment.getProtocol();
        }

        @Override
        public Long getCreated() {
            return this.segment.getTimestamp();
        }

        @Override
        public Long getConsuming() {
            return this.watcher.getConsuming();
        }

        @Override
        public String getDevice() {
            return this.getUserContext().getDevice();
        }

        @Override
        public String getQuery() {
            return this.segment.getContent();
        }

        @Override
        public void setProviderAndToken(String provider, String token) {
            this.putMetadata(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN, token);
            this.putMetadata(ProviderRequestService.KEY_PROVIDER, provider);
        }

        @Override
        public String getTrace() {
            return this.segment.getTrace();
        }

        @Override
        public void addMediaContext(MediaContext mediaContext) {
            this.mediaContext = this.mediaContext != null ? this.mediaContext : new ArrayList<MediaContext>();
            this.mediaContext.add(mediaContext);
        }

        @Override
        public void setUserContext(UserContext userContext) {
            this.segment.setUserContext(userContext);
        }

        @Override
        public void addHistories(List<History> histories) {
            if (CollectionUtils.isEmpty(histories)) {
                return;
            }
            if (!CollectionUtils.isEmpty(this.histories)) {
                this.histories.addAll(histories);
            } else {
                this.histories = histories;
            }
        }

        @Override
        public void setProtocol(String protocol) {
            this.segment.setProtocol(protocol);
        }

        @Override
        public void setUpstream(String upstream) {
            this.segment.setUpstream(upstream);
        }

        @Override
        public void setQuery(String query) {
            this.segment.setContent(query);
        }

        @Override
        public <T> T getMetadata(String key, Class<T> clazz) throws Exception {
            Assert.notNull(clazz, "The class can not be null");
            if (!MapUtils.isEmpty(this.segment.getMetadata())) {
                Object val = this.segment.getMetadata().get(key);
                if (val != null) {
                    return clazz.isAssignableFrom(val.getClass()) ? clazz.cast(val) : JsonUtils.transfer(val, clazz);
                } else {
                    return null;
                }
            } else {
                return null;
            }
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
            if (!CollectionUtils.isEmpty(this.segment.getMetadata())) {
                this.segment.getMetadata().remove(key);
            }
        }

        @Override
        public Boolean containMetadata(String key) {
            return MapUtils.getObject(this.getMetadata(), key) != null;
        }

        @Override
        public void beginFunCallTrack(String funCallTrack) {
            this.funCallTrack = funCallTrack;
        }

        @Override
        public void beginFunCallTrack() {
            this.beginFunCallTrack(UUID.randomUUID().toString());
        }

        @Override
        public void beginChatTrack() {
            this.chatTrack = true;
        }

        @Override
        public void closeFunCallTrack() {
            this.funCallTrack = null;
        }

        @Override
        public Boolean containFunCallTrack() {
            return !StringUtils.isEmpty(this.funCallTrack);
        }

        @Override
        public Boolean containHistories() {
            return !CollectionUtils.isEmpty(this.getHistories());
        }

        @Override
        public WorkflowTaskImpl printQuery() {
            if (log.isInfoEnabled()) {
                log.info("The query={}", this.getQuery());
            }
            return this;
        }

        @Override
        public WorkflowTaskImpl emptyQuery() {
            this.segment.setContent(null);
            return this;
        }

        @Override
        public Boolean containChatTrack() {
            return this.chatTrack;
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
        public Boolean isEntry() {
            return this.segment.isEntry();
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
            this.setQuery(JsonUtils.write(object));
        }

        @Override
        public <T> T getObjectQuery(Class<T> clazz) throws Exception {
            return JsonUtils.read(this.getQuery(), clazz);
        }

        @Override
        public void resetQuery() {
            this.segment.setContent(this.markQuery);
        }

        @Override
        public void markQuery() {
            this.markQuery = this.segment.getContent();
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
    }

    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired(required = false)
        protected EventListenerService eventListenerService;

        @Autowired
        protected WorkflowQueue workflowQueue;

        @Bean(Notifier.LOCALHOST)
        @ConditionalOnMissingBean(name = Notifier.LOCALHOST)
        public LocalhostNotifier localhostNotifier() throws Exception {
            LocalhostNotifier localhostNotifier = new LocalhostNotifier();
            BeanUtils.copyProperties(this, localhostNotifier);
            log.info("LocalhostNotifier inited");
            return localhostNotifier;
        }
    }
}
