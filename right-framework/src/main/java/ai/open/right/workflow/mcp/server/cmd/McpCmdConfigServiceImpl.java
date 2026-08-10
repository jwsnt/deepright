package ai.open.right.workflow.mcp.server.cmd;

import ai.open.right.workflow.flow.config.McpExportConfig;
import ai.open.right.workflow.mcp.server.McpCmdConfigService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.util.HashMap;
import java.util.Map;

@Setter
@Getter
@Slf4j
public class McpCmdConfigServiceImpl implements McpCmdConfigService {

    protected final Map<String, McpExportConfig> mcpExportConfigs = new HashMap<String, McpExportConfig>();

    // 发布配置
    public void export(String name, McpExportConfig mcpExportConfig) throws Exception {
        this.mcpExportConfigs.put(name, mcpExportConfig);
    }

    @Override
    public McpExportConfig fetch(String name) throws Exception {
        McpExportConfig mcpExportConfig = this.mcpExportConfigs.get(StringUtils.trim(name));
        Assert.notNull(mcpExportConfig, "Mcp prompt config can not be empty: " + name);
        return mcpExportConfig;
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        private final McpCmdConfigServiceImpl mcpCmdConfigServiceImp = new McpCmdConfigServiceImpl();

        @Bean(McpCmdConfigService.NAME_RESOURCES_TEMPLATES)
        @ConditionalOnMissingBean(name = McpCmdConfigService.NAME_RESOURCES_TEMPLATES)
        public McpCmdConfigServiceImpl mcpCmdConfigService4ResourcesTemplates() throws Exception {
            if (log.isDebugEnabled()) {
                log.debug("McpCmdConfigServiceImpl for resources templates inited");
            }
            return this.mcpCmdConfigServiceImp;
        }

        @Bean(McpCmdConfigService.NAME_RESOURCES)
        @ConditionalOnMissingBean(name = McpCmdConfigService.NAME_RESOURCES)
        public McpCmdConfigServiceImpl mcpCmdConfigService4Resources() throws Exception {
            if (log.isDebugEnabled()) {
                log.debug("McpCmdConfigServiceImpl for resources inited");
            }
            return this.mcpCmdConfigServiceImp;
        }

        @Bean(McpCmdConfigService.NAME_PROMPTS)
        @ConditionalOnMissingBean(name = McpCmdConfigService.NAME_PROMPTS)
        public McpCmdConfigServiceImpl mcpCmdConfigService4Prompt() throws Exception {
            if (log.isDebugEnabled()) {
                log.debug("McpCmdConfigServiceImpl for prompt inited");
            }
            return this.mcpCmdConfigServiceImp;
        }

        @Bean(McpCmdConfigService.NAME_TOOLS)
        @ConditionalOnMissingBean(name = McpCmdConfigService.NAME_TOOLS)
        public McpCmdConfigService mcpCmdConfigService4Tools() throws Exception {
            if (log.isDebugEnabled()) {
                log.debug("McpCmdConfigServiceImpl for tools inited");
            }
            return this.mcpCmdConfigServiceImp;
        }
    }
}
