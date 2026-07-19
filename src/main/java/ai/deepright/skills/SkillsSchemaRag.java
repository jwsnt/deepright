package ai.deepright.skills;

import ai.deepright.feature.FeatureFlag;
import ai.deepright.llm.provider.RequestProviderUtils;
import ai.deepright.utils.TemplateChecker;
import ai.open.right.WorkflowException;
import ai.open.right.resouce.ResourceService;
import ai.open.right.workflow.flow.WorkflowTask;
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

import java.io.BufferedInputStream;
import java.nio.charset.StandardCharsets;
import java.util.Collection;
import java.util.HashMap;
import java.util.Map;
import java.util.stream.Collectors;

@Slf4j
@Getter
@Setter
public class SkillsSchemaRag extends RagSkills implements SkillsChecker {

    protected Map<String, String> activePlugins = new HashMap<String, String>();

    protected ResourceService resourceService;

    protected String template4browser;

    protected String template4creator;

    protected String template4remote;

    protected String template4feishu;

    protected String template4miniapp;

    protected String template4usage;

    protected String template4email;

    protected String template4image;

    protected String template4html;

    protected String template4init;

    protected String skillDeepRight;

    protected String skillMiniApp;

    protected String skillCreator;

    protected String skillFeishu;

    @PostConstruct
    public void init() throws Exception {
        super.init();
        // IOUtils/JsonUtils负责关闭资源
        this.template4miniapp = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4miniapp).openStream()), StandardCharsets.UTF_8);
        this.template4browser = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4browser).openStream()), StandardCharsets.UTF_8);
        this.template4creator = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4creator).openStream()), StandardCharsets.UTF_8);
        this.template4remote = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4remote).openStream()), StandardCharsets.UTF_8);
        this.template4feishu = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4feishu).openStream()), StandardCharsets.UTF_8);
        this.template4usage = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4usage).openStream()), StandardCharsets.UTF_8);
        this.template4email = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4email).openStream()), StandardCharsets.UTF_8);
        this.template4image = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4image).openStream()), StandardCharsets.UTF_8);
        this.template4html = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4html).openStream()), StandardCharsets.UTF_8);
        this.template4init = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4init).openStream()), StandardCharsets.UTF_8);
        // 覆盖（rewrite），不需要重入，启动检测，必要资源
        WorkflowException.checkCondition(StringUtils.isEmpty(this.template4miniapp), "The template miniapp must not be empty");
        WorkflowException.checkCondition(StringUtils.isEmpty(this.template4browser), "The template browser must not be empty");
        WorkflowException.checkCondition(StringUtils.isEmpty(this.template4creator), "The template extract must not be empty");
        WorkflowException.checkCondition(StringUtils.isEmpty(this.template4remote), "The template remote must not be empty");
        WorkflowException.checkCondition(StringUtils.isEmpty(this.template4feishu), "The template feishu must not be empty");
        WorkflowException.checkCondition(StringUtils.isEmpty(this.template4email), "The template email must not be empty");
        WorkflowException.checkCondition(StringUtils.isEmpty(this.template4image), "The template image must not be empty");
        WorkflowException.checkCondition(StringUtils.isEmpty(this.template4usage), "The template usage must not be empty");
        WorkflowException.checkCondition(StringUtils.isEmpty(this.template4html), "The template html must not be empty");
        WorkflowException.checkCondition(StringUtils.isEmpty(this.template4init), "The template init must not be empty");
        this.activePlugins.put(SkillsChecker.PLUGIN_BROWSER_SKILL, SkillsChecker.PLUGIN_BROWSER_NAME);
        this.activePlugins.put(SkillsChecker.PLUGIN_REMOTE_SKILL, SkillsChecker.PLUGIN_REMOTE_NAME);
        this.activePlugins.put(SkillsChecker.PLUGIN_FEISHU_SKILL, SkillsChecker.PLUGIN_FEISHU_NAME);
        this.activePlugins.put(SkillsChecker.PLUGIN_EMAIL_SKILL, SkillsChecker.PLUGIN_EMAIL_NAME);
    }

    @Override
    public Boolean allowedSkill(WorkflowTask workTask, String skill) throws Exception {
        // 插件Skills需要判断激活, 如果插件没激活则不加载技能
        return this.activePlugins.containsKey(skill) && FeatureFlag.isActivePlugin(workTask, this.activePlugins.get(skill));
    }

    @Override
    protected String buildMetadata(RagConfig ragConfig, RagData ragData, Collection<SkillMetadata> skillMetadata) throws Exception {
        if (!FeatureFlag.isSkillExtract(ragData.getQuery())) {
            // 如果不是提取Skills，需要过滤掉"__internal_creator"
            return super.buildMetadata(ragConfig, ragData, skillMetadata.stream()
                    .filter(skill -> !StringUtils.equalsIgnoreCase(this.skillCreator, skill.getName()))
                    .collect(Collectors.toList()));
        } else {
            return super.buildMetadata(ragConfig, ragData, skillMetadata);
        }
    }

    @Override
    protected Boolean allowedSkill(RagConfig ragConfig, RagData ragData, String skill) throws Exception {
        // 插件Skills需要判断激活, 如果插件没激活则不加载技能
        return this.activePlugins.containsKey(skill) ? this.allowedSkill(ragData.getQuery(), skill) : super.allowedSkill(ragConfig, ragData, skill);
    }

    @Override
    protected Object buildSkills(RagConfig ragConfig, RagData ragData, Skills skills) throws Exception {
        // Use markdown
        StringBuffer buffer = new StringBuffer();
        buffer.append(this.buildUsage(ragConfig, ragData, skills));
        buffer.append(this.buildMetadata(ragConfig, ragData, skills.getSkills().values()));
        String query = this.template4init.replace(ragConfig.getReplace(), buffer.toString());
        query = this.buildUsage(ragConfig, ragData, skills, query);
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

    protected String buildUsage(RagConfig ragConfig, RagData ragData, Skills skills, String query) throws Exception {
        String usage = this.template4usage.replace("#feishu", FeatureFlag.isActivePlugin(ragData.getQuery(), SkillsChecker.PLUGIN_FEISHU_NAME) ? this.template4feishu.replace("#deepright", this.skillDeepRight).replace("#feishu", this.skillFeishu) : "");
        usage = usage.replace("#browser", this.allowedSkill(ragConfig, ragData, SkillsChecker.PLUGIN_BROWSER_SKILL) ? this.template4browser.replace("#browser", SkillsChecker.PLUGIN_BROWSER_SKILL) : "");
        usage = usage.replace("#remote", this.allowedSkill(ragConfig, ragData, SkillsChecker.PLUGIN_REMOTE_SKILL) ? this.template4remote.replace("#remote", SkillsChecker.PLUGIN_REMOTE_SKILL) : "");
        usage = usage.replace("#miniapp", this.template4miniapp.replace("#miniapp", this.skillMiniApp).replace("#html", FeatureFlag.isHtml(ragData.getQuery()) ? this.template4html : ""));
        usage = usage.replace("#email", FeatureFlag.isActivePlugin(ragData.getQuery(), SkillsChecker.PLUGIN_EMAIL_NAME) ? this.template4email.replace("#deepright", this.skillDeepRight) : "");
        usage = usage.replace("#creator", this.isSkillExtract(ragConfig, ragData, skills) ? this.template4creator.replace("#creator", this.skillCreator) : "");
        usage = usage.replace("#image", RequestProviderUtils.isMultiOutputModel(ragData.getQuery()) ? this.template4image : "");
        query = query.replace("#skill_usage", usage);
        if (log.isWarnEnabled() && !TemplateChecker.check(query)) {
            log.warn("The query template contains unexpected characters, please check: {}", query);
        }
        return query;
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

        @Value("${skills.schema.template.creator:classpath:config/skills/creator.md}")
        protected String template4creator;

        @Value("${skills.schema.template.browser:classpath:config/skills/browser.md}")
        protected String template4browser;

        @Value("${skills.schema.template.miniapp:classpath:config/skills/miniapp.md}")
        protected String template4miniapp;

        @Value("${skills.schema.template.remote:classpath:config/skills/remote.md}")
        protected String template4remote;

        @Value("${skills.schema.template.feishu:classpath:config/skills/feishu.md}")
        protected String template4feishu;

        @Value("${skills.schema.template.usage:classpath:config/skills/usage.md}")
        protected String template4usage;

        @Value("${skills.schema.template.image:classpath:config/skills/image.md}")
        protected String template4image;

        @Value("${skills.schema.template.email:classpath:config/skills/email.md}")
        protected String template4email;

        @Value("${skills.schema.template.html:classpath:config/skills/html.md}")
        protected String template4html;

        @Value("${skills.schema.template.init:classpath:config/skills/main.md}")
        protected String template4init;

        @Value("${skills.schema.deepright:__internal_deepright}")
        protected String skillDeepRight;

        @Value("${skills.schema.miniapp:__internal_miniapp}")
        protected String skillMiniApp;

        @Value("${skills.schema.creator:__internal_creator}")
        protected String skillCreator;

        @Value("${skills.schema.feishu:__internal_feishu}")
        protected String skillFeishu;

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
