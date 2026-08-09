package ai.deepright.skills;

import ai.deepright.cli.CliPubData;
import ai.deepright.cli.CliPubSub;
import ai.deepright.cli.CliSubFetcher;
import ai.deepright.cli.CliSubOps;
import ai.deepright.feature.FeatureField;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.feature.FeatureUtils;
import ai.deepright.router.RouterDevice;
import ai.deepright.utils.TemplateChecker;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.AllowedConfig;
import ai.open.right.workflow.flow.file.impl.SysStore;
import ai.open.right.workflow.skill.SkillMetadata;
import ai.open.right.workflow.skill.Skills;
import ai.open.right.workflow.skill.SkillsFetcher;
import ai.open.right.workflow.skill.impl.FileSystemFetcher;
import com.google.common.collect.ImmutableMap;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
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
import java.io.File;
import java.nio.charset.StandardCharsets;
import java.nio.file.Paths;
import java.util.List;
import java.util.Map;

@Getter
@Setter
@Slf4j
public class MixedSkillFetcher extends FileSystemFetcher {

    protected CliSubFetcher cliSubFetcher;

    protected SysStore sysStore;

    protected String template4mcpJson;

    protected String template4mcpDef;

    protected String template4load;

    protected Integer oversize;

    protected String skills;

    protected String cli;

    @PostConstruct
    public void init() throws Exception {
        super.init();
        this.template4mcpJson = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4mcpJson).openStream()), StandardCharsets.UTF_8);
        this.template4mcpDef = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4mcpDef).openStream()), StandardCharsets.UTF_8);
        this.template4load = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4load).openStream()), StandardCharsets.UTF_8);
        WorkflowException.checkCondition(StringUtils.isEmpty(this.template4mcpJson), "The template mcp json must not be empty");
        WorkflowException.checkCondition(StringUtils.isEmpty(this.template4mcpDef), "The template mcp def must not be empty");
        WorkflowException.checkCondition(StringUtils.isEmpty(this.template4load), "The template load must not be empty");
    }

    @Override
    // path=SKILL.md
    public String fetchResource(WorkflowTask workTask, String name, String path) throws Exception {
        SkillMetadata skill = this.fetchSkills(workTask).getSkills().get(name);
        // location固定为技能文件本身的绝对路径，而不是目录路径
        // /xxx/skills/skill-creator/SKILL.md
        WorkflowException.checkCondition(skill == null, "The skill can not be empty: " + name);
        String location = MapUtils.getString(skill.getInternal(), "location");
        if (StringUtils.isEmpty(location)) {
            // 服务端Skills
            return super.fetchResource(workTask, name, path);
        }
        String resource = this.normalizePath(name, StringUtils.defaultIfEmpty(path, FileSystemFetcher.FILE));
        String fullPath = Paths.get(this.buildLocation(workTask, location)).getParent().resolve(resource).toString();
        if (StringUtils.equalsIgnoreCase(resource, FileSystemFetcher.FILE)) {
            String escaped = FeatureUtils.escapePath(workTask, fullPath);
            CliPubData pubData = this.cliSubFetcher.fetch(workTask, new RouterDevice(workTask), CliSubOps.builder()
                    .app(List.of("cat"))
                    .r(List.of(escaped))
                    .exempted(true)
                    .build(), escaped, "");
            WorkflowException.checkCondition(!(pubData.isOk()), pubData.getCmd());
            return this.replaceContent(workTask, name, path, this.removeHeader(pubData.getCmd()));
        } else {
            return this.buildRelated(workTask, fullPath, this.isBinary(workTask, name, path), name, path);
        }
    }

    @Override
    public Skills fetchSkills(WorkflowTask workTask, AllowedConfig allowedConfig) throws Exception {
        return this.fetchClient(workTask, super.fetchSkills(workTask, allowedConfig).copy());
    }

    @Override
    protected String buildRelated(WorkflowTask workTask, String location, Boolean binary, String name, String path) throws Exception {
        SkillMetadata skill = this.fetchSkills(workTask).getSkills().get(name);
        // location固定为技能文件本身的绝对路径，而不是目录路径
        WorkflowException.checkCondition(skill == null, "The skill can not be empty: " + name);
        String client = MapUtils.getString(skill.getInternal(), "location");
        if (!StringUtils.isEmpty(client)) {
            // 客户端资源
            String escaped = FeatureUtils.escapePath(workTask, location);
            CliPubData pubData = this.cliSubFetcher.fetch(workTask, new RouterDevice(workTask), CliSubOps.builder()
                    .app(binary ? List.of("mkdir", "curl", "base64", "gunzip") : List.of("mkdir", "cat"))
                    .exempted(true)
                    .build(), escaped, "");
            WorkflowException.checkCondition(!(pubData.isOk()), pubData.getCmd());
            String prefix = this.buildPrefix(workTask, location, binary, name, path);
            String suffix = this.buildSuffix(workTask, location, binary, name, path);
            return !binary ? prefix + this.replaceContent(workTask, name, path, this.removeHeader(pubData.getCmd())) + suffix : pubData.getCmd();
        } else {
            // 服务端资源，反向写入CLI
            String cliPath = this.buildPath(workTask, name, path);
            String content = this.buildContent(workTask, location, binary, name, path);
            // 创建目录并推送Skills
            this.cliSubFetcher.command(workTask, new RouterDevice(workTask), CliSubOps.builder()
                    .app(binary ? List.of("mkdir", "curl", "base64", "gunzip") : List.of("mkdir", "cat"))
                    // 精确匹配
                    .exempted(true).build(), CliPubSub.buildPushCmd(workTask, this.sysStore, binary, this.oversize, content, cliPath), "").valid();
            String resource = this.template4load.replace("#path", cliPath);
            resource = resource.replace("#tools_skill", this.skills);
            resource = resource.replace("#tools_cli", this.cli);
            if (log.isWarnEnabled() && !TemplateChecker.check(resource)) {
                log.warn("The template contains unexpected characters; please check: {}", resource);
            }
            return resource;
        }
    }

    protected String buildLocation(WorkflowTask workTask, String location) throws Exception {
        return !Paths.get(location).isAbsolute() ? new StringBuffer(FeatureUtils.buildWorkspace(workTask)).append(File.separator).append(location).toString() : location;
    }

    // 构建CLI目录
    protected String buildPath(WorkflowTask workTask, String name, String path) throws Exception {
        // @See CliRag
        String workspace = FeatureUtils.buildWorkspace(workTask);
        WorkflowException.checkCondition(StringUtils.isEmpty(workspace), "The workspace can not be empty");
        return this.buildSkills(workTask) + File.separator + name + File.separator + path;
    }

    protected Skills fetchClient(WorkflowTask workTask, Skills skills) throws Exception {
        List<Map<String, Object>> client = List.class.cast(MapUtils.getObject(workTask.getMetadata(), FeatureField.KEY_SKILLS));
        if (!CollectionUtils.isEmpty(client)) {
            for (Map<String, Object> each : client) {
                SkillMetadata skill = SkillMetadata.builder()
                        .internal(ImmutableMap.of("location", each.get("location")))
                        .description(MapUtils.getString(each, "description"))
                        .name(MapUtils.getString(each, "name"))
                        .build();
                skills.getSkills().put(MapUtils.getString(each, "name"), skill);
            }
        }
        return skills;
    }

    @Override
    protected String replaceContent(WorkflowTask workTask, String name, String path, String data) throws Exception {
        data = this.updatePluginsDir(workTask, name, path, data);
        data = this.updateWorkspace(workTask, name, path, data);
        data = this.updateTerminal(workTask, name, path, data);
        data = this.updateProvider(workTask, name, path, data);
        data = this.updateDevice(workTask, name, path, data);
        data = this.updateConfig(workTask, name, path, data);
        data = this.updateOrigin(workTask, name, path, data);
        data = this.updateAgent(workTask, name, path, data);
        data = this.updateChat(workTask, name, path, data);
        data = this.updateSoul(workTask, name, path, data);
        data = this.updateUser(workTask, name, path, data);
        data = this.updateSys(workTask, name, path, data);
        data = this.updateApp(workTask, name, path, data);
        data = this.updateDir(workTask, name, path, data);
        data = this.updateGit(workTask, name, path, data);
        data = this.updateMcp(workTask, name, path, data);
        return data;
    }

    protected String updatePluginsDir(WorkflowTask workTask, String name, String path, String data) throws Exception {
        return data.replace("#plugins_dir", FeatureUtils.buildPluginsDir(workTask));
    }

    protected String updateWorkspace(WorkflowTask workTask, String name, String path, String data) throws Exception {
        return data.replace("#workspace", FeatureUtils.buildWorkspace(workTask));
    }

    protected String updateTerminal(WorkflowTask workTask, String name, String path, String data) throws Exception {
        return data.replace("#terminal", FeatureUtils.buildTerminal(workTask));
    }

    protected String updateProvider(WorkflowTask workTask, String name, String path, String data) throws Exception {
        return data.replace("#provider", FeatureUtils.buildTargetProvider(workTask));
    }

    protected String updateConfig(WorkflowTask workTask, String name, String path, String data) throws Exception {
        return data.replace("#config", FeatureUtils.buildConfig(workTask));
    }

    protected String updateDevice(WorkflowTask workTask, String name, String path, String data) throws Exception {
        return data.replace("#device", workTask.getDevice());
    }

    protected String updateOrigin(WorkflowTask workTask, String name, String path, String data) throws Exception {
        return data.replace("#origin", FeatureUtils.buildOrigin(workTask));
    }

    protected String updateAgent(WorkflowTask workTask, String name, String path, String data) throws Exception {
        return data.replace("#agentId", FeatureUtils.buildAgentId(workTask));
    }

    protected String updateChat(WorkflowTask workTask, String name, String path, String data) throws Exception {
        return data.replace("#chat", workTask.getChat());
    }

    // SOUL.md
    protected String updateSoul(WorkflowTask workTask, String name, String path, String data) throws Exception {
        return data.replace("#soul", FeatureUtils.buildSoul(workTask));
    }

    // USER.md
    protected String updateUser(WorkflowTask workTask, String name, String path, String data) throws Exception {
        return data.replace("#user", FeatureUtils.buildUser(workTask));
    }

    protected String updateSys(WorkflowTask workTask, String name, String path, String data) throws Exception {
        return data.replace("#sys", FeatureUtils.buildSys(workTask));
    }

    protected String updateApp(WorkflowTask workTask, String name, String path, String data) throws Exception {
        return data.replace("#app", FeatureUtils.buildApp(workTask));
    }

    protected String updateDir(WorkflowTask workTask, String name, String path, String data) throws Exception {
        return data.replace("#dir", this.buildDir(workTask));
    }

    protected String updateGit(WorkflowTask workTask, String name, String path, String data) throws Exception {
        return data.replace("#git", StringUtils.defaultIfEmpty(FeatureUtils.buildGit(workTask), ""));
    }

    protected String updateMcp(WorkflowTask workTask, String name, String path, String data) throws Exception {
        String mcp = FeatureUtils.buildMcp(workTask);
        return data.replace("#mcp", !StringUtils.isEmpty(mcp) ? this.template4mcpJson.replace("#mcp", FeatureUtils.escapeShell(workTask, mcp)) : this.template4mcpDef);
    }

    protected String buildDir(WorkflowTask workTask) throws Exception {
        // @see CliRag
        String dir = FeatureUtils.buildPluginsDir(workTask);
        if (FeatureFlag.isMacOs(workTask)) {
            WorkflowException.checkCondition(!(StringUtils.containsIgnoreCase(FeatureUtils.buildApp(workTask), ".app")), "The mac apps must be installed in the Applications folder.");
            return Paths.get(dir).getParent() + FeatureUtils.buildFileSeparator(workTask) + "Resources";
        } else {
            return dir;
        }
    }

    protected String buildSkills(WorkflowTask workTask) throws Exception {
        return FeatureUtils.buildWorkspace(workTask) + File.separator + "skills";
    }

    @Configuration
    @Setter
    @Getter
    public static class MixedInitConfig extends InitConfig {

        @Autowired
        protected CliSubFetcher cliSubFetcher;

        @Autowired
        protected SysStore sysStore;

        @Value("${skills.fetcher.template.mcp.json:classpath:config/skills/mcp_json.md}")
        protected String template4mcpJson;

        @Value("${skills.fetcher.template.mcp.def:classpath:config/skills/mcp_def.md}")
        protected String template4mcpDef;

        @Value("${skills.fetcher.template.load:classpath:config/skills/load.md}")
        protected String template4load;

        @Value("${cli.push.oversize:1048576}")
        protected Integer oversize;

        @Value("${skills.fetcher.skills:skills}")
        protected String skills;

        // 加载用的技能名
        @Value("${skills.fetcher.cli:cli}")
        protected String cli;

        @Override
        @Bean
        @ConditionalOnMissingBean(value = SkillsFetcher.class)
        public MixedSkillFetcher skillsFetcher() throws Exception {
            this.usage = this.usage.replace("{SKILL_NAME}", this.name);
            MixedSkillFetcher mixedGenFetcher = new MixedSkillFetcher();
            BeanUtils.copyProperties(this, mixedGenFetcher);
            log.info("MixedSkillFetcher inited: dir={}, usage={}, expire={}", mixedGenFetcher.getDir(), mixedGenFetcher.getUsage(), mixedGenFetcher.getExpire());
            return mixedGenFetcher;
        }
    }
}
