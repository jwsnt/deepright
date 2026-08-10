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
import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Slf4j
@Setter
@Getter
public class McpCmdToolsList implements McpCmdExportService {

    protected final List<McpToolExport> mcpCmdExports = new ArrayList<McpToolExport>();

    protected McpCmdConfigService mcpCmdConfigService;

    @Override
    public void export(McpExportConfig mcpExportConfig) throws Exception {
        if (mcpExportConfig.inMethod(McpMethod.KEY_TOOLS_LIST)) {
            Map<String, Object> inputSchema = new HashMap<String, Object>();
            inputSchema.put("type", "object");
            if (!CollectionUtils.isEmpty(mcpExportConfig.getProperties())) {
                inputSchema.put("properties", mcpExportConfig.getProperties());
            }
            if (!CollectionUtils.isEmpty(mcpExportConfig.getRequired())) {
                inputSchema.put("required", mcpExportConfig.getRequired());
            }
            McpToolExport mcpToolExport = McpToolExport.builder()
                    .description(mcpExportConfig.getDescription())
                    .name(this.buildName(mcpExportConfig))
                    .inputSchema(inputSchema)
                    .build();
            this.mcpCmdConfigService.export(mcpToolExport.getName(), mcpExportConfig);
            this.mcpCmdExports.add(mcpToolExport);
        }
    }

    @Override
    public void cmd(McpRequest mcpRequest) throws Exception {
        // 获取注册的Tools List
        mcpRequest.write(McpCmdResponse.builder().result(ImmutableMap.of("tools", !CollectionUtils.isEmpty(this.mcpCmdExports) ? this.mcpCmdExports : new ArrayList<Object>())).build());
    }

    // 构建名称
    protected String buildName(McpExportConfig mcpExportConfig) throws Exception {
        return StringUtils.defaultString(mcpExportConfig.getName(), SplitUtils.join(mcpExportConfig.getWorkflow(), mcpExportConfig.getBiz()));
    }

    @Setter
    @Getter
    @Builder
    public static class McpToolExport {

        protected Map<String, Object> inputSchema;

        protected String description;

        protected String name;
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        @Qualifier(McpCmdConfigService.NAME_TOOLS)
        protected McpCmdConfigService mcpCmdConfigService;

        @Bean(McpMethod.KEY_TOOLS_LIST)
        @ConditionalOnMissingBean(name = McpMethod.KEY_TOOLS_LIST)
        public McpCmdToolsList mcpToolsList() throws Exception {
            McpCmdToolsList mcpToolsList = new McpCmdToolsList();
            BeanUtils.copyProperties(this, mcpToolsList);
            if (log.isDebugEnabled()) {
                log.debug("McpCmdToolsList inited");
            }
            return mcpToolsList;
        }
    }
}
