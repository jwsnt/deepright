package ai.open.right.workflow.flow.llm;

import ai.open.right.context.RedirectContext;
import ai.open.right.context.UserContext;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.History;
import com.fasterxml.jackson.annotation.JsonIgnore;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.util.Assert;

import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;

@Setter
@Getter
@ToString
public class SegmentDelegate implements Segment {

    public final static String ROLE_ANSWER = "answer";

    public final static String ROLE_QUERY = "query";

    protected Map<String, Object> metadata;

    @JsonIgnore
    protected StringBuffer contentBuffer;

    protected SegmentUsage segmentUsage;

    protected UserContext userContext;

    @JsonIgnore
    protected WorkflowTask workTask;

    protected Boolean finished = false;

    @JsonIgnore
    protected String original;

    @JsonIgnore
    protected String previous;

    protected String protocol;

    protected String workflow;

    protected String upstream;

    protected String notifier;

    @JsonIgnore
    protected String initial;

    protected Boolean silent = false;

    protected Boolean stream = true;

    protected Integer index = 0;

    @JsonIgnore
    protected Integer start = 0;

    protected Integer code = ProtocolCode.C200;

    protected String role = Segment.ROLE_ANSWER;

    protected String biz;

    protected String id;

    public SegmentDelegate(WorkflowTask workTask) {
        this();
        this.workTask = workTask;
        this.contentBuffer = new StringBuffer(StringUtils.defaultString(workTask.getQuery(), ""));
        this.upstream = SplitUtils.join(workTask.getWorkflow(), workTask.getBiz());
        this.metadata = new HashMap<String, Object>(workTask.getMetadata());
        this.userContext = workTask.getUserContext();
        this.notifier = workTask.getNotifier();
        this.workflow = workTask.getWorkflow();
        this.original = workTask.getOriginal();
        this.previous = workTask.getPrevious();
        this.initial = workTask.getInitial();
        this.biz = workTask.getBiz();
    }

    public SegmentDelegate() {
        this.id = UUID.randomUUID().toString();
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
        return History.getReferenceHistory(this.workTask.getHistories(), History.REFERENCE_SERVER);
    }

    @Override
    public SegmentDelegate incrDeepness() {
        this.workTask.incrDeepness();
        return this;
    }

    @Override
    public void setDeepness(Integer deepness) {
        this.workTask.setDeepness(deepness);
    }

    @Override
    public Integer getDeepness() {
        return this.workTask.getDeepness();
    }

    @Override
    public String getConversation() {
        return this.workTask.getConversation();
    }

    @Override
    public SegmentUsage getUsage() {
        return this.segmentUsage;
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
    public String getContent() {
        return this.contentBuffer.substring(this.start);
    }

    @Override
    public Long getTimestamp() {
        return this.workTask.getCreated();
    }

    @Override
    // 获取内容长度
    public Integer getStart() {
        return this.start;
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
    public String getChat() {
        return this.workTask.getChat();
    }

    @Override
    public Boolean isFromFunMerge() {
        return ProviderRequestService.isFromFunMerge(this.getMetadata());
    }

    @Override
    public Boolean isFromFunCall() {
        return ProviderRequestService.isFromFunCall(this.getMetadata());
    }

    public Boolean isFinished() {
        return this.finished != null ? this.finished : false;
    }

    @Override
    public void putMetadata(Map<String, Object> metadata) {
        this.metadata = metadata;
    }

    @Override
    public void setMetadata(Map<String, Object> metadata) {
        Assert.notNull(metadata, "The metadata can not be empty");
        if (!MapUtils.isEmpty(metadata)) {
            // 非空追加
            this.getMetadata().putAll(metadata);
        } else {
            // Metadata本身会被其他Segment引用，不能清空，空覆盖（清空）
            this.metadata = metadata;
        }
    }

    @Override
    public void setMetadata(String key, Object val) {
        this.getMetadata().put(key, val);
    }

    @Override
    public void delMetadata() {
        this.metadata = null;
    }

    public void setUsage(SegmentUsage segmentUsage) {
        this.segmentUsage = segmentUsage;
    }

    @Override
    public void setContent(String content) {
        String currentContent = StringUtils.defaultString(content, "");
        if (this.contentBuffer != null) {
            this.contentBuffer.delete(0, this.contentBuffer.length());
            this.contentBuffer.append(currentContent);
        } else {
            this.contentBuffer = new StringBuffer(currentContent);
        }
        this.start = 0;
    }

    @Override
    public void setBiz(String biz) {
        this.biz = biz;
    }

    @Override
    public void reset(Boolean finished, Integer index) {
        this.finished = finished;
        this.index = index;
    }

    @Override
    public void mark() {
        this.start = this.contentBuffer.length();
    }

    public void init() {
        this.finished = null;
        this.notifier = null;
        this.stream = null;
        this.index = null;
        this.code = null;
        this.role = null;
    }

    @Override
    public Boolean isEntry() {
        // 通过NettyRequest（需要使用WorkTask属性）
        return StringUtils.isEmpty(this.workTask.getUpstream()) && !this.workTask.isFromFunCall() && (RedirectContext.DEEPNESS.equals(this.getDeepness()) || this.getDeepness() == null);
    }

    @Override
    public Segment copyWithWorkflow(String workflow) {
        Segment copy = this.copy();
        copy.setWorkflow(workflow);
        return copy;
    }

    @Override
    public Segment copyWithNotifier(String notifier) {
        Segment copy = this.copy();
        copy.setNotifier(notifier);
        return copy;
    }

    @Override
    public Segment copyWithStart(Integer start) {
        Segment segment = this.copy();
        segment.setStart(start);
        return segment;
    }

    @Override
    public Segment copyWithId() {
        SegmentDelegate segmentDelegate = new SegmentDelegate();
        BeanUtils.copyProperties(this, segmentDelegate, "deepness");
        segmentDelegate.workflow = this.getWorkflow();
        return segmentDelegate;
    }

    @Override
    public Segment copy() {
        // 同时Copy WorkTask
        SegmentDelegate segmentDelegate = new SegmentDelegate();
        // Deepness通过Workflow代理，不能Set
        // SegmentDelegate.GetMetadata会自动创建，需要覆盖
        BeanUtils.copyProperties(this, segmentDelegate, "deepness", "id");
        segmentDelegate.workflow = this.getWorkflow();
        return segmentDelegate;
    }
}
