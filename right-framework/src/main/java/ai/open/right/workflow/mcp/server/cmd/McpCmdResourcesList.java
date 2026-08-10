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
public class McpCmdResourcesList implements McpCmdExportService {

    protected final List<McpResourceExport> mcpCmdExports = new ArrayList<McpResourceExport>();

    protected McpCmdConfigService mcpCmdConfigService;

    @Override
    public void export(McpExportConfig mcpExportConfig) throws Exception {
        if (mcpExportConfig.inMethod(McpMethod.KEY_RESOURCES_LIST)) {
            McpResourceExport mcpResourceExport = McpResourceExport.builder()
                    .description(mcpExportConfig.getDescription())
                    .name(this.buildName(mcpExportConfig))
                    .uri(this.buildUri(mcpExportConfig))
                    .build();
            this.mcpCmdConfigService.export(mcpResourceExport.getUri(), mcpExportConfig);
            this.mcpCmdExports.add(mcpResourceExport);
        }
    }

    @Override
    public void cmd(McpRequest mcpRequest) throws Exception {
        // 获取注册的Resources List
        mcpRequest.write(McpCmdResponse.builder().result(ImmutableMap.of("resources", !CollectionUtils.isEmpty(this.mcpCmdExports) ? this.mcpCmdExports : new ArrayList<Object>())).build());
    }

    // 构建名称
    protected String buildName(McpExportConfig mcpExportConfig) throws Exception {
        return StringUtils.defaultString(mcpExportConfig.getName(), SplitUtils.join(mcpExportConfig.getWorkflow(), mcpExportConfig.getBiz()));
    }

    // 构建URI
    protected String buildUri(McpExportConfig mcpExportConfig) throws Exception {
        return StringUtils.defaultString(mcpExportConfig.getUri(), SplitUtils.join(SplitUtils.SPLIT_SLASH, mcpExportConfig.getWorkflow(), mcpExportConfig.getBiz()));
    }

    @Setter
    @Getter
    @Builder
    public static class McpResourceExport {

        protected String description;

        protected String mimeType = "text/plain";

        protected String name;

        protected String uri;
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        @Qualifier(McpCmdConfigService.NAME_RESOURCES)
        protected McpCmdConfigService mcpCmdConfigService;

        @Bean(McpMethod.KEY_RESOURCES_LIST)
        @ConditionalOnMissingBean(name = McpMethod.KEY_RESOURCES_LIST)
        public McpCmdResourcesList mcpCmdResourcesList() throws Exception {
            McpCmdResourcesList mcpCmdResourcesList = new McpCmdResourcesList();
            BeanUtils.copyProperties(this, mcpCmdResourcesList);
            if (log.isDebugEnabled()) {
                log.debug("McpCmdResourcesList inited..");
            }
            return mcpCmdResourcesList;
        }
    }
}
