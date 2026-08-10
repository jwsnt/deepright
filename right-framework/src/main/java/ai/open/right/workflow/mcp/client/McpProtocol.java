package ai.open.right.workflow.mcp.client;

import lombok.Getter;

public enum McpProtocol {

    PROTOCOL_RESOURCES_TEMPLATES_LIST("resources/templates/list", McpProtocol.TYPE_REQUEST),

    PROTOCOL_RESOURCES_LIST("resources/list", McpProtocol.TYPE_REQUEST),

    PROTOCOL_RESOURCES_READ("resources/read", McpProtocol.TYPE_REQUEST),

    PROTOCOL_PROMPTS_LIST("prompts/list", McpProtocol.TYPE_REQUEST),

    PROTOCOL_PROMPTS_GET("prompts/get", McpProtocol.TYPE_REQUEST),

    PROTOCOL_TOOLS_LIST("tools/list", McpProtocol.TYPE_REQUEST),

    PROTOCOL_TOOLS_CALL("tools/call", McpProtocol.TYPE_REQUEST),

    PROTOCOL_INITIALIZED("notifications/initialized", McpProtocol.TYPE_NOTIFY),

    PROTOCOL_INITIALIZE("initialize", McpProtocol.TYPE_REQUEST);

    @Getter
    protected final String name;

    protected final String type;

    public static final String TYPE_REQUEST = "request";

    public static final String TYPE_NOTIFY = "notify";

    private McpProtocol(String name, String type) {
        this.name = name;
        this.type = type;
    }

    public Boolean isRequest() {
        return this.type != null && this.type.equals(McpProtocol.TYPE_REQUEST);
    }
}
