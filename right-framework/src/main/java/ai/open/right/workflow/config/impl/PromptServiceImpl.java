package ai.open.right.workflow.config.impl;

import ai.open.right.workflow.config.Prompt;
import ai.open.right.workflow.config.PromptSearch;
import ai.open.right.workflow.config.PromptService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.util.Map;

@Setter
@Getter
@Slf4j
public class PromptServiceImpl implements PromptService {

    public static final String NAME = "PromptServiceImpl";

    protected Map<String, PromptService> promptService;

    // Prompt配置的加载方式，默认为FilePromptService（prompt.file）
    protected String instance;

    protected PromptService instance(PromptSearch promptSearch) throws Exception {
        // 是否为动态化System Prompt
        return promptSearch.getLlmConfig().hasDynamicPrompt() ? this.buildDynamic() : this.instance();
    }

    protected PromptService instance() throws Exception {
        PromptService promptService = this.promptService.get(StringUtils.trim(this.instance));
        Assert.notNull(promptService, "PromptService can not be empty: " + this.instance);
        return promptService;
    }

    @Override
    public Prompt get(PromptSearch promptSearch) throws Exception {
        return this.instance(promptSearch).get(promptSearch);
    }

    @Override
    public Prompt search(PromptSearch promptSearch) throws Exception {
        return this.instance(promptSearch).search(promptSearch);
    }

    protected PromptService buildDynamic() throws Exception {
        Assert.notNull(this.promptService, "The dynamic prompt can not be empty");
        return this.promptService.get(DyPromptService.NAME);
    }
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected Map<String, PromptService> promptService;

        @Value("${prompt:prompt.file}")
        // Prompt配置的加载方式，默认为FilePromptService（prompt.file）
        protected String instance;

        @Bean(name = PromptServiceImpl.NAME)
        @ConditionalOnMissingBean(name = PromptServiceImpl.NAME)
        public PromptService promptService() throws Exception {
            PromptServiceImpl promptService = new PromptServiceImpl();
            BeanUtils.copyProperties(this, promptService);
            log.info("PromptServiceImpl inited: instance={}", promptService.getInstance());
            return promptService;
        }
    }
}
