package ai.open.right.workflow.flow.llm.rag.mcp;

import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAsync;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.mcp.client.McpClientService;
import ai.open.right.workflow.mcp.client.McpResult;
import ai.open.right.workflow.mcp.client.McpRuntime;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.dimension.McpDimensionService;
import ai.open.right.workflow.mcp.client.rewrtier.McpRewriteService;
import ai.open.right.workflow.mcp.client.trigger.McpTriggerService;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.concurrent.Callable;
import java.util.concurrent.ExecutorService;

@Slf4j
@Setter
@Getter
// 使用MCP Resource增强内容
public class RagMcpService extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_mcp";

    protected McpDimensionService mcpDimensionService;

    protected McpRewriteService mcpRewriteService;

    protected McpTriggerService mcpTriggerService;

    protected McpClientService mcpClientService;

    protected ExecutorService executorService;

    // Rag MCP调用下游思考链（Workflow）超时
    protected Integer timeout4Llm;

    // Rag MCP整体超时
    protected Integer timeout;

    protected Boolean allowed(RagConfig ragConfig, RagData ragData) throws Exception {
        return super.allowed(ragConfig, ragData) && ragConfig.hasRagMcp();
    }

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        if (log.isDebugEnabled()) {
            log.debug("Rag mcp start");
        }
        return new RagAsync(ragConfig, this.executorService.submit(new McpFuture(ragConfig, ragData)), ragConfig.getTimeout(this.timeout));
    }

    protected McpDimension buildMcpDimension(RagMcpConfig ragMcpConfig, RagData ragData) throws Exception {
        McpDimension mcpDimension = McpDimension.builder()
                .device(ragData.getQuery().getUserContext().getDevice())
                .workflow(ragData.getQuery().getWorkflow())
                .chat(ragData.getQuery().getChat())
                .biz(ragData.getQuery().getBiz())
                .mcpConfig(ragMcpConfig)
                .build();
        return this.buildMcpDimension(mcpDimension, ragData);
    }

    protected McpDimension buildMcpDimension(McpDimension mcpDimension, RagData ragData) throws Exception {
        return RagMcpService.this.mcpDimensionService.buildDimension(mcpDimension, ragData.getQuery());
    }

    protected McpRuntime buildMcpRuntime(RagMcpConfig ragMcpConfig, RagData ragData) throws Exception {
        return McpRuntime.builder()
                .dynamic(ragMcpConfig.getDynamic())
                .timeout(ragMcpConfig.getTimeout())
                .workTask(ragData.getQuery())
                .build();
    }

    public class McpFuture implements Callable<Void> {

        protected final RagConfig ragConfig;

        protected final RagData ragData;

        public McpFuture(RagConfig ragConfig, RagData ragData) {
            this.ragConfig = ragConfig;
            this.ragData = ragData;
        }

        @Override
        public Void call() throws Exception {
            try {
                RagMcpConfig ragMcpConfig = this.ragConfig.getRagMcpConfig();
                String name = StringUtils.trim(ragMcpConfig.getName());
                if (log.isDebugEnabled()) {
                    log.debug("The rag mcp resource's name={}", name);
                }
                if (ragMcpConfig.hasDynamic()) {
                    // 动态选择MCP
                    SyncConfig syncConfig = SyncConfig.builder()
                            .timeout(this.ragConfig.getTimeout4Llm(RagMcpService.this.timeout4Llm))
                            .workflow(ragMcpConfig.getDynamic())
                            .workTask(this.ragData.getQuery())
                            .build();
                    name = SyncWorkflowTask.exeWorkflow(RagMcpService.this.getNotifierService(), syncConfig).get();
                    if (log.isDebugEnabled()) {
                        log.debug("The rag mcp resource's name is determined after dynamic selection={}", name);
                    }
                }
                McpDimension mcpDimension = RagMcpService.this.buildMcpDimension(this.ragConfig.getRagMcpConfig(), this.ragData);
                // 触发MCP Resources Read
                RagMcpService.this.mcpTriggerService.beforeResourcesRead(mcpDimension, name, this.ragData.getQuery());
                // 调用MCP并使用MCP Listener rewrite
                McpResult<String> result = RagMcpService.this.mcpClientService.resourcesRead(mcpDimension.getClient(), name, RagMcpService.this.buildMcpRuntime(ragMcpConfig, this.ragData), mcpDimension);
                result = RagMcpService.this.mcpRewriteService.resourcesRead(mcpDimension, name, this.ragData.getQuery(), result);
                String resource = result.getResult();
                if (log.isInfoEnabled()) {
                    log.info("The rag mcp resource's result={}", resource);
                }
                RagService.updatePrompt(this.ragConfig, this.ragData, this.ragConfig.getReplace(), resource);
                return null;
            } catch (Exception e) {
                if (this.ragConfig.getStopOnFailed()) {
                    throw e;
                } else {
                    if (log.isInfoEnabled()) {
                        log.info(e.getMessage(), e);
                    }
                    return null;
                }
            }
        }
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Autowired
        protected McpDimensionService mcpDimensionService;

        @Autowired
        protected McpRewriteService mcpRewriteService;

        @Autowired
        protected McpTriggerService mcpTriggerService;

        @Autowired
        protected McpClientService mcpClientService;

        @Autowired
        @Qualifier("executor")
        protected ExecutorService executorService;

        @Value("${mcp.timeout.llm:1800000}")
        // Rag MCP调用下游思考链（Workflow）超时
        protected Integer timeout4Llm;

        @Value("${mcp.timeout:1800000}")
        // Rag MCP整体超时
        protected Integer timeout;

        @Bean(RagMcpService.RAG_KEY)
        @ConditionalOnMissingBean(name = RagMcpService.RAG_KEY)
        public RagMcpService ragMcpService() throws Exception {
            RagMcpService ragMcpService = new RagMcpService();
            BeanUtils.copyProperties(this, ragMcpService);
            log.info("RagMcpService inited: timeout={},timeout4Llm={},timeout4Condition={}", ragMcpService.getTimeout(), ragMcpService.getTimeout4Llm(), ragMcpService.getTimeout4Condition());
            return ragMcpService;
        }
    }
}
