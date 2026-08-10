package ai.open.right.workflow.flow.llm.rag.query;

import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.config.PromptSearch;
import ai.open.right.workflow.config.PromptService;
import ai.open.right.workflow.config.impl.PromptServiceImpl;
import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
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
import org.springframework.util.StringUtils;

import java.util.Arrays;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

@Slf4j
@Setter
@Getter
// 使用静态Prompt增强内容
public class RagStaticQuery extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_query_static";

    protected PromptService promptService;

    protected Boolean allowed(RagConfig ragConfig, RagData ragData) throws Exception {
        return super.allowed(ragConfig, ragData) && StringUtils.hasText(ragConfig.getTemplate());
    }

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        if (log.isDebugEnabled()) {
            log.debug("Rag static query start");
        }
        PromptSearch promptSearch = PromptSearch.builder()
                .llmConfig(ragData.getConfig())
                .workTask(ragData.getQuery())
                .build();
        String[] pair = SplitUtils.split(ragConfig.getTemplate(), ragData.getQuery().getBiz());
        promptSearch.setPrompt(pair[1]);
        promptSearch.setBiz(pair[0]);
        if (log.isDebugEnabled()) {
            log.debug("The rag static prompt's dimension={}", Arrays.toString(pair));
        }
        // 查找Prompt
        String query = this.promptService.get(promptSearch).getContent();
        Assert.hasText(query, "Query can not be empty");
        if (log.isDebugEnabled()) {
            log.debug("The original rag static prompt={}", query);
        }
        if (ragConfig.isOverride()) {
            // 是否替换占位符，精确匹配
            query = query.replace(ragConfig.getReplace(), ragData.getQuery().getQuery());
            if (log.isInfoEnabled()) {
                log.info("The replaced rag static prompt={}", query);
            }
        }
        RagService.updatePrompt(ragConfig, ragData, null, query);
        return new RagAtOnce(ragConfig);
    }

    @ConditionalOnProperty(name = "query.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Autowired
        @Qualifier(PromptServiceImpl.NAME)
        protected PromptService promptService;

        @Bean(RagStaticQuery.RAG_KEY)
        @ConditionalOnMissingBean(name = RagStaticQuery.RAG_KEY)
        public RagStaticQuery ragStaticQuery() throws Exception {
            RagStaticQuery ragStaticQuery = new RagStaticQuery();
            BeanUtils.copyProperties(this, ragStaticQuery);
            log.info("RagStaticQuery inited, timeout4Condition={}", ragStaticQuery.getTimeout4Condition());
            return ragStaticQuery;
        }
    }
}
