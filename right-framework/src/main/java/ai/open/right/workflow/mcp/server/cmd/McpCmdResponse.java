package ai.open.right.workflow.mcp.server.cmd;

import ai.open.right.workflow.mcp.server.McpResponse;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;

import java.util.Map;

@Setter
@Getter
@Builder
public class McpCmdResponse implements McpResponse {

    protected Map<String, Object> result;

    @Builder.Default
    protected Boolean notifier = false;

    @Builder.Default
    protected Boolean wrap = true;
}
