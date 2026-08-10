package ai.open.right.workflow.flow.script;

import ai.open.right.context.UserContext;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.SegmentUsage;
import ai.open.right.workflow.flow.llm.store.history.History;
import com.fasterxml.jackson.annotation.JsonIgnore;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;

import java.util.List;
import java.util.Map;
import java.util.UUID;

public class ScriptSegment implements Segment {

    public static final ScriptUsage USAGE = new ScriptUsage();

    @JsonIgnore
    protected final Segment segment;

    @Getter
    @JsonIgnore
    protected final Object data;

    @Getter
    @Setter
    @JsonIgnore
    protected History history;

    protected String id;

    public ScriptSegment(WorkflowTask workTask, ScriptResponse scriptResponse) throws Exception {
        this(Segment.failed(workTask, scriptResponse.hasData() ? JsonUtils.write(JsonUtils.write(scriptResponse.getData())) : "", workTask.getNotifier(), scriptResponse.getCode()), scriptResponse.getData());
    }

    public ScriptSegment(Segment segment, Object data) {
        this.id = UUID.randomUUID().toString();
        this.segment = segment;
        this.data = data;
    }

    @Override
    public Map<String, Object> getMetadata() {
        return this.segment.getMetadata();
    }

    @Override
    public ScriptSegment incrDeepness() {
        this.segment.incrDeepness();
        return this;
    }

    @Override
    public void setDeepness(Integer deepness) {
        this.segment.setDeepness(deepness);
    }

    @Override
    public UserContext getUserContext() {
        return this.segment.getUserContext();
    }

    @Override
    public List<History> getHistories() {
        return this.segment.getHistories();
    }

    @Override
    public String getConversation() {
        return this.segment.getConversation();
    }

    @Override
    public ScriptUsage getUsage() {
        return ScriptSegment.USAGE;
    }

    @Override
    public Integer getDeepness() {
        return this.segment.getDeepness();
    }

    @Override
    public String getDimension() {
        return StringUtils.joinWith("-", this.getBiz(), this.getChat(), this.getDevice());
    }

    @Override
    public String getOriginal() {
        return this.segment.getOriginal();
    }

    @Override
    public String getPrevious() {
        return this.segment.getPrevious();
    }

    @Override
    public String getProtocol() {
        return this.segment.getProtocol();
    }

    @Override
    public String getWorkflow() {
        return this.segment.getWorkflow();
    }

    @Override
    public String getUpstream() {
        return this.segment.getUpstream();
    }

    @Override
    public String getNotifier() {
        return this.segment.getNotifier();
    }

    @Override
    public String getContent() {
        return this.segment.getContent();
    }

    @Override
    public String getInitial() {
        return this.segment.getInitial();
    }

    @Override
    public Long getTimestamp() {
        return this.segment.getTimestamp();
    }

    @Override
    public Boolean isFinished() {
        return this.segment.isFinished();
    }

    @Override
    public Boolean getStream() {
        return this.segment.getStream();
    }

    @Override
    public Boolean getSilent() {
        return this.segment.getSilent();
    }

    @Override
    // 获取内容长度
    public Integer getStart() {
        return this.segment.getStart();
    }

    @Override
    public Integer getIndex() {
        return this.segment.getIndex();
    }

    @Override
    public Integer getCode() {
        return this.segment.getCode();
    }

    @Override
    public String getTrace() {
        return this.segment.getTrace();
    }

    @Override
    public String getDevice() {
        return this.segment.getUserContext().getDevice();
    }

    @Override
    public String getRole() {
        return this.segment.getRole();
    }

    @Override
    public String getChat() {
        return this.segment.getChat();
    }

    @Override
    public String getBiz() {
        return this.segment.getBiz();
    }

    @Override
    public String getId() {
        return this.id;
    }

    @Override
    public void putMetadata(Map<String, Object> metadata) {
        this.segment.putMetadata(metadata);
    }

    @Override
    public void setMetadata(Map<String, Object> metadata) {
        this.segment.setMetadata(metadata);
    }

    @Override
    public void setMetadata(String key, Object val) {
        this.segment.setMetadata(key, val);
    }

    @Override
    public void delMetadata() {
        this.segment.delMetadata();
    }

    @Override
    public void setUserContext(UserContext userContext) {
        this.segment.setUserContext(userContext);
    }

    @Override
    public void setUsage(SegmentUsage segmentUsage) {
        // Nothing
    }

    @Override
    public void setProtocol(String protocol) {
        this.segment.setProtocol(protocol);
    }

    @Override
    public void setWorkflow(String workflow) {
        this.segment.setWorkflow(workflow);
    }

    @Override
    public void setNotifier(String notifier) {
        this.segment.setNotifier(notifier);
    }

    @Override
    public void setUpstream(String upstream) {
        this.segment.setUpstream(upstream);
    }

    @Override
    public void setContent(String content) {
        this.segment.setContent(content);
    }

    @Override
    public void setSilent(Boolean silent) {
        this.segment.setSilent(silent);
    }

    @Override
    public void setStart(Integer start) {
        this.segment.setStart(start);
    }

    @Override
    public void setBiz(String biz) {
        this.segment.setBiz(biz);
    }

    @Override
    public void reset(Boolean finished, Integer index) {
        this.segment.reset(finished, index);
    }

    @Override
    public void init() {
        this.segment.init();
    }

    @Override
    public void mark() {
        this.segment.mark();
    }

    @Override
    public Boolean isFromFunMerge() {
        return this.segment.isFromFunMerge();
    }

    @Override
    public Boolean isFromFunCall() {
        return this.segment.isFromFunCall();
    }

    @Override
    public Boolean isEntry() {
        return this.segment.isEntry();
    }

    @Override
    public Segment copyWithWorkflow(String workflow) {
        Segment segment = this.copy();
        segment.setWorkflow(workflow);
        return segment;
    }

    @Override
    public Segment copyWithNotifier(String notifier) {
        Segment segment = this.copy();
        segment.setNotifier(notifier);
        return segment;
    }

    @Override
    public Segment copyWithStart(Integer start) {
        Segment segment = this.copy();
        segment.setStart(start);
        return segment;
    }

    @Override
    public Segment copyWithId() {
        ScriptSegment scriptSegment = new ScriptSegment(this.segment, this.data);
        scriptSegment.setWorkflow(this.getWorkflow());
        scriptSegment.id = this.getId();
        return scriptSegment;
    }

    @Override
    public Segment copy() {
        ScriptSegment scriptSegment = new ScriptSegment(this.segment, this.data);
        scriptSegment.setWorkflow(this.getWorkflow());
        return scriptSegment;
    }
}
