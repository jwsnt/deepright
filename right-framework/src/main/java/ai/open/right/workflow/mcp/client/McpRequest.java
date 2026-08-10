package ai.open.right.workflow.mcp.client;

import com.fasterxml.jackson.annotation.JsonIgnore;
import lombok.Getter;

import java.util.Map;

@Getter
public class McpRequest {

    protected final String jsonrpc = "2.0";

    @JsonIgnore
    protected final McpProtocol protocol;

    protected final String method;

    protected final Object params;

    protected String id;

    public McpRequest(McpProtocol protocol, Map<String, Object> params) {
        this.method = (this.protocol = protocol).getName();
        if (this.protocol.isRequest()) {
            this.id = String.valueOf(System.currentTimeMillis());
        }
        this.params = params;
    }

    public McpRequest(McpProtocol protocol) {
        this(protocol, null);
    }

    public Boolean fetchResponse() {
        return this.protocol.isRequest();
    }
}
