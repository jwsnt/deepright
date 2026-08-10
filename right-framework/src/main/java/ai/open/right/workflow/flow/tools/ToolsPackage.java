package ai.open.right.workflow.flow.tools;

import lombok.Builder;
import lombok.Getter;
import lombok.Setter;

@Setter
@Getter
@Builder
public class ToolsPackage {

    // Tools Body
    protected Object tools;

    // Initial Query
    protected String query;
}
