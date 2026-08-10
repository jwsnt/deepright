package ai.open.right.workflow.skill.impl;

import ai.open.right.WorkflowException;
import ai.open.right.release.ResourceReleaser;
import ai.open.right.resouce.PlaceholderResolver;
import ai.open.right.resouce.ResourceService;
import ai.open.right.utils.GzipUtils;
import ai.open.right.utils.SuffixUtils;
import ai.open.right.utils.YamlUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.AllowedConfig;
import ai.open.right.workflow.skill.SkillMetadata;
import ai.open.right.workflow.skill.Skills;
import ai.open.right.workflow.skill.SkillsFetcher;
import com.google.common.cache.Cache;
import com.google.common.cache.CacheBuilder;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.io.FileUtils;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.io.File;
import java.io.IOException;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.nio.file.*;
import java.nio.file.attribute.BasicFileAttributes;
import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.concurrent.Callable;
import java.util.concurrent.TimeUnit;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

@Slf4j
public class FileSystemFetcher implements SkillsFetcher {

    private static final Map<String, SkillMetadata> EMPTY = Collections.unmodifiableMap(new LinkedHashMap<String, SkillMetadata>());

    private static final Pattern YAML_BLOCK_PATTERN = Pattern.compile("(?s)^---\\s*(.*?)\\s*---");

    public static final String INTERNAL_SKILLS = "__SKILLS__";

    public static final String INTERNAL_USAGE = "__USAGE__";

    public static final String FILE = "SKILL.md";

    protected Cache<String, Object> internalSkillsCache;

    @Getter
    @Setter
    protected PlaceholderResolver placeholderResolver;

    @Getter
    @Setter
    protected ResourceReleaser resourceReleaser;

    @Getter
    @Setter
    protected ResourceService resourceService;

    @Getter
    @Setter
    protected Boolean cached;

    @Getter
    @Setter
    protected Integer expire;

    @Setter
    @Getter
    // 是否释放（解压）Jar内资源
    protected Boolean release;

    @Setter
    @Getter
    protected String prefix;

    @Getter
    @Setter
    // 技能本身的描述
    protected String usage;

    @Getter
    @Setter
    protected String name;

    @Getter
    @Setter
    protected String args;

    @Getter
    @Setter
    protected String dir;

    @PostConstruct
    public void init() throws Exception {
        this.internalSkillsCache = CacheBuilder.newBuilder().expireAfterWrite(this.expire, TimeUnit.MILLISECONDS).maximumSize(Integer.MAX_VALUE).build();
    }

    @Override
    public Skills fetchSkills(WorkflowTask workTask, AllowedConfig allowedConfig) throws Exception {
        try {
            SkillsCallable skillsCallable = new SkillsCallable(allowedConfig, this.buildVisitor(), this.buildUsage(), this.buildPath());
            return this.cached ? Skills.class.cast(this.internalSkillsCache.get(FileSystemFetcher.INTERNAL_SKILLS, skillsCallable)) : skillsCallable.call();
        } catch (Exception e) {
            WorkflowException.dolog(e);
            return Skills.builder().skills(FileSystemFetcher.EMPTY).usage(this.buildUsage()).build();
        }
    }

    @Override
    public Skills fetchSkills(WorkflowTask workTask) throws Exception {
        return this.fetchSkills(workTask, new AllowedConfig());
    }

    @Override
    public String fetchResource(WorkflowTask workTask, String name, String path) throws Exception {
        try {
            // 如果为空则加载SKILL.md
            String resourcePath = this.normalizePath(name, StringUtils.defaultIfEmpty(path, FileSystemFetcher.FILE));
            SkillMetadata skill = this.fetchSkills(workTask).getSkills().get(name);
            Assert.notNull(skill, "The skill path for " + name + " was not found in cache");
            String location = Paths.get(skill.getPath()).getParent().toString() + File.separator + resourcePath;
            return StringUtils.equalsIgnoreCase(resourcePath, FileSystemFetcher.FILE) ? this.buildContent(workTask, location, false, name, path) : this.buildRelated(workTask, location, this.isBinary(workTask, name, path), name, path);
        } catch (Exception e) {
            WorkflowException.dolog(e);
            return "The skill resource failed to load, cause: " + e.getMessage();
        }
    }

    protected String buildRelated(WorkflowTask workTask, String location, Boolean binary, String name, String path) throws Exception {
        return this.buildContent(workTask, location, binary, name, path);
    }

    protected String buildContent(WorkflowTask workTask, String location, Boolean binary, String name, String path) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("The skill resource file={}", location);
        }
        String resource = this.buildSkillFetchCallable(false, binary, location).call();
        Assert.hasText(resource, "The skill resource can not be empty=" + resource);
        if (log.isDebugEnabled()) {
            log.debug("The skill resource={}", resource);
        }
        String prefix = this.buildPrefix(workTask, location, binary, name, path);
        String suffix = this.buildSuffix(workTask, location, binary, name, path);
        return !binary ? (prefix + this.replaceContent(workTask, name, path, this.removeHeader(resource)) + suffix) : resource;
    }

    protected String buildPrefix(WorkflowTask workTask, String location, Boolean binary, String name, String path) throws Exception {
        return System.lineSeparator() + "The following is the beginning of resource `" + path + "` in skill `" + name + "`" + System.lineSeparator();
    }

    protected String buildSuffix(WorkflowTask workTask, String location, Boolean binary, String name, String path) throws Exception {
        return System.lineSeparator() + "The above is the full content of resource `" + path + "` in skill `" + name + "`" + System.lineSeparator();
    }

    protected String replaceContent(WorkflowTask workTask, String name, String path, String data) throws Exception {
        try {
            // 从WorkTask动态替换内容
            Map<String, Object> args = StringUtils.isEmpty(this.args) ? workTask.getMetadata() : workTask.getMetadata(this.args, Map.class);
            if (!MapUtils.isEmpty(args)) {
                // 不以#开头的key添加#前缀
                String[] k = args.keySet().stream()
                        .map(key -> StringUtils.startsWith(key, this.prefix) ? key : this.prefix + key)
                        .toArray(String[]::new);
                // 是String直接使用，不是String使用ToString，null使用""
                String[] v = args.values().stream().map(value -> value == null ? "" : (String.class.isAssignableFrom(value.getClass()) ? (String) value : value.toString())).toArray(String[]::new);
                String replace = StringUtils.replaceEach(data, k, v);
                if (log.isDebugEnabled()) {
                    log.debug("Skill replace={}", replace);
                }
                return replace;
            } else {
                return data;
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
            return data;
        }
    }

    protected Boolean isBinary(WorkflowTask workTask, String name, String path) throws Exception {
        // SuffixUtils.isBinary 期望的是扩展名（如 png、pdf），不是完整路径；无扩展名时按二进制处理
        String suffix = "";
        if (StringUtils.isNotEmpty(path)) {
            int lastDot = path.lastIndexOf('.');
            if (lastDot >= 0 && lastDot < path.length() - 1) {
                suffix = path.substring(lastDot + 1);
            }
        }
        return SuffixUtils.isBinary(suffix);
    }

    protected String normalizePath(String name, String path) throws Exception {
        // 转为当前操作系统相关，并去除起始的/
        path = Path.of(path).toString();
        path = StringUtils.startsWith(path, File.separator) ? path.substring(1) : path;
        // 如果以/技能名/开头或技能名/开头
        String target = name + File.separator;
        if (StringUtils.startsWithIgnoreCase(path, target) || StringUtils.startsWithIgnoreCase(path, File.separator + target)) {
            path = StringUtils.removeStartIgnoreCase(path, target);
        }
        // 如果以/skills/开头或者skills/开头
        target = this.name + File.separator;
        if (StringUtils.startsWithIgnoreCase(path, target) || StringUtils.startsWithIgnoreCase(path, File.separator + target)) {
            path = StringUtils.removeStartIgnoreCase(path, target);
        }
        // 等于技能名或路径残缺
        if (StringUtils.equalsIgnoreCase(path, name) || StringUtils.isEmpty(path) || StringUtils.equalsIgnoreCase(path, ".md")) {
            path = "SKILL.md";
        }
        return path;
    }

    protected String removeHeader(String content) throws Exception {
        Matcher matcher = FileSystemFetcher.YAML_BLOCK_PATTERN.matcher(content);
        if (matcher.find() && matcher.groupCount() >= 1) {
            return StringUtils.trim(content.substring(matcher.group(0).length()));
        }
        return content;
    }

    protected SkillFetchCallable buildSkillFetchCallable(Boolean tolerated, Boolean binary, String path) throws Exception {
        return new SkillFetchCallable(this.placeholderResolver, this.resourceService, tolerated, binary, path);
    }

    protected SkillVisitor buildVisitor() throws Exception {
        return new SkillVisitor(this.placeholderResolver);
    }

    protected String buildUsage() throws Exception {
        Assert.hasText(this.usage, "The usage can not be empty");
        SkillFetchCallable skillFetchCallable = this.buildSkillFetchCallable(true, false, this.usage);
        String usage = this.cached ? String.class.cast(this.internalSkillsCache.get(FileSystemFetcher.INTERNAL_USAGE, skillFetchCallable)) : skillFetchCallable.call();
        if (log.isDebugEnabled()) {
            log.debug("Skill usage={}", usage);
        }
        return usage;
    }

    protected String buildPath() throws Exception {
        if (StringUtils.isEmpty(this.dir) && log.isInfoEnabled()) {
            log.info("The skills directory cannot be empty. Please set the `skills.dir` parameter");
        }
        // this.resourceService.root().getClassLoader().getResource("skills")仅允许在开发时使用
        return !StringUtils.isEmpty(this.dir) ? this.dir : this.buildDef();
    }

    protected String buildDef() throws Exception {
        URL url = this.resourceService.root().getClassLoader().getResource("skills");
        Assert.notNull(url, "The default skills path can not be find");
        String path = null;
        if (this.release && StringUtils.startsWithIgnoreCase(url.getPath(), ResourceReleaser.KEY_NESTED)) {
            // 尝试Jar的资源定位
            String root = this.resourceReleaser.getRoot();
            path = Paths.get(root.substring(0, root.length() - ".jar".length()), "BOOT-INF", "classes", "skills").toString();
        } else {
            path = url.getPath();
        }
        if (log.isInfoEnabled()) {
            log.info("The skills def path={}", path);
        }
        return path;
    }

    public static class SkillVisitor extends SimpleFileVisitor<Path> {

        @Getter
        private final Map<String, SkillMetadata> skills = new LinkedHashMap<String, SkillMetadata>();

        protected final PlaceholderResolver placeholderResolver;

        public SkillVisitor(PlaceholderResolver placeholderResolver) {
            this.placeholderResolver = placeholderResolver;
        }

        @Override
        public FileVisitResult visitFile(Path file, BasicFileAttributes attrs) throws IOException {
            if (file.getFileName().toString().equals(FileSystemFetcher.FILE)) {
                this.process(file);
            }
            return FileVisitResult.CONTINUE;
        }

        protected void process(Path file) {
            try {
                String content = Files.readString(file, StandardCharsets.UTF_8);
                Matcher matcher = FileSystemFetcher.YAML_BLOCK_PATTERN.matcher(content);
                if (!matcher.find() || matcher.groupCount() < 1) {
                    return;
                }
                Map<String, Object> skill = this.yaml(this.placeholderResolver.replace(matcher.group(1)));
                if (MapUtils.isEmpty(skill)) {
                    return;
                }
                String path = this.path(file.toFile().getAbsolutePath());
                Map<String, Object> meta = this.metadata(skill);
                String comp = this.compatibility(skill);
                String desc = this.description(skill);
                String[] tools = this.tools(skill);
                String name = this.name(skill);
                Assert.hasText(desc, "Skill's description can not be empty: " + path);
                Assert.hasText(name, "Skill's name can not be empty: " + path);
                SkillMetadata skillMetadata = SkillMetadata.builder().compatibility(comp).allowedTools(tools).description(desc).metadata(meta).path(path).name(name).build();
                this.skills.put(name, skillMetadata);
            } catch (Exception e) {
                WorkflowException.dolog(e);
            }
        }

        protected Map<String, Object> metadata(Map<String, Object> skill) throws Exception {
            try {
                return MapUtils.getMap(skill, "metadata");
            } catch (Exception e) {
                // 如果解析失败
                if (log.isDebugEnabled()) {
                    log.debug(e.getMessage(), e);
                }
                return null;
            }
        }

        protected String compatibility(Map<String, Object> skill) throws Exception {
            return MapUtils.getString(skill, "compatibility");
        }

        protected String description(Map<String, Object> skill) throws Exception {
            return MapUtils.getString(skill, "description");
        }

        protected String[] tools(Map<String, Object> skill) throws Exception {
            try {
                return StringUtils.split(MapUtils.getString(skill, "allowed-tools"), " ");
            } catch (Exception e) {
                if (log.isDebugEnabled()) {
                    log.debug(e.getMessage(), e);
                }
                return null;
            }
        }

        protected Map<String, Object> yaml(String content) throws Exception {
            return YamlUtils.read(content);
        }

        protected String name(Map<String, Object> skill) throws Exception {
            return MapUtils.getString(skill, "name");
        }

        protected String path(String path) throws Exception {
            return path;
        }
    }

    public static class SkillFetchCallable implements Callable<String> {

        protected final PlaceholderResolver placeholderResolver;

        protected final ResourceService resourceService;

        protected final Boolean tolerated;

        protected final Boolean binary;

        protected final String path;

        public SkillFetchCallable(PlaceholderResolver placeholderResolver, ResourceService resourceService, Boolean tolerated, Boolean binary, String path) {
            this.placeholderResolver = placeholderResolver;
            this.resourceService = resourceService;
            this.tolerated = tolerated;
            this.binary = binary;
            this.path = path;
        }

        public String call() throws Exception {
            try {
                return this.placeholderResolver.replace(this.build());
            } catch (Exception e) {
                if (log.isDebugEnabled()) {
                    log.debug(e.getMessage(), e);
                }
                if (this.tolerated) {
                    return this.path;
                } else {
                    throw e;
                }
            }
        }

        protected String build() throws Exception {
            int index = StringUtils.indexOf(this.path, ":");
            // Windows盘符及文件系统
            if (index > 1) {
                URL url = this.resourceService.url(this.path);
                return this.binary ? GzipUtils.compressAsBase64(IOUtils.toByteArray(url)) : IOUtils.toString(url, StandardCharsets.UTF_8);
            } else {
                File file = new File(this.path);
                return this.binary ? GzipUtils.compressAsBase64(FileUtils.readFileToByteArray(file)) : FileUtils.readFileToString(file, StandardCharsets.UTF_8);
            }
        }
    }

    public static class SkillsCallable implements Callable<Skills> {

        protected final AllowedConfig allowedConfig;

        protected final SkillVisitor skillVisitor;

        protected final String usage;

        protected final String path;

        public SkillsCallable(AllowedConfig allowedConfig, SkillVisitor skillVisitor, String usage, String path) {
            this.allowedConfig = allowedConfig;
            this.skillVisitor = skillVisitor;
            this.usage = usage;
            this.path = path;
        }

        public Skills call() throws Exception {
            Files.walkFileTree(Paths.get(this.path), this.skillVisitor);
            Map<String, SkillMetadata> metadata = new LinkedHashMap<String, SkillMetadata>();
            for (String key : this.skillVisitor.getSkills().keySet()) {
                SkillMetadata val = this.skillVisitor.getSkills().get(key);
                if (this.allowedConfig == null || this.allowedConfig.allowed(val.getName())) {
                    metadata.put(key, val);
                } else if (log.isDebugEnabled()) {
                    log.debug("The skill={} is not allowed", val.getName());
                }
            }
            return Skills.builder()
                    .usage(this.usage)
                    .skills(metadata)
                    .build();
        }
    }

    @ConditionalOnProperty(name = "skills.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected PlaceholderResolver placeholderResolver;

        @Autowired
        protected ResourceReleaser resourceReleaser;

        @Autowired
        protected ResourceService resourceService;

        @Value("${resource.release:false}")
        // 是否释放（解压）Jar内资源
        protected Boolean release;

        @Value("${skills.cached:true}")
        protected Boolean cached;

        @Value("${skills.expire:180000}")
        protected Integer expire;

        @Value("${skills.prefix:#}")
        protected String prefix;

        @Value("${skills.name:skills}")
        protected String name;

        @Value("${skills.usage:The `SKILL` must include YAML frontmatter metadata followed by Markdown content; required `name` for skill name, required `description` for skill effects & applicable scenarios, optional compatibility for environment requirements (target products, system packages, network access, etc.), optional `metadata` for arbitrary key-value mappings, optional `allowed-tools` for a space-separated pre-approved tool list.The `skill-related` associated files/scripts in the same-level directory must be read with tool `[{SKILL_NAME}]`.}")
        protected String usage;

        @Value("${skills.args:}")
        protected String args;

        @Value("${skills.dir:}")
        protected String dir;

        @Bean
        @ConditionalOnMissingBean(value = SkillsFetcher.class)
        public SkillsFetcher skillsFetcher() throws Exception {
            this.usage = this.usage.replace("{SKILL_NAME}", this.name);
            FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
            BeanUtils.copyProperties(this, fileSystemFetcher);
            log.info("FileSystemFetcher inited: dir={}, usage={}, cached={}, expire={}", fileSystemFetcher.getDir(), fileSystemFetcher.getUsage(), fileSystemFetcher.getCached(), fileSystemFetcher.getExpire());
            return fileSystemFetcher;
        }
    }
}