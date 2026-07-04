package ai.deepright.skills;

import ai.deepright.feature.FeatureFlag;
import ai.deepright.feature.FeatureUtils;
import ai.deepright.utils.TemplateChecker;
import ai.open.right.resouce.ResourceService;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.skills.RagSkills;
import ai.open.right.workflow.skill.SkillMetadata;
import ai.open.right.workflow.skill.Skills;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.io.BufferedInputStream;
import java.nio.charset.StandardCharsets;
import java.util.Collection;
import java.util.HashMap;
import java.util.Map;
import java.util.stream.Collectors;

@Slf4j
@Getter
@Setter
public class SkillsSchemaRag extends RagSkills {

    protected Map<String, String> activePlugins = new HashMap<String, String>();

    protected ResourceService resourceService;

    protected String template4extract;

    protected String template4init;

    protected String skillCreator;

    @PostConstruct
    public void init() throws Exception {
        super.init();
        // IOUtils/JsonUtils负责关闭资源
        this.template4extract = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4extract).openStream()), StandardCharsets.UTF_8);
        this.template4init = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4init).openStream()), StandardCharsets.UTF_8);
        // 覆盖（rewrite），不需要重入，启动检测，必要资源
        Assert.hasText(this.template4extract, "The template template4extract must not be empty");
        Assert.hasText(this.template4init, "The template init must not be empty");
        this.activePlugins.put("__internal_browser", "browser");
        this.activePlugins.put("__internal_remote", "remote");
    }

    @Override
    protected String buildMetadata(RagConfig ragConfig, RagData ragData, Collection<SkillMetadata> skillMetadata) throws Exception {
        if (!FeatureFlag.isSkillExtract(ragData.getQuery())) {
            // 如果不是提取Skills，需要过滤
            return super.buildMetadata(ragConfig, ragData, skillMetadata.stream()
                    .filter(skill -> !StringUtils.equalsIgnoreCase(this.skillCreator, skill.getName()))
                    .collect(Collectors.toList()));
        } else {
            return super.buildMetadata(ragConfig, ragData, skillMetadata);
        }
    }

    @Override
    protected Boolean allowedSkill(RagConfig ragConfig, RagData ragData, SkillMetadata skill) throws Exception {
        // 插件Skills需要判断激活, 如果插件没激活则不加载技能
        return this.activePlugins.containsKey(skill.getName()) ? FeatureFlag.isActivePlugin(ragData.getQuery(), this.activePlugins.get(skill.getName())) : super.allowedSkill(ragConfig, ragData, skill);
    }

    @Override
    protected Object buildSkills(RagConfig ragConfig, RagData ragData, Skills skills) throws Exception {
        // Use markdown
        StringBuffer buffer = new StringBuffer();
        buffer.append(this.buildUsage(ragConfig, ragData, skills));
        buffer.append(this.buildMetadata(ragConfig, ragData, skills.getSkills().values()));
        String query = this.template4init.replace(ragConfig.getReplace(), buffer.toString());
        query = query.replace("#skill_extract", this.isSkillExtract(ragConfig, ragData, skills) ? this.template4extract.replace("#workspace", FeatureUtils.buildWorkspace(ragData.getQuery())).replace("#skill_create", this.skillCreator) : "");
        if (log.isWarnEnabled() && !TemplateChecker.check(query)) {
            log.warn("The query template contains unexpected characters, please check: {}", query);
        }
        return System.lineSeparator() + query + System.lineSeparator();
    }

    @Override
    protected void updatePrompt(RagConfig ragConfig, RagData ragData, Skills skills) throws Exception {
        // 当上下文超过10K tokens时，模型对尾部指令的注意力权重下降
        RagService.updatePrompt(ragConfig, ragData, ragConfig.getReplace(), this.buildSkills(ragConfig, ragData, skills));
    }

    protected Boolean isSkillExtract(RagConfig ragConfig, RagData ragData, Skills skills) throws Exception {
        return FeatureFlag.isSkillExtract(ragData.getQuery()) && MapUtils.getObject(skills.getSkills(), this.skillCreator) != null;
    }

    @Configuration
    @Setter
    @Getter
    public static class SkillsSchemaInitConfig extends InitConfig {

        @Autowired
        protected ResourceService resourceService;

        @Value("${skills.schema.template.extract:classpath:config/skills/extract.md}")
        protected String template4extract;

        @Value("${skills.schema.template.init:classpath:config/skills/main.md}")
        protected String template4init;

        @Value("${skills.schema.creator:skill-creator}")
        protected String skillCreator;

        @Override
        @Bean(RagSkills.RAG_KEY)
        @ConditionalOnMissingBean(name = RagSkills.RAG_KEY)
        public SkillsSchemaRag ragSkills() throws Exception {
            SkillsSchemaRag schemaSkillsRag = new SkillsSchemaRag();
            BeanUtils.copyProperties(this, schemaSkillsRag);
            log.info("SchemaSkillsRag inited. timeout4Condition={}", schemaSkillsRag.getTimeout4Condition());
            return schemaSkillsRag;
        }
    }
}
