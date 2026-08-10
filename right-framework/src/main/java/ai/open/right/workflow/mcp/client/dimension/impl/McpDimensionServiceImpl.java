package ai.open.right.workflow.mcp.client.dimension.impl;

import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.McpConfig;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.dimension.McpDimensionService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

@Slf4j
@Getter
@Setter
public class McpDimensionServiceImpl implements McpDimensionService {

    protected NamesService namesService;

    @Override
    public McpDimension buildDimension(McpDimension dimension, WorkflowTask workTask) throws Exception {
        McpConfig mcpConfig = dimension.getMcpConfig();
        if (mcpConfig != null && !StringUtils.isEmpty(mcpConfig.getClient()) && !StringUtils.isEmpty(mcpConfig.getName())) {
            // 存在McpConfig并指定了Client/Name
            return dimension.bind(new String[]{mcpConfig.getClient(), mcpConfig.getName()});
        } else {
            // 编码前缀
            if (this.namesService.isPrefix(workTask.getWorkflow())) {
                String[] pair = this.namesService.decode(workTask.getWorkflow());
                Assert.isTrue(pair.length == 2, "Mcp name is invalid");
                return dimension.bind(pair);
            } else {
                return dimension;
            }
        }
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected NamesService namesService;

        @Bean
        @ConditionalOnMissingBean(value = McpDimensionService.class)
        public McpDimensionServiceImpl mcpDimensionService() throws Exception {
            McpDimensionServiceImpl mcpDimensionService = new McpDimensionServiceImpl();
            BeanUtils.copyProperties(this, mcpDimensionService);
            log.info("McpDimensionServiceImpl inited");
            return mcpDimensionService;
        }
    }
}
