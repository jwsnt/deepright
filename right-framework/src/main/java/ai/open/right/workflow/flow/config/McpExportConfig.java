package ai.open.right.workflow.flow.config;

import ai.open.right.utils.CollectionsUtils;
import ai.open.right.utils.SplitUtils;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.RegExUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.CollectionUtils;

import java.util.List;
import java.util.Map;

@Setter
@Getter
@Slf4j
public class McpExportConfig {

    protected Map<String, Object> properties;

    protected List<String> required;

    // 支持的协议（Tools/List，Resources/List等）
    protected List<String> methods;

    protected String description;

    // MCP Resource Template配置
    protected String uriTemplate;

    // MCP Resource/Resource Template配置
    protected String mimeType;

    protected String workflow;

    // 默认Query，用于Prompt/Resources
    protected String query;

    protected String name;

    // MCP Resource配置
    protected String uri;

    protected String biz;

    public McpExportConfig merge(McpExportConfig mcpExportConfig) throws Exception {
        if (mcpExportConfig != null) {
            this.description = StringUtils.defaultIfBlank(this.description, mcpExportConfig.description);
            this.uriTemplate = StringUtils.defaultIfBlank(this.uriTemplate, mcpExportConfig.uriTemplate);
            this.properties = CollectionsUtils.merge(this.properties, mcpExportConfig.properties);
            this.mimeType = StringUtils.defaultIfBlank(this.mimeType, mcpExportConfig.mimeType);
            this.workflow = StringUtils.defaultIfBlank(this.workflow, mcpExportConfig.workflow);
            this.required = CollectionsUtils.merge(this.required, mcpExportConfig.required);
            this.methods = CollectionsUtils.merge(this.methods, mcpExportConfig.methods);
            this.query = StringUtils.defaultIfBlank(this.query, mcpExportConfig.query);
            this.name = StringUtils.defaultIfBlank(this.name, mcpExportConfig.name);
            this.uri = StringUtils.defaultIfBlank(this.uri, mcpExportConfig.uri);
            this.biz = StringUtils.defaultIfBlank(this.biz, mcpExportConfig.biz);
        }
        return this;
    }

    public Boolean inMethod(String method) {
        if (CollectionUtils.isEmpty(this.getMethods())) {
            return true;
        }
        Boolean included = this.getMethods().contains(StringUtils.lowerCase(method));
        if (log.isInfoEnabled()) {
            log.info("McpExportConfig check method={},included={}", method, included);
        }
        return included;
    }

    public Boolean hasQuery() {
        return !StringUtils.isEmpty(this.query);
    }

    public String getMimeType() {
        return this.mimeType != null ? this.mimeType : "text/plain";
    }

    // 获取标准化Template模板名称，用于内部查询
    public static String buildTemplateFormat(String uriTemplate) {
        return RegExUtils.replaceAll(uriTemplate, "/\\{.*?\\}", "") + SplitUtils.SPLIT_AT;
    }
}
