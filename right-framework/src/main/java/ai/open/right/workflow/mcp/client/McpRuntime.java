package ai.open.right.workflow.mcp.client;

import ai.open.right.workflow.flow.WorkflowTask;
import lombok.Builder;
import lombok.Getter;
import lombok.ToString;

@Getter
@Builder
@ToString
public class McpRuntime {

    protected WorkflowTask workTask;

    protected Integer timeout;

    protected String dynamic;

    @Builder.Default
    protected String prefix = "";

    @Builder.Default
    protected String suffix = "";

    public Integer getTimeout(Integer timeout) {
        return this.timeout != null ? this.timeout : timeout;
    }
}
