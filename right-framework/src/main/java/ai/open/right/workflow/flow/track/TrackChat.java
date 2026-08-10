package ai.open.right.workflow.flow.track;

import ai.open.right.workflow.flow.llm.store.Dimension;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;

@Setter
@Getter
@Builder
public class TrackChat {

    protected TrackChatBody trackChatBody;

    protected Dimension dimension;
}
