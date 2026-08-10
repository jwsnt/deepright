package ai.open.right.workflow.mcp.server.impl;

import ai.open.right.WorkflowException;
import ai.open.right.workflow.mcp.server.McpCmdExportService;
import ai.open.right.workflow.mcp.server.McpDistributor;
import ai.open.right.workflow.mcp.server.McpRequest;
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

import java.util.Map;

@Slf4j
@Setter
@Getter
public class McpDistributorImpl implements McpDistributor {

    protected Map<String, McpCmdExportService> cmdServices;

    @Override
    public void distribute(McpRequest mcpRequest) throws Exception {
        try {
            McpCmdExportService cmdService = this.cmdServices.get(mcpRequest.getMethod());
            Assert.notNull(cmdService, "Can not support method: " + mcpRequest.getMethod() + "/" + mcpRequest.getId());
            cmdService.cmd(mcpRequest);
        } catch (Exception e) {
            WorkflowException.dolog(e);
            // 通用错误
            mcpRequest.error(e.getMessage());
        }
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected Map<String, McpCmdExportService> cmdServices;

        @Bean
        @ConditionalOnMissingBean(value = McpDistributor.class)
        public McpDistributor mcpRequestService() throws Exception {
            McpDistributorImpl mcpRequestService = new McpDistributorImpl();
            BeanUtils.copyProperties(this, mcpRequestService);
            if (log.isDebugEnabled()) {
                log.debug("McpDistributorImpl inited, cmd={}", this.cmdServices.keySet());
            }
            return mcpRequestService;
        }
    }
}
