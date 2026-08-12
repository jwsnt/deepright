package ai.open.right.netty.chat.distribute;

import ai.open.right.WorkflowException;
import ai.open.right.context.RedirectContext;
import ai.open.right.context.UserContext;
import ai.open.right.netty.NettyAlarm;
import ai.open.right.netty.NettyWriter;
import ai.open.right.protocol.Protocol;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.WorkflowWatcher;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import com.fasterxml.jackson.annotation.JsonIgnore;
import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import io.netty.channel.ChannelHandlerContext;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;

import java.util.*;

@Setter
@Getter
@Slf4j
@JsonIgnoreProperties(ignoreUnknown = true)
public class NettyRequest implements WorkflowTask {

    public static final String UNKNOWN = "UNKNOWN";

    private final WorkflowWatcher watcher = WorkflowWatcher.builder().build();

    @JsonIgnore
    private Long created = System.currentTimeMillis();

    // 多媒体内容
    @JsonProperty("media")
    protected List<MediaContext> mediaContext;

    // Netty通道
    @JsonIgnore
    protected ChannelHandlerContext channel;

    protected Map<String, Object> metadata;

    protected UserContext userContext;

    // 终端消息（历史记录）
    @JsonProperty("messages")
    protected List<History> histories;

    // 用于回写Chat Track（chatTrack）
    @JsonIgnore
    protected NettyTrack nettyTrack;

    protected Boolean ignoreClosed = false;

    protected String conversation;

    protected String funCallTrack;

    // 当True时回调NettyTrack
    protected Boolean chatTrack = false;

    protected String markQuery;

    // 思考链（Workflow）深度
    protected Integer deepness;

    // 通知方式（Endpoint/Localhost/Source）
    protected String notifier;

    // 上个思考链（Workflow），首次等于Workflow
    protected String upstream;

    // 接管型FunCall Notifier
    protected String takeover;

    protected String workflow;

    // Chat/Tools
    protected String protocol;

    // 思考链（Workflow）起始的Query
    protected String original;

    // 思考链（Workflow）上一次Query
    protected String previous;

    // 思考链（Workflow）当前初始Query
    protected String initial;

    protected String trace;

    // 思考链（Workflow）当前Query（可被更改）
    protected String query;

    protected String chat;

    protected String biz;

    public NettyRequest init() {
        // 标记外部历史记录
        if (!CollectionUtils.isEmpty(this.histories)) {
            for (History history : this.histories) {
                history.setReference(History.REFERENCE_CLIENT);
            }
        }
        // 初始化UserContext默认属性
        this.setUserContext(UserContext.setDefault(this.getUserContext()));
        // 为Conversation/Chat/Protocol
        if (StringUtils.isEmpty(this.getConversation())) {
            this.setConversation(String.valueOf(this.getCreated()));
        }
        if (StringUtils.isEmpty(this.getProtocol())) {
            this.setProtocol(Protocol.CHAT);
        }
        if (StringUtils.isEmpty(this.getChat())) {
            this.setChat(String.valueOf(this.getCreated()));
        }
        // 初始化时（网络到达）时original/previous/initial相等query;
        this.original = this.query;
        this.previous = this.query;
        this.initial = this.query;
        return this;
    }

    @Override
    public NettyRequest incrDeepness() {
        if (this.deepness != null) {
            this.deepness = this.deepness + RedirectContext.DEEPNESS;
        } else {
            this.deepness = RedirectContext.DEEPNESS;
        }
        return this;
    }

    @Override
    public List<History> getHistories() {
        // 被动创建
        this.histories = this.histories != null ? this.histories : new ArrayList<History>();
        return this.histories;
    }

    @Override
    public String getDimension() {
        return StringUtils.joinWith("-", this.getBiz(), this.getChat(), this.getDevice());
    }

    @Override
    public Long getConsuming() {
        return this.watcher.getConsuming();
    }

    @Override
    public String getDevice() {
        return this.userContext.getDevice();
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
        this.metadata = this.metadata != null ? this.metadata : new HashMap<String, Object>();
        return this.metadata;
    }

    @Override
    public <T> T getMetadata(String key, Class<T> clazz) throws Exception {
        Assert.notNull(clazz, "The class can not be null");
        if (!MapUtils.isEmpty(this.metadata)) {
            Object val = this.metadata.get(key);
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
        this.metadata = this.metadata != null ? this.metadata : new HashMap<String, Object>();
        this.metadata.put(key, val);
    }

    @Override
    public void delMetadata(String key) {
        if (!MapUtils.isEmpty(this.metadata)) {
            this.metadata.remove(key);
        }
    }

    public void addQuery(String query) {
        if (this.query != null) {
            this.query += System.lineSeparator() + query;
        } else {
            this.query = query;
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

    // 创建Media上下文集合
    public void initMediaContext() {
        if (!this.hasMediaContext()) {
            this.mediaContext = new ArrayList<MediaContext>();
        }
    }

    public Boolean hasMediaContext() {
        return !CollectionUtils.isEmpty(this.mediaContext);
    }

    @Override
    public Boolean containFunCallTrack() {
        return !StringUtils.isEmpty(this.funCallTrack);
    }

    @Override
    public Boolean containChatTrack() {
        return this.chatTrack;
    }

    @Override
    public Boolean containHistories() {
        return !CollectionUtils.isEmpty(this.getHistories());
    }

    @Override
    public NettyRequest printQuery() {
        if (log.isInfoEnabled()) {
            log.info("The query={}", this.getQuery());
        }
        return this;
    }

    @Override
    public NettyRequest emptyQuery() {
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
    public Boolean isEntry() {
        // 通过SegmentDelegate
        return StringUtils.isEmpty(this.getUpstream()) && !this.isFromFunCall() && (RedirectContext.DEEPNESS.equals(this.getDeepness()) || this.getDeepness() == null);
    }

    @Override
    public void writeSource(Segment segment) throws Exception {
        if (NettyWriter.isWsService(this.channel) || (segment.isFinished() && segment.getIndex() == 1)) {
            // 写入Track和通道
            this.track(segment);
            NettyWriter.write(this.channel, segment);
            return;
        }
        if (NettyWriter.isHttpService(this.channel)) {
            this.writeHttp(segment);
        }
    }

    public void writeHttp(Segment segment) throws Exception {
        // Http Stream
        if (NettyWriter.isStream(this.channel)) {
            // 写入Track和通道
            this.track(segment);
            NettyWriter.write(this.channel, segment);
            return;
        }
        // Http Once
        if (NettyWriter.isOnce(this.channel)) {
            if (segment.isFinished()) {
                // 写入Track和通道
                this.track(segment);
                NettyWriter.write(this.channel, segment);
            }
        }
    }

    @Override
    public void writeBack(Segment segment) throws Exception {
        this.writeSource(segment);
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
    public void ignoreClosed() throws Exception {
        this.ignoreClosed = true;
    }

    @Override
    public void checkClosed() throws Exception {
        if (!this.getIgnoreClosed() && this.isClosed()) {
            // 静默的异常，标记被动关闭
            throw new WorkflowException("The task (netty) was closed (" + SplitUtils.join(this.getWorkflow(), this.getBiz()) + ")", ProtocolCode.CN1).needSilent();
        }
    }

    @Override
    public Boolean isClosed() throws Exception {
        return !this.channel.channel().isActive();
    }

    @Override
    public void close() throws Exception {
        this.channel.close().addListener(NettyAlarm.INSTANCE);
    }

    // 写入Track
    protected void track(Segment segment) {
        if (this.chatTrack) {
            this.nettyTrack.track(this, segment);
        }
    }

    public static class NettyRequestChecker {

        public static void check(NettyRequest request) {
            Assert.hasText(request.getConversation(), "Conversation can not be empty, please check request body");
            Assert.notNull(request.getUserContext(), "User Context can not be empty, please check request body");
            Assert.notNull(request.getCreated(), "Timestamp can not be empty, please check request body");
            Assert.notNull(request.getProtocol(), "Protocol can not be empty, please check request body");
            Assert.hasText(request.getTrace(), "Trace can not be empty, please check request body");
            Assert.hasText(request.getChat(), "Chat can not be empty, please check request body");
            Assert.hasText(request.getBiz(), "Biz can not be empty, please check request body");
            UserContext.UserContextChecker.check(request.getUserContext());
        }
    }

}
