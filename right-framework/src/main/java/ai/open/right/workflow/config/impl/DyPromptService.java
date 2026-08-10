package ai.open.right.workflow.config.impl;

import ai.open.right.workflow.config.Prompt;
import ai.open.right.workflow.config.PromptSearch;
import ai.open.right.workflow.config.PromptService;
import ai.open.right.workflow.flow.llm.config.LLMDynamic;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import ai.open.right.workflow.sync.impl.NotifierCallable;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

@Slf4j
@Setter
@Getter
// 动态生成Prompt
public class DyPromptService implements PromptService {

    public static final String NAME = "prompt.dynamic";

    protected NotifierService notifierService;

    // 调用下游思考链（Workflow）生成System Prompt的超时
    protected Integer timeout;

    @Override
    public Prompt get(PromptSearch promptSearch) throws Exception {
        return this.search(promptSearch);
    }

    @Override
    public Prompt search(PromptSearch promptSearch) throws Exception {
        LLMDynamic llmDynamic = promptSearch.getLlmConfig().getDynamic();
        try {
            SyncConfig syncConfig = SyncConfig.builder()
                    // 如果指定了Notifier（Localhost、Endpoint、Source）
                    .syncCallable(llmDynamic.hasNotifier() ? new NotifierCallable(llmDynamic.getNotifier()) : null)
                    .timeout(llmDynamic.getTimeout(this.timeout))
                    .workTask(promptSearch.getWorkTask())
                    // 下游的Prompt配置名
                    .workflow(llmDynamic.getDynamic())
                    .build();
            String dynamicPrompt = SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get();
            if (log.isDebugEnabled()) {
                log.debug("Dynamic prompt={}", dynamicPrompt);
            }
            Assert.hasText(dynamicPrompt, "Dynamic prompt can not be empty, please check workflow config: " + promptSearch);
            // 构建Dynamic特殊Prompt
            Prompt prompt = new Prompt(promptSearch.getBiz(), DyPromptService.NAME, dynamicPrompt);
            Prompt.PromptChecker.check(prompt);
            return prompt;
        } catch (Exception e) {
            // 任意异常，如果配置为不终止，则使用上游Query作为Prompt（Query与System Prompt一致）
            if (!llmDynamic.getStopOnFailed()) {
                if (log.isWarnEnabled()) {
                    log.warn(e.getMessage(), e);
                }
                // 使用上游思考链（Workflow）的Workflow和Query
                return new Prompt(promptSearch.getBiz(), promptSearch.getWorkTask().getWorkflow(), promptSearch.getWorkTask().getQuery());
            } else {
                throw e;
            }
        }
    }

    @ConditionalOnProperty(name = "prompt.dynamic.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected NotifierService notifierService;

        // 调用下游思考链（Workflow）生成System Prompt的超时
        @Value("${prompt.timeout:1800000}")
        protected Integer timeout;

        @Bean(name = DyPromptService.NAME)
        @ConditionalOnMissingBean(name = DyPromptService.NAME)
        public PromptService dyPromptService() throws Exception {
            DyPromptService dyPromptService = new DyPromptService();
            BeanUtils.copyProperties(this, dyPromptService);
            log.info("DyPromptService inited: timeout={}", dyPromptService.getTimeout());
            return dyPromptService;
        }
    }
}
