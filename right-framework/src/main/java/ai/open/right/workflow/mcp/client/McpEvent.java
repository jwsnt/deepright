package ai.open.right.workflow.mcp.client;

import ai.open.right.listener.Event;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import com.fasterxml.jackson.annotation.JsonIgnore;

import java.util.HashMap;
import java.util.Map;

public class McpEvent implements Event {

    public static final String TYPE = "mcp";

    private final Long now = System.currentTimeMillis();

    @JsonIgnore
    protected final McpDimension mcpDimension;

    @JsonIgnore
    protected final Object result;

    @JsonIgnore
    protected final String client;

    @JsonIgnore
    protected final String param;

    public McpEvent(McpDimension mcpDimension, Object result, String client, String param) {
        this.mcpDimension = mcpDimension;
        this.client = client;
        this.result = result;
        this.param = param;
    }

    public McpEvent(McpDimension mcpDimension, Object result, String client) {
        this(mcpDimension, result, client, null);
    }

    @Override
    public String getType() {
        return McpEvent.TYPE;
    }

    @Override
    public Object getData() {
        Map<String, Object> body = new HashMap<String, Object>();
        body.put("mcpDimension", this.mcpDimension);
        body.put("result", this.result);
        body.put("client", this.client);
        body.put("param", this.param);
        return body;
    }

    @Override
    public McpEvent init() {
        return this;
    }

    @Override
    public Long getNow() {
        return this.now;
    }

    @Override
    public String getBiz() {
        return this.mcpDimension.getBiz();
    }

    @Override
    public String getChat() {
        return this.mcpDimension.getChat();
    }

    @Override
    public String getDevice() {
        return this.mcpDimension.getDevice();
    }

    @Override
    public String getWorkflow() {
        return this.mcpDimension.getWorkflow();
    }

    @Override
    public String getDimension() {
        return this.mcpDimension.getDimension();
    }
}
