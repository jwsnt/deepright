package ai.open.right.workflow.mcp.server.cmd;

import ai.open.right.workflow.flow.config.McpExportConfig;
import ai.open.right.workflow.mcp.McpMethod;
import ai.open.right.workflow.mcp.server.McpCmdExportService;
import ai.open.right.workflow.mcp.server.McpRequest;
import lombok.extern.slf4j.Slf4j;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
public class McpCmdInitialized implements McpCmdExportService {

    public static final McpCmdResponse NOTIFIER = McpCmdResponse.builder().notifier(true).build();

    @Override
    public void export(McpExportConfig mcpExportConfig) throws Exception {

    }

    @Override
    public void cmd(McpRequest mcpRequest) throws Exception {
        mcpRequest.write(McpCmdInitialized.NOTIFIER);
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    public static class InitConfig {

        @Bean(McpMethod.KEY_NOTIFICATIONS_INITIALIZED)
        @ConditionalOnMissingBean(name = McpMethod.KEY_NOTIFICATIONS_INITIALIZED)
        public McpCmdInitialized mcpInitialized() throws Exception {
            McpCmdInitialized mcpInitialized = new McpCmdInitialized();
            if (log.isDebugEnabled()) {
                log.debug("McpCmdInitialized inited..");
            }
            return mcpInitialized;
        }
    }
}
