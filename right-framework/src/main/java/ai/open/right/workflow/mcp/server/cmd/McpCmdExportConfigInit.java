package ai.open.right.workflow.mcp.server.cmd;

import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.config.McpExportConfig;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.config.WorkflowConfigService;
import ai.open.right.workflow.mcp.config.McpConfigInit;
import ai.open.right.workflow.mcp.server.McpCmdExportService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.util.List;
import java.util.Map;

// 初始化MCP服务发布配置
@Slf4j
@Setter
@Getter
public class McpCmdExportConfigInit implements McpConfigInit {

    public static final String NAME = "McpCmdExportConfigInit";

    // MCP CMD
    protected List<McpCmdExportService> mcpCmdExportServices;

    // 获取思考链（Workflow）配置
    protected WorkflowConfigService workflowConfigService;

    @Override
    public void init(Map<String, Object> config) throws Exception {
        // 获取MCP Exports配置
        List<String> exports = List.class.cast(config.get("mcpExports"));
        if (log.isInfoEnabled()) {
            log.info("Mcp exports={}", exports);
        }
        if (!CollectionUtils.isEmpty(exports) && !CollectionUtils.isEmpty(this.mcpCmdExportServices)) {
            for (String export : exports) {
                String[] pair = SplitUtils.split(export);
                Assert.isTrue(pair.length == 2, "Mcp export(" + export + ") must conform to the format of Biz@workflow");
                // 查询WorkflowConfig
                WorkflowConfig workflowConfig = this.workflowConfigService.config(pair[0], pair[1]);
                if (workflowConfig.hasMcp() && workflowConfig.getMcpConfig().hasExport()) {
                    // 如果有配置MCP则加载
                    McpExportConfig mcpExportConfig = workflowConfig.getMcpConfig().getExportConfig();
                    // 绑定BIZ/WORKFLOW
                    mcpExportConfig.setWorkflow(pair[1]);
                    mcpExportConfig.setBiz(pair[0]);
                    for (McpCmdExportService mcpCmdExportService : this.mcpCmdExportServices) {
                        mcpCmdExportService.export(mcpExportConfig);
                    }
                    if (log.isDebugEnabled()) {
                        log.debug("Mcp export={}", export);
                    }
                }
            }
        }
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        // MCP CMD
        @Autowired
        protected List<McpCmdExportService> mcpCmdExportServices;

        @Autowired
        // 获取思考链（Workflow）配置
        protected WorkflowConfigService workflowConfigService;

        @Bean(McpCmdExportConfigInit.NAME)
        @ConditionalOnMissingBean(name = McpCmdExportConfigInit.NAME)
        public McpCmdExportConfigInit mcpCmdExportConfigInit() throws Exception {
            McpCmdExportConfigInit mcpCmdExportConfigInit = new McpCmdExportConfigInit();
            BeanUtils.copyProperties(this, mcpCmdExportConfigInit);
            if (log.isDebugEnabled()) {
                log.debug("McpCmdExportConfigInit inited");
            }
            return mcpCmdExportConfigInit;
        }
    }
}
