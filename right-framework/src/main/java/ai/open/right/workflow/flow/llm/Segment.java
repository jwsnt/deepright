package ai.open.right.workflow.flow.llm;

import ai.open.right.context.RedirectContext;
import ai.open.right.context.UserContext;
import ai.open.right.netty.chat.NettySegment;
import ai.open.right.protocol.Protocol;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.store.Dimension;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.notify.Notifier;
import com.fasterxml.jackson.annotation.JsonIgnore;
import lombok.Builder;
import lombok.Getter;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;

import java.util.List;
import java.util.Map;

public interface Segment extends NettySegment, Dimension, RedirectContext {

    public final static String ROLE_ANSWER = "answer";

    public final static String ROLE_QUERY = "query";

    public Map<String, Object> getMetadata();

    @JsonIgnore
    public UserContext getUserContext();

    @JsonIgnore
    public List<History> getHistories();

    public String getConversation();

    public String getProtocol();

    public String getWorkflow();

    @JsonIgnore
    public String getUpstream();

    @JsonIgnore
    public String getNotifier();

    public String getContent();

    public Long getTimestamp();

    public Boolean isFinished();

    @JsonIgnore
    public Boolean getStream();

    @JsonIgnore
    public Boolean getSilent();

    @JsonIgnore
    // 获取读取起始索引
    public Integer getStart();

    public Integer getIndex();

    public Integer getCode();

    public String getTrace();

    @JsonIgnore
    public String getRole();

    public String getChat();

    public String getBiz();

    public void putMetadata(Map<String, Object> metadata);

    public void setMetadata(Map<String, Object> metadata);

    public void setMetadata(String key, Object val);

    public void delMetadata();

    public void setUserContext(UserContext userContext);

    public void setUsage(SegmentUsage segmentUsage);

    public void setProtocol(String protocol);

    public void setWorkflow(String workflow);

    public void setNotifier(String notifier);

    public void setUpstream(String upstream);

    public void setContent(String content);

    public void setSilent(Boolean silent);

    // 指定读取起始索引
    public void setStart(Integer start);

    public void setBiz(String biz);

    public void reset(Boolean finished, Integer index);

    public void init();

    public Segment copyWithWorkflow(String workflow);

    public Segment copyWithNotifier(String notifier);

    public Segment copyWithStart(Integer start);

    public Segment copyWithId();

    public Segment copy();

    public static Segment build(WorkflowTask workTask, SegmentConfig segmentConfig) {
        SegmentDelegate segment = new SegmentDelegate(workTask);
        segmentConfig.setDefault();
        if (segmentConfig.getPureMeta() != null && segmentConfig.getPureMeta()) {
            segment.delMetadata();
        }
        if (segmentConfig.getContent() != null) {
            segment.setContentBuffer(segmentConfig.getContent());
        }
        if (segmentConfig.getMetadata() != null) {
            segment.setMetadata(segmentConfig.getMetadata());
        }
        if (segmentConfig.getDeepness() != null) {
            segment.setDeepness(segmentConfig.getDeepness());
        }
        if (segmentConfig.getWorkflow() != null) {
            segment.setWorkflow(segmentConfig.getWorkflow());
        }
        if (segmentConfig.getUpstream() != null) {
            segment.setUpstream(segmentConfig.getUpstream());
        }
        if (segmentConfig.getFinished() != null) {
            segment.setFinished(segmentConfig.getFinished());
        }
        if (segmentConfig.getNotifier() != null) {
            segment.setNotifier(segmentConfig.getNotifier());
        }
        if (segmentConfig.getProtocol() != null) {
            segment.setProtocol(segmentConfig.getProtocol());
        }
        if (segmentConfig.getStream() != null) {
            segment.setStream(segmentConfig.getStream());
        }
        if (segmentConfig.getIndex() != null) {
            segment.setIndex(segmentConfig.getIndex());
        }
        if (segmentConfig.getCode() != null) {
            segment.setCode(segmentConfig.getCode());
        }
        if (segmentConfig.getBiz() != null) {
            segment.setBiz(segmentConfig.getBiz());
        }
        SegmentChecker.check(segment);
        return segment;
    }

    public static Segment failed(WorkflowTask workTask, Exception exception, String notifier, Integer code) {
        Throwable t = exception.getCause();
        while (t != null && t.getCause() != null) {
            t = t.getCause();
        }
        String content = StringUtils.defaultString(t != null ? t.getMessage() : exception.getMessage(), exception.getClass().getSimpleName());
        return Segment.failed(workTask, content, notifier, code);
    }

    public static Segment failed(WorkflowTask workTask, String message, String notifier, Integer code) {
        Segment.SegmentConfig segmentConfig = SegmentConfig.builder().content(new StringBuffer(message)).notifier(StringUtils.defaultIfEmpty(notifier, Notifier.ENDPOINT)).code(code).build();
        return Segment.build(workTask, segmentConfig);
    }

    @Getter
    @Builder
    public static class SegmentConfig {

        protected Map<String, Object> metadata;

        protected StringBuffer content;

        protected Boolean pureMeta;

        protected Boolean finished;

        protected Integer deepness;

        @Builder.Default
        protected String notifier = Notifier.LOCALHOST;

        protected String workflow;

        protected String upstream;

        protected String protocol;

        protected Boolean stream;

        protected Integer index;

        protected Integer code;

        protected String biz;

        public void setDefault() {
            if (this.pureMeta == null) {
                this.pureMeta = false;
            }
            if (this.finished == null) {
                this.finished = true;
            }
            if (this.protocol == null) {
                this.protocol = Protocol.CHAT;
            }
            if (this.stream == null) {
                this.stream = false;
            }
            if (this.index == null) {
                this.index = 0;
            }
            if (this.code == null) {
                this.code = ProtocolCode.C200;
            }
        }
    }

    public static class SegmentChecker {
        public static void check(Segment segment) {
            Assert.hasText(segment.getConversation(), "Conversation can not be empty");
            Assert.notNull(segment.getUserContext(), "UserContext can not be empty");
            Assert.notNull(segment.getContent(), "Segment content can not be empty");
            Assert.notNull(segment.getWorkflow(), "Workflow can not be empty");
            Assert.notNull(segment.getDeepness(), "Deepness can not be empty");
            Assert.hasText(segment.getNotifier(), "Notifier can not be empty");
            Assert.notNull(segment.getChat(), "Chat can not be empty");
            Assert.hasText(segment.getBiz(), "Biz can not be empty");
            UserContext.UserContextChecker.check(segment.getUserContext());
        }
    }
}
