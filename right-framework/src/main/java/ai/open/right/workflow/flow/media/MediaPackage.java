package ai.open.right.workflow.flow.media;

import lombok.Builder;
import lombok.Getter;

@Getter
@Builder
public class MediaPackage {

    protected String content;

    protected String source;
}
