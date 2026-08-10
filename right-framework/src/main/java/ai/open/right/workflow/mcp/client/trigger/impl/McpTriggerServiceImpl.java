package ai.open.right.workflow.mcp.client.trigger.impl;

import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.trigger.McpTrigger;
import ai.open.right.workflow.mcp.client.trigger.McpTriggerService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.util.Map;

@Setter
@Getter
@Slf4j
public class McpTriggerServiceImpl implements McpTriggerService {

    protected Map<String, McpTrigger> triggers;

    protected NamesService namesService;

    protected McpTrigger global;

    public void beforeToolsCall(McpDimension dimension, Map<String, Object> arguments, WorkflowTask workTask) throws Exception {
        if (this.global != null) {
            // 全局
            if (log.isDebugEnabled()) {
                log.debug("Mcp tools trigger=global");
            }
            this.global.beforeToolsCall(dimension, arguments, workTask);
        }
        if (dimension.hasTrigger()) {
            String key = dimension.getMcpConfig().getTrigger();
            McpTrigger trigger = this.triggers.get(key);
            if (log.isDebugEnabled()) {
                log.debug("Mcp tools trigger for tools call: key={},trigger={}", key, trigger);
            }
            Assert.notNull(trigger, "Mcp tools call trigger for tools call can not be empty: " + key);
            trigger.beforeToolsCall(dimension, arguments, workTask);
        }
    }

    public void beforePromptGet(McpDimension dimension, Map<String, Object> arguments, WorkflowTask workTask) throws Exception {
        if (this.global != null) {
            // 全局
            if (log.isDebugEnabled()) {
                log.debug("Mcp prompt trigger=global");
            }
            this.global.beforePromptGet(dimension, arguments, workTask);
        }
        if (dimension.hasTrigger()) {
            String key = dimension.getMcpConfig().getTrigger();
            McpTrigger trigger = this.triggers.get(key);
            if (log.isInfoEnabled()) {
                log.info("Mcp tools trigger for prompt get: key={},trigger={}", key, trigger);
            }
            Assert.notNull(trigger, "Mcp tools call trigger for prompt get can not be empty: " + key);
            trigger.beforePromptGet(dimension, arguments, workTask);
        }
    }

    public void beforeResourcesRead(McpDimension dimension, String uri, WorkflowTask workTask) throws Exception {
        if (this.global != null) {
            // 全局
            if (log.isDebugEnabled()) {
                log.debug("Mcp resource trigger=global");
            }
            this.global.beforeResourcesRead(dimension, uri, workTask);
        }
        if (dimension.hasTrigger()) {
            String key = dimension.getMcpConfig().getTrigger();
            McpTrigger trigger = this.triggers.get(key);
            if (log.isDebugEnabled()) {
                log.debug("Mcp tools trigger for resources read: key={},trigger={}", key, trigger);
            }
            Assert.notNull(trigger, "Mcp tools call trigger for resources read can not be empty: " + key);
            trigger.beforeResourcesRead(dimension, uri, workTask);
        }
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired(required = false)
        protected Map<String, McpTrigger> triggers;

        @Autowired
        protected NamesService namesService;

        @Autowired(required = false)
        @Qualifier(McpTrigger.NAME)
        // 全局Trigger
        protected McpTrigger global;

        @Bean
        @ConditionalOnMissingBean(value = McpTriggerService.class)
        public McpTriggerService mcpTriggerService() throws Exception {
            McpTriggerServiceImpl mcpTriggerService = new McpTriggerServiceImpl();
            if (!CollectionUtils.isEmpty(this.triggers)) {
                this.triggers.remove(McpTrigger.NAME);
            }
            BeanUtils.copyProperties(this, mcpTriggerService);
            log.info("McpRewriteServiceImpl inited, triggers={}, global={}", mcpTriggerService.getTriggers(), mcpTriggerService.getGlobal());
            return mcpTriggerService;
        }
    }
}
