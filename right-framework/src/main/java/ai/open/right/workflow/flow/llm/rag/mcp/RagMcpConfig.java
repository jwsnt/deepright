package ai.open.right.workflow.flow.llm.rag.mcp;

import ai.open.right.workflow.flow.config.McpConfig;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;

@Setter
@Getter
public class RagMcpConfig extends McpConfig {

    public Boolean hasDynamic() {
        return !StringUtils.isEmpty(this.getDynamic());
    }
}
