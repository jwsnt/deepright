package ai.open.right.workflow.mcp.server.cmd;

import lombok.*;

@Setter
@Getter
@Builder
@AllArgsConstructor
@NoArgsConstructor
public class McpCmdPrompt {

    public static final String ROLE_ASSISTANT = "assistant";

    public static final String ROLE_USER = "user";

    protected String content;

    @Builder.Default
    protected String role = "user";
}
