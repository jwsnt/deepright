package ai.open.right.workflow.mcp.server.cmd;

import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.config.McpExportConfig;
import ai.open.right.workflow.mcp.McpMethod;
import ai.open.right.workflow.mcp.server.McpCmdConfigService;
import ai.open.right.workflow.mcp.server.McpCmdExportService;
import ai.open.right.workflow.mcp.server.McpRequest;
import com.google.common.collect.ImmutableMap;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.CollectionUtils;

import java.util.ArrayList;
import java.util.List;

@Slf4j
@Setter
@Getter
public class McpCmdResourcesTemplatesList implements McpCmdExportService {

    protected final List<McpResourceTemplateExport> mcpCmdExports = new ArrayList<McpResourceTemplateExport>();

    protected McpCmdConfigService mcpCmdConfigService;

    @Override
    public void export(McpExportConfig mcpExportConfig) throws Exception {
        if (mcpExportConfig.inMethod(McpMethod.KEY_RESOURCES_TEMPLATES_LIST)) {
            McpResourceTemplateExport mcpResourceTemplateExport = McpResourceTemplateExport.builder()
                    .uriTemplate(this.buildUriTemplate(mcpExportConfig))
                    .description(mcpExportConfig.getDescription())
                    .name(this.buildName(mcpExportConfig))
                    .build();
            // 使用标准化URI Template做为Key
            this.mcpCmdConfigService.export(McpExportConfig.buildTemplateFormat(mcpResourceTemplateExport.getUriTemplate()), mcpExportConfig);
            this.mcpCmdExports.add(mcpResourceTemplateExport);
        }
    }

    @Override
    public void cmd(McpRequest mcpRequest) throws Exception {
        // 获取注册的Resources Templates List
        mcpRequest.write(McpCmdResponse.builder().result(ImmutableMap.of("resourceTemplates", !CollectionUtils.isEmpty(this.mcpCmdExports) ? this.mcpCmdExports : new ArrayList<Object>())).build());
    }

    // 构建URI
    protected String buildUriTemplate(McpExportConfig mcpExportConfig) throws Exception {
        return StringUtils.defaultString(mcpExportConfig.getUriTemplate(), SplitUtils.join(SplitUtils.SPLIT_SLASH, mcpExportConfig.getWorkflow(), mcpExportConfig.getBiz()) + SplitUtils.SPLIT_SLASH + "{query}");
    }

    // 构建名称
    protected String buildName(McpExportConfig mcpExportConfig) throws Exception {
        return StringUtils.defaultString(mcpExportConfig.getName(), SplitUtils.join(mcpExportConfig.getWorkflow(), mcpExportConfig.getBiz()));
    }

    @Setter
    @Getter
    @Builder
    public static class McpResourceTemplateExport {

        protected String description;

        protected String uriTemplate;

        protected String mimeType = "text/plain";

        protected String name;
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        @Qualifier(McpCmdConfigService.NAME_RESOURCES_TEMPLATES)
        protected McpCmdConfigService mcpCmdConfigService;

        @Bean(McpMethod.KEY_RESOURCES_TEMPLATES_LIST)
        @ConditionalOnMissingBean(name = McpMethod.KEY_RESOURCES_TEMPLATES_LIST)
        public McpCmdResourcesTemplatesList mcpCmdResourcesTemplatesList() throws Exception {
            McpCmdResourcesTemplatesList mcpCmdResourcesTemplatesList = new McpCmdResourcesTemplatesList();
            BeanUtils.copyProperties(this, mcpCmdResourcesTemplatesList);
            if (log.isDebugEnabled()) {
                log.debug("McpCmdResourcesTemplatesList inited..");
            }
            return mcpCmdResourcesTemplatesList;
        }
    }
}
