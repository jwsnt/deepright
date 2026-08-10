package ai.open.right.workflow.flow.media;

import lombok.Builder;
import lombok.Getter;

@Getter
@Builder
public class MediaInlineData {

    protected String mediaType;

    protected String data;
}
