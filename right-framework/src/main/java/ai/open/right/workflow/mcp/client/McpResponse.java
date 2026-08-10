package ai.open.right.workflow.mcp.client;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.mcp.client.utils.McpContentUtils;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.BooleanUtils;
import org.springframework.util.CollectionUtils;

import java.util.List;
import java.util.Map;

@Setter
@Getter
@Slf4j
public class McpResponse {

    protected String jsonrpc = "2.0";

    protected Object result;

    protected String id;

    public McpResponse check(Boolean interrupt) throws Exception {
        if (this.result != null && Map.class.isAssignableFrom(this.result.getClass())) {
            Map<String, Object> content = Map.class.cast(this.result);
            List<Map<String, Object>> contents = List.class.cast(content.get("content"));
            if (BooleanUtils.isTrue(Boolean.class.cast(content.get("isError")))) {
                if (interrupt) {
                    throw new WorkflowException(CollectionUtils.isEmpty(contents) ? "" : McpContentUtils.error(contents.getFirst()), ProtocolCode.C500);
                } else {
                    if (log.isInfoEnabled()) {
                        log.info("MCP response error={}", content);
                    }
                }
            }
        }
        return this;
    }

    public McpResponse check() throws Exception {
        return this.check(false);
    }
}
