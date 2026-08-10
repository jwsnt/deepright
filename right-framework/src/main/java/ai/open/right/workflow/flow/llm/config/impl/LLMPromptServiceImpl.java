package ai.open.right.workflow.flow.llm.config.impl;

import ai.open.right.WorkflowException;
import ai.open.right.workflow.config.PromptSearch;
import ai.open.right.workflow.config.PromptService;
import ai.open.right.workflow.config.impl.PromptServiceImpl;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.LLMPromptService;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.apache.commons.lang3.builder.ToStringBuilder;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

@Slf4j
@Setter
@Getter
public class LLMPromptServiceImpl implements LLMPromptService {

    // Key => Rag
    protected Map<String, RagService> ragService;

    protected PromptService promptService;

    @Override
    public String prompt(ProviderRequest request, LLMConfig config, LLMQuery query) throws Exception {
        PromptSearch promptSearch = PromptSearch.builder().llmConfig(config).workTask(query).build();
        String prompt = this.promptService.search(promptSearch).getContent();
        // 没有配置动态System Prompt且获取不到
        if (!config.hasDynamicPrompt() && StringUtils.isEmpty(prompt)) {
            throw new WorkflowException("Prompt can not be empty. promptSearch=" + promptSearch);
        }
        if (log.isDebugEnabled()) {
            log.debug("Find prompt={}-{}", promptSearch, prompt);
        }
        // 使用Rag改写System Prompt或Query
        String actualPrompt = this.ragDoing(request, config, query, prompt);
        if (log.isDebugEnabled()) {
            log.debug("Rewrite prompt={}", actualPrompt);
        }
        return actualPrompt;
    }

    protected void ragGroup(LLMConfig config, List<RagFuture> future, RagData data) throws Exception {
        Assert.notNull(this.ragService, "The rag service can not be empty");
        // sort值越大排越前面，越早执行
        for (RagConfig rConfig : this.sortConfig(config, data)) {
            try {
                RagService ragService = this.ragService.get(rConfig.getKey().toLowerCase());
                Assert.notNull(ragService, "Can not find rag: " + rConfig.getKey());
                future.add(ragService.rag(rConfig, data));
            } catch (Exception e) {
                // 如果同步Rag失败需要中断Workflow
                if (!rConfig.getStopOnFailed()) {
                    log.warn(e.getMessage(), e);
                } else {
                    throw e;
                }
            }
        }
    }

    // 使用Rag改写Prompt
    protected String ragDoing(ProviderRequest request, LLMConfig config, LLMQuery query, String prompt) throws Exception {
        if (CollectionUtils.isEmpty(config.getRagConfig())) {
            return prompt;
        }
        List<RagFuture> ragFuture = new ArrayList<RagFuture>();
        RagData ragData = RagData.builder().request(request).config(config).query(query).prompt(prompt).build();
        this.ragGroup(config, ragFuture, ragData);
        // 执行，并检查是否抛出异常
        this.ragEach(ragFuture);
        // 更新最新Prompt
        return ragData.getPrompt();
    }

    protected List<RagConfig> sortConfig(LLMConfig llmConfig, RagData ragData) throws Exception {
        return llmConfig.getRagConfig().stream()
                .sorted(Comparator.comparingInt(RagConfig::getSort).reversed())
                .collect(Collectors.toList());
    }

    // 处理每个Rag
    protected void ragEach(List<RagFuture> ragFuture) throws Exception {
        // 执行Rag
        for (RagFuture future : ragFuture) {
            try {
                future.run();
            } catch (Exception e) {
                // 如果异步Rag失败需要中断Workflow
                if (!future.config().getStopOnFailed()) {
                    log.warn(e.getMessage(), e);
                } else {
                    throw e;
                }
            }
        }
    }
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        // Key => Rag
        @Autowired(required = false)
        protected Map<String, RagService> ragService;

        @Autowired
        @Qualifier(PromptServiceImpl.NAME)
        protected PromptService promptService;

        @Bean
        @ConditionalOnMissingBean(value = LLMPromptService.class)
        public LLMPromptService llmPromptService() throws Exception {
            LLMPromptServiceImpl llmPromptService = new LLMPromptServiceImpl();
            BeanUtils.copyProperties(this, llmPromptService);
            log.info("LLMPromptServiceImpl inited={}", ToStringBuilder.reflectionToString(llmPromptService));
            return llmPromptService;
        }
    }
}
