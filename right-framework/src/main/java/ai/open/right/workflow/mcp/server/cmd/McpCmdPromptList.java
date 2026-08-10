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
public class McpCmdPromptList implements McpCmdExportService {

    // 发布的Prompt服务
    protected final List<McpPromptExport> mcpCmdExports = new ArrayList<McpPromptExport>();

    protected McpCmdConfigService mcpCmdConfigService;

    @Override
    public void export(McpExportConfig mcpExportConfig) throws Exception {
        if (mcpExportConfig.inMethod(McpMethod.KEY_PROMPTS_LIST)) {
            Map<String, Object> inputSchema = new HashMap<String, Object>();
            inputSchema.put("type", "object");
            if (!CollectionUtils.isEmpty(mcpExportConfig.getProperties())) {
                inputSchema.put("properties", mcpExportConfig.getProperties());
            }
            if (!CollectionUtils.isEmpty(mcpExportConfig.getRequired())) {
                inputSchema.put("required", mcpExportConfig.getRequired());
            }
            McpPromptExport mcpPromptExport = McpPromptExport.builder()
                    .description(mcpExportConfig.getDescription())
                    .name(this.buildName(mcpExportConfig))
                    .inputSchema(inputSchema)
                    .build();
            this.mcpCmdConfigService.export(mcpPromptExport.getName(), mcpExportConfig);
            this.mcpCmdExports.add(mcpPromptExport);
        }
    }

    @Override
    public void cmd(McpRequest mcpRequest) throws Exception {
        // 获取注册的Prompt List
        mcpRequest.write(McpCmdResponse.builder().result(ImmutableMap.of("prompts", !CollectionUtils.isEmpty(this.mcpCmdExports) ? this.mcpCmdExports : new ArrayList<Object>())).build());
    }

    // 构建名称
    protected String buildName(McpExportConfig mcpExportConfig) throws Exception {
        return StringUtils.defaultString(mcpExportConfig.getName(), SplitUtils.join(mcpExportConfig.getWorkflow(), mcpExportConfig.getBiz()));
    }

    @Setter
    @Getter
    @Builder
    public static class McpPromptExport {

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
        @Qualifier(McpCmdConfigService.NAME_PROMPTS)
        protected McpCmdConfigService mcpCmdConfigService;

        @Bean(McpMethod.KEY_PROMPTS_LIST)
        @ConditionalOnMissingBean(name = McpMethod.KEY_PROMPTS_LIST)
        public McpCmdPromptList mcpCmdPromptList() throws Exception {
            McpCmdPromptList mcpCmdPromptList = new McpCmdPromptList();
            BeanUtils.copyProperties(this, mcpCmdPromptList);
            if (log.isDebugEnabled()) {
                log.debug("McpCmdPromptList inited..");
            }
            return mcpCmdPromptList;
        }
    }
}
