package ai.open.right.workflow.flow.track;

import ai.open.right.workflow.flow.llm.Segment;
import lombok.Getter;
import lombok.Setter;

import java.util.Map;

@Setter
@Getter
public class TrackChatBody {

    protected Map<String, Object> metadata;

    protected String conversation;

    protected String workflow;

    protected String content;

    protected Long timestamp;

    protected Integer code;

    protected String biz;

    public TrackChatBody(Segment segment) {
        this.conversation = segment.getConversation();
        this.timestamp = segment.getTimestamp();
        this.metadata = segment.getMetadata();
        this.workflow = segment.getWorkflow();
        this.content = segment.getContent();
        this.code = segment.getCode();
    }

    public TrackChatBody() {

    }

}
