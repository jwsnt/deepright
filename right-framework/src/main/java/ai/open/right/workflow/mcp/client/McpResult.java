package ai.open.right.workflow.mcp.client;

import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

@Setter
@Getter
@ToString
public class McpResult<T> {

    protected String client;

    protected String name;

    protected T result;
}
