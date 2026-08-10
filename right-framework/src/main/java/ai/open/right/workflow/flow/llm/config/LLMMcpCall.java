package ai.open.right.workflow.flow.llm.config;

import ai.open.right.utils.CollectionsUtils;
import ai.open.right.workflow.flow.config.McpConfig;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;

import java.util.HashMap;
import java.util.Map;

@Setter
@Getter
public class LLMMcpCall extends McpConfig {

    // Mcp静态参数
    protected Map<String, Object> arguments;

    // MCP响应重写（对应McpRewriter实现）
    protected String rewriter;

    // 当Replace=True时，开启MCP Prompt返回结果仅一条记忆且记忆Role=User则替换Query
    protected Boolean replace;

    // Query key
    protected String query;

    public LLMMcpCall merge(LLMMcpCall llmMcpCall) throws Exception {
        super.merge(llmMcpCall);
        if (llmMcpCall != null) {
            this.rewriter = StringUtils.defaultIfBlank(this.rewriter, llmMcpCall.rewriter);
            this.arguments = CollectionsUtils.merge(this.arguments, llmMcpCall.arguments);
            this.replace = this.replace != null ? this.replace : llmMcpCall.replace;
            this.query = StringUtils.defaultIfBlank(this.query, llmMcpCall.query);
        }
        return this;
    }

    public Map<String, Object> arguments(String query) {
        if (!StringUtils.isEmpty(this.query)) {
            this.arguments = this.arguments != null ? this.arguments : new HashMap<String, Object>();
            this.arguments.put(this.query, query);
            return this.arguments;
        }
        return this.arguments;
    }

    public Boolean hasRewriter() {
        return !StringUtils.isEmpty(this.rewriter);
    }

    public Boolean hasClient() {
        return !StringUtils.isEmpty(this.getClient());
    }

    public Boolean getReplace() {
        return this.replace != null ? this.replace : false;
    }
}