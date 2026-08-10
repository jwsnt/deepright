package ai.open.right.workflow.flow.config.impl;

import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.config.Config;
import ai.open.right.workflow.config.ConfigSearch;
import ai.open.right.workflow.config.ConfigService;
import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.config.impl.ConfigServiceImpl;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.assistant.mcp.McpPromptGetAssistant;
import ai.open.right.workflow.flow.assistant.mcp.McpResourceReadAssistant;
import ai.open.right.workflow.flow.assistant.mcp.McpToolsCallAssistant;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.config.WorkflowConfigService;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.LLMFunCall;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.slf4j.MDC;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
@Setter
@Getter
public class WorkflowConfigServiceImpl implements WorkflowConfigService {

    protected ConfigService configService;

    protected NamesService namesService;

    // 默认使用的模型供应商
    protected String provider;

    @Override
    public WorkflowConfig config(ConfigSearch configSearch, String workflow) throws Exception {
        // UpStream重置Biz，否则使用当前Biz
        configSearch.setBiz(SplitUtils.biz(workflow, configSearch.getBiz()));
        if (log.isDebugEnabled()) {
            log.debug("Find config={}", configSearch);
        }
        Config config = this.configService.get(configSearch);
        Object source = config.getConfigs().get(workflow);
        WorkflowConfig workflowConfig = source != null ? JsonUtils.transfer(source, WorkflowConfig.class) : new WorkflowConfig();
        this.configLLMFunCall(workflowConfig, workflow);
        this.configLLMConfig(workflowConfig, workflow);
        workflowConfig = workflowConfig.init();
        this.configExtended(workflowConfig, configSearch.getBiz());
        return workflowConfig;
    }

    @Override
    public WorkflowConfig config(WorkflowTask workTask, String workflow) throws Exception {
        // 检查是否为Native MCP配置
        WorkflowConfig workflowConfig = this.buildMcpConfig(workflow);
        if (workflowConfig != null) {
            if (log.isDebugEnabled()) {
                log.debug("Build the native mcp config");
            }
            return workflowConfig;
        }
        // 检查是否为思考链（Workflow）包装的MCP配置
        if (this.namesService.isPrefixWorkflow(workflow)) {
            // 解码Biz/Workflow
            String[] pair = this.namesService.decode(workflow);
            workTask.setBiz(pair[0]);
            workTask.setWorkflow(pair[1]);
            // 更新入参Workflow
            workflow = workTask.getWorkflow();
            // 更新MDC
            MDC.put("dimension", workTask.getDimension());
        }
        ConfigSearch configSearch = ConfigSearch.builder()
                .language(workTask.getUserContext().getLanguage())
                .device(workTask.getUserContext().getDevice())
                // this.config会为Upstream重置BIZ
                .biz(workTask.getBiz())
                .build();
        return this.config(configSearch, workflow);
    }

    @Override
    public WorkflowConfig config(String biz, String workflow) throws Exception {
        return this.config(ConfigSearch.builder()
                .biz(biz)
                .build(), workflow);
    }

    @Override
    public WorkflowConfig config(WorkflowTask workTask) throws Exception {
        return this.config(workTask, workTask.getWorkflow());
    }

    protected void configLLMFunCall(WorkflowConfig workflowConfig, String workflow) throws Exception {
        if (!workflowConfig.hasFunCall()) {
            workflowConfig.setLlmFunCall(this.buildDefaultLLMFunCall(workflow));
        }
    }

    protected void configLLMConfig(WorkflowConfig workflowConfig, String workflow) throws Exception {
        if (!workflowConfig.hasLlm()) {
            // 使用默认LLM
            workflowConfig.setLlmConfig(this.buildDefaultLLMConfig(workflow, workflowConfig.getChain()));
        } else if (workflowConfig.getLlmConfig().getProvider() == null) {
            // 没有指定模型供应商则使用默认
            workflowConfig.getLlmConfig().setProvider(this.provider);
        }
    }

    // 配置继承
    protected void configExtended(WorkflowConfig workflowConfig, String biz) throws Exception {
        if (workflowConfig.hasExtended()) {
            for (String config : StringUtils.split(workflowConfig.getExtended(), WorkflowConfig.EXTENDED)) {
                String[] part = SplitUtils.split(config, biz);
                WorkflowConfig target = this.config(part[0], part[1]);
                if (log.isDebugEnabled()) {
                    log.debug("The extended config={}", target);
                }
                workflowConfig.merge(target);
            }
        }
    }

    public LLMConfig buildDefaultLLMConfig(String workflow, String chain) throws Exception {
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setStream(LLMConfig.STREAM);
        llmConfig.setProvider(this.provider);
        llmConfig.setPrompt(workflow);
        llmConfig.init(chain);
        return llmConfig;
    }

    public LLMFunCall buildDefaultLLMFunCall(String workflow) throws Exception {
        return new LLMFunCall();
    }

    public WorkflowConfig buildMcpConfig(String workflow) throws Exception {
        // 检查是否为Native MCP的前缀（Workflow）：Resource/Prompt/Tools，并选择对应Workflow
        String mcpWorkflow = this.namesService.isPrefixResource(workflow) ? McpResourceReadAssistant.WORKFLOW_NAME : this.namesService.isPrefixPrompt(workflow) ? McpPromptGetAssistant.WORKFLOW_NAME : this.namesService.isPrefixTools(workflow) ? McpToolsCallAssistant.WORKFLOW_NAME : null;
        if (!StringUtils.isEmpty(mcpWorkflow)) {
            WorkflowConfig workflowConfig = new WorkflowConfig();
            workflowConfig.setLlmConfig(this.buildDefaultLLMConfig(mcpWorkflow, workflowConfig.getChain()));
            workflowConfig.getLlmConfig().setProvider(this.provider);
            workflowConfig.setAssistant(mcpWorkflow);
            return workflowConfig.init();
        } else {
            return null;
        }
    }
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        @Qualifier(ConfigServiceImpl.NAME)
        protected ConfigService configService;

        @Autowired
        protected NamesService namesService;

        @Value("${request.provider:deepseek}")
        // 默认使用的模型供应商
        protected String provider;

        @Bean
        @ConditionalOnMissingBean(value = WorkflowConfigService.class)
        public WorkflowConfigService workflowConfigService() throws Exception {
            WorkflowConfigServiceImpl workflowConfigService = new WorkflowConfigServiceImpl();
            BeanUtils.copyProperties(this, workflowConfigService);
            log.info("WorkflowConfigService inited: provider={}", workflowConfigService.getProvider());
            return workflowConfigService;
        }
    }
}
