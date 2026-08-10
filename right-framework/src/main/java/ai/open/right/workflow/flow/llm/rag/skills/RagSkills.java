package ai.open.right.workflow.flow.llm.rag.skills;

import ai.open.right.WorkflowException;
import ai.open.right.utils.YamlUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.skill.SkillMetadata;
import ai.open.right.workflow.skill.Skills;
import ai.open.right.workflow.skill.SkillsFetcher;
import com.google.common.cache.Cache;
import com.google.common.cache.CacheBuilder;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.util.Collection;
import java.util.concurrent.Callable;
import java.util.concurrent.TimeUnit;

@Slf4j
@Setter
@Getter
public class RagSkills extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_skills";

    protected Cache<String, Skills> skillsCache;

    protected SkillsFetcher skillFetcher;

    protected Integer repeat;

    protected Integer expire;

    @PostConstruct
    public void init() throws Exception {
        this.skillsCache = CacheBuilder.newBuilder()
                .expireAfterWrite(this.expire, TimeUnit.MILLISECONDS)
                .maximumSize(Integer.MAX_VALUE).build();
    }

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        if (log.isDebugEnabled()) {
            log.debug("Rag skills start");
        }
        try {
            // 从缓存读取
            this.updatePrompt(ragConfig, ragData, this.skillsCache.get(this.key(ragConfig, ragData), new SkillsCallable(ragConfig.hasRagSkills() ? ragConfig.getRagSkillsConfig() : null, ragData.getQuery())));
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
        return new RagAtOnce(ragConfig);
    }

    protected String buildMetadata(RagConfig ragConfig, RagData ragData, Collection<SkillMetadata> skillMetadata) throws Exception {
        if (!CollectionUtils.isEmpty(skillMetadata)) {
            StringBuffer builder = new StringBuffer();
            builder.append(StringUtils.repeat("-", this.repeat)).append(System.lineSeparator());
            for (SkillMetadata each : skillMetadata) {
                // 技能可见度检查
                if (this.allowedSkill(ragConfig, ragData, each)) {
                    builder.append(YamlUtils.write(each));
                    builder.append(StringUtils.repeat("-", this.repeat)).append(System.lineSeparator());
                }
            }
            return builder.toString();
        } else {
            return "";
        }
    }

    protected Boolean allowedSkill(RagConfig ragConfig, RagData ragData, SkillMetadata skill) throws Exception {
        return this.allowedSkill(ragConfig, ragData, skill.getName());
    }

    protected Boolean allowedSkill(RagConfig ragConfig, RagData ragData, String skill) throws Exception {
        RagSkillsConfig ragSkillsConfig = ragConfig.getRagSkillsConfig();
        return ragSkillsConfig != null ? ragSkillsConfig.allowed(skill) : true;
    }

    // 构建Skill
    protected Object buildSkills(RagConfig ragConfig, RagData ragData, Skills skills) throws Exception {
        // Use markdown
        StringBuffer builder = new StringBuffer();
        builder.append(this.buildUsage(ragConfig, ragData, skills));
        builder.append(this.buildMetadata(ragConfig, ragData, skills.getSkills().values()));
        return builder.toString();
    }

    protected String buildUsage(RagConfig ragConfig, RagData ragData, Skills skills) throws Exception {
        if (!StringUtils.isEmpty(skills.getUsage())) {
            StringBuffer builder = new StringBuffer();
            builder.append(System.lineSeparator()).append("```").append("SKILLS_USAGE").append(System.lineSeparator());
            builder.append(skills.getUsage());
            builder.append(System.lineSeparator()).append("```").append(System.lineSeparator());
            return builder.toString();
        } else {
            return "";
        }
    }

    protected void updatePrompt(RagConfig ragConfig, RagData ragData, Skills skills) throws Exception {
        RagService.updatePrompt(ragConfig, ragData, ragConfig.getReplace(), this.buildSkills(ragConfig, ragData, skills));
    }

    // 子类覆盖
    protected String key(RagConfig ragConfig, RagData ragData) throws Exception {
        return "";
    }

    public class SkillsCallable implements Callable<Skills> {

        protected final RagSkillsConfig ragSkillsConfig;

        protected final WorkflowTask workTask;

        public SkillsCallable(RagSkillsConfig ragSkillsConfig, WorkflowTask workTask) {
            this.ragSkillsConfig = ragSkillsConfig;
            this.workTask = workTask;
        }

        public Skills call() throws Exception {
            Assert.notNull(RagSkills.this.skillFetcher, "The skill fetcher can not be empty, please config `skills.enable`");
            return RagSkills.this.skillFetcher.fetchSkills(this.workTask, this.ragSkillsConfig);
        }
    }

    @ConditionalOnProperty(name = "skills.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Autowired(required = false)
        protected SkillsFetcher skillFetcher;

        @Value("${skills.expire:1800000}")
        protected Integer expire;

        @Value("${skills.repeat:5}")
        protected Integer repeat;

        @Bean(RagSkills.RAG_KEY)
        @ConditionalOnMissingBean(name = RagSkills.RAG_KEY)
        public RagSkills ragSkills() throws Exception {
            RagSkills ragSkills = new RagSkills();
            BeanUtils.copyProperties(this, ragSkills);
            log.info("RagSkills inited. timeout4Condition={}", ragSkills.getTimeout4Condition());
            return ragSkills;
        }
    }
}
