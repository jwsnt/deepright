package ai.open.right.workflow.flow.config;

import ai.open.right.utils.CollectionsUtils;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;
import org.springframework.util.StringUtils;

@Setter
@Getter
@ToString
public class McpConfig extends AllowedConfig {

    @JsonProperty("export")
    protected McpExportConfig exportConfig;

    // MCP服务调用超时
    protected Integer timeout;

    // MCP响应监听器，McpRewrite的实现
    protected String rewriter;

    // MCP触发监听器，McpTrigger的实现
    protected String trigger;

    // MCP在Fun Call时的思考链（Workflow）配置，不配置则用默认
    protected String dynamic;

    // Prompt Get时Query在ReQuery时的额外前缀：Prefix + Query + Suffix
    protected String prefix = "";

    // Prompt Get时Query在ReQuery时的额外后缀：Prefix + Query + Suffix
    protected String suffix = "";

    // Client
    protected String client;

    // Name
    protected String name;

    public McpConfig merge(McpConfig mcpConfig) throws Exception {
        super.merge(mcpConfig);
        if (mcpConfig != null) {
            this.exportConfig = this.exportConfig != null ? this.exportConfig.merge(mcpConfig.exportConfig) : mcpConfig.exportConfig;
            this.rewriter = StringUtils.hasText(this.rewriter) ? this.rewriter : mcpConfig.rewriter;
            this.trigger = StringUtils.hasText(this.trigger) ? this.trigger : mcpConfig.trigger;
            this.dynamic = StringUtils.hasText(this.dynamic) ? this.dynamic : mcpConfig.dynamic;
            this.prefix = StringUtils.hasText(this.prefix) ? this.prefix : mcpConfig.prefix;
            this.suffix = StringUtils.hasText(this.suffix) ? this.suffix : mcpConfig.suffix;
            this.client = StringUtils.hasText(this.client) ? this.client : mcpConfig.client;
            this.whiteList = CollectionsUtils.merge(this.whiteList, mcpConfig.whiteList);
            this.blackList = CollectionsUtils.merge(this.blackList, mcpConfig.blackList);
            this.name = StringUtils.hasText(this.name) ? this.name : mcpConfig.name;
            this.timeout = this.timeout != null ? this.timeout : mcpConfig.timeout;
        }
        return this;
    }

    public Boolean hasRewriter() {
        return StringUtils.hasText(this.rewriter);
    }

    public Boolean hasTrigger() {
        return StringUtils.hasText(this.trigger);
    }

    public Boolean hasExport() {
        return this.exportConfig != null;
    }
}
