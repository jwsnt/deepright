package ai.open.right.workflow.mcp.server.cmd;

import ai.open.right.workflow.flow.config.McpExportConfig;
import ai.open.right.workflow.mcp.McpMethod;
import ai.open.right.workflow.mcp.client.McpClient;
import ai.open.right.workflow.mcp.server.McpCmdExportService;
import ai.open.right.workflow.mcp.server.McpRequest;
import com.google.common.collect.ImmutableMap;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
@Setter
@Getter
public class McpCmdInitialize implements McpCmdExportService {

    protected McpCmdResponse init;

    protected String project;

    @PostConstruct
    public void init() {
        // 初始化协议
        this.init = McpCmdResponse.builder()
                .result(ImmutableMap.<String, Object>builder()
                        .put("protocolVersion", McpClient.VERSION)
                        .put("capabilities", ImmutableMap.<String, Object>builder()
                                .put("experimental", ImmutableMap.of())
                                .put("prompts", ImmutableMap.of("listChanged", true))
                                .put("resources", ImmutableMap.<String, Object>builder()
                                        .put("listChanged", true)
                                        .put("subscribe", false)
                                        .build())
                                .put("tools", ImmutableMap.of("listChanged", true))
                                .build())
                        .put("serverInfo", ImmutableMap.<String, Object>builder()
                                .put("name", this.project)
                                .put("version", "1.0.0")
                                .build())
                        .build())
                .build();
    }

    @Override
    public void export(McpExportConfig mcpExportConfig) throws Exception {

    }

    @Override
    public void cmd(McpRequest mcpRequest) throws Exception {
        // 初始化协议
        mcpRequest.write(this.init);
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Value("${spring.application.name:}")
        protected String project;

        @Bean(McpMethod.KEY_INITIALIZE)
        @ConditionalOnMissingBean(name = McpMethod.KEY_INITIALIZE)
        public McpCmdInitialize mcpInitialize() throws Exception {
            McpCmdInitialize mcpInitialize = new McpCmdInitialize();
            BeanUtils.copyProperties(this, mcpInitialize);
            if (log.isDebugEnabled()) {
                log.debug("McpCmdInitialize inited..");
            }
            return mcpInitialize;
        }
    }
}
