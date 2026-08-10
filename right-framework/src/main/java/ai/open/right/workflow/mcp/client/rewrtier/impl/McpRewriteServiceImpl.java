package ai.open.right.workflow.mcp.client.rewrtier.impl;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.mcp.client.McpContext;
import ai.open.right.workflow.mcp.client.McpResult;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.rewrtier.McpRewriteService;
import ai.open.right.workflow.mcp.client.rewrtier.McpRewriter;
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

import java.util.List;
import java.util.Map;

@Setter
@Getter
@Slf4j
public class McpRewriteServiceImpl implements McpRewriteService {

    protected Map<String, McpRewriter> rewriter;

    protected McpRewriter global;

    public McpResult<List<Map<String, Object>>> toolsCall(McpDimension dimension, Map<String, Object> arguments, WorkflowTask workTask, McpResult<List<Map<String, Object>>> result) throws Exception {
        if (dimension.hasListener()) {
            String key = dimension.getMcpConfig().getRewriter();
            McpRewriter rewriter = this.rewriter.get(key);
            if (log.isDebugEnabled()) {
                log.debug("Mcp tools rewriter: key={},rewriter={}", key, rewriter);
            }
            Assert.notNull(rewriter, "Mcp tools call rewriter can not be empty: " + key);
            return rewriter.toolsCall(this.<List<Map<String, Object>>>buildMcpContext(dimension, arguments, workTask, result));
        }
        if (this.global != null) {
            if (log.isDebugEnabled()) {
                log.debug("Mcp tools global rewriter=global");
            }
            return this.global.toolsCall(this.<List<Map<String, Object>>>buildMcpContext(dimension, arguments, workTask, result));
        }
        return result;
    }

    public McpResult<List<History>> promptGet(McpDimension dimension, Map<String, Object> arguments, WorkflowTask workTask, McpResult<List<History>> result) throws Exception {
        if (dimension.hasListener()) {
            String key = dimension.getMcpConfig().getRewriter();
            McpRewriter rewriter = this.rewriter.get(key);
            if (log.isDebugEnabled()) {
                log.debug("Mcp prompt rewriter: key={},rewriter={}", key, rewriter);
            }
            Assert.notNull(rewriter, "Mcp prompt get listener can not be empty: " + key);
            return rewriter.promptGet(this.<List<History>>buildMcpContext(dimension, arguments, workTask, result));
        }
        if (this.global != null) {
            if (log.isDebugEnabled()) {
                log.debug("Mcp prompt global rewriter=global");
            }
            return this.global.promptGet(this.<List<History>>buildMcpContext(dimension, arguments, workTask, result));
        }
        return result;
    }

    public McpResult<String> resourcesRead(McpDimension dimension, String uri, WorkflowTask workTask, McpResult<String> result) throws Exception {
        if (dimension.hasListener()) {
            String key = dimension.getMcpConfig().getRewriter();
            McpRewriter rewriter = this.rewriter.get(key);
            if (log.isDebugEnabled()) {
                log.debug("Mcp resources: key={},rewriter={}", key, rewriter);
            }
            Assert.notNull(rewriter, "Mcp resources read listener can not be empty: " + key);
            return rewriter.resourcesRead(this.<String>buildMcpContext(dimension, uri, workTask, result));
        }
        if (this.global != null) {
            if (log.isDebugEnabled()) {
                log.debug("Mcp resource global rewriter=global");
            }
            return this.global.resourcesRead(this.<String>buildMcpContext(dimension, uri, workTask, result));
        }
        return result;
    }

    protected <T> McpContext<T> buildMcpContext(McpDimension dimension, Map<String, Object> arguments, WorkflowTask workTask, McpResult<T> result) throws Exception {
        McpContext<T> context = new McpContext<T>();
        context.setMcpConfig(dimension.getMcpConfig());
        context.setDimension(dimension);
        context.setArguments(arguments);
        context.setWorkTask(workTask);
        context.setResult(result);
        return context;
    }

    protected <T> McpContext<T> buildMcpContext(McpDimension dimension, String uri, WorkflowTask workTask, McpResult<T> result) throws Exception {
        McpContext<T> context = new McpContext<T>();
        context.setMcpConfig(dimension.getMcpConfig());
        context.setDimension(dimension);
        context.setWorkTask(workTask);
        context.setResult(result);
        context.setUri(uri);
        return context;
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired(required = false)
        protected Map<String, McpRewriter> rewriter;

        @Autowired(required = false)
        @Qualifier(McpRewriter.NAME)
        protected McpRewriter global;

        @Bean
        @ConditionalOnMissingBean(value = McpRewriteService.class)
        public McpRewriteService mcpRewriteService() throws Exception {
            McpRewriteServiceImpl mcpRewriteService = new McpRewriteServiceImpl();
            if (!CollectionUtils.isEmpty(this.rewriter)) {
                this.rewriter.remove(McpRewriter.NAME);
            }
            BeanUtils.copyProperties(this, mcpRewriteService);
            log.info("McpRewriteServiceImpl inited, rewriter={}, global={}", mcpRewriteService.getRewriter(), mcpRewriteService.getGlobal());
            return mcpRewriteService;
        }
    }
}
