package ai.deepright.memory.git;

import ai.deepright.cli.CliPrinter;
import ai.deepright.cli.CliPubData;
import ai.deepright.cli.CliSubFetcher;
import ai.deepright.cli.CliSubOps;
import ai.deepright.complex.ComplexityMode;
import ai.deepright.complex.ComplexityUtils;
import ai.deepright.feature.FeatureField;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.feature.FeatureUtils;
import ai.deepright.lang.XmlResourceLang;
import ai.deepright.llm.provider.RequestModelSelect;
import ai.deepright.memory.MemoryRecall;
import ai.deepright.memory.MemoryService;
import ai.deepright.router.RouterService;
import ai.deepright.utils.SecurityTruncater;
import ai.deepright.utils.TemplateChecker;
import ai.deepright.workflow.worktask.HeartbeatWorkTask;
import ai.open.right.WorkflowException;
import ai.open.right.resouce.ResourceService;
import ai.open.right.utils.BytesUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.function.impl.BaseFunction;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import ai.open.right.workflow.sync.impl.BaseCallable;
import jakarta.annotation.PostConstruct;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.apache.commons.lang3.time.FastDateFormat;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;
import org.unix4j.Unix4j;
import org.unix4j.line.Line;
import org.unix4j.unix.Grep;

import java.nio.charset.StandardCharsets;
import java.util.*;

@Slf4j
@Getter
@Setter
public class GitMemoryService extends BaseFunction implements GitPath, MemoryService {

    public static final FastDateFormat DATE_FORMAT = FastDateFormat.getInstance("yyyy-MM-dd HH:mm:ss");

    public static final String LANG_KEY_RECALL_MESSAGE = "memory.recall.message";

    public static final String LANG_KEY_INIT_MESSAGE = "memory.init.message";

    public static final String LANG_KEY_IMPORTANT = "memory.markdown.important";

    public static final String LANG_KEY_DATETIME = "memory.markdown.datetime";

    public static final String LANG_KEY_DIGEST = "memory.markdown.digest";

    public static final String LANG_KEY_NEEDS = "memory.markdown.needs";

    public static final String LANG_KEY_INIT = "memory.init.git";

    public static final String NAME = "memory.git";

    public static final String KEY_GIT = "git";

    protected NotifierService notifierService;

    protected ResourceService resourceService;

    protected CliSubFetcher cliSubFetcher;

    protected RouterService routerService;

    protected String template4summaryStore;

    protected String template4summaryQuery;

    protected String template4memory;

    protected Double complexRate;

    protected Integer truncate;

    protected Integer context;

    protected Integer timeout;

    protected Integer recall;

    protected String memory;

    protected Integer limit;

    protected Integer fetch;

    @PostConstruct
    public void init() throws Exception {
        this.template4summaryStore = IOUtils.toString(this.resourceService.url(this.template4summaryStore).openStream(), StandardCharsets.UTF_8);
        this.template4summaryQuery = IOUtils.toString(this.resourceService.url(this.template4summaryQuery).openStream(), StandardCharsets.UTF_8);
        this.template4memory = IOUtils.toString(this.resourceService.url(this.template4memory).openStream(), StandardCharsets.UTF_8);
        // IOUtils/JsonUtils负责关闭资源
        // 覆盖（rewrite），不需要重入
        // 启动检测，必要资源
        Assert.hasText(this.template4summaryStore, "The template summary store must not be empty");
        Assert.hasText(this.template4summaryQuery, "The template summary query must not be empty");
        Assert.hasText(this.template4memory, "The template init must not be empty");
    }

    @Override
    public String recall(WorkflowTask workTask, MemoryRecall recall) throws Exception {
        recall.checkValid();
        this.notify(workTask, XmlResourceLang.get(GitMemoryService.LANG_KEY_RECALL_MESSAGE));
        // 如果没有指定After就使用LastTimeLine
        Long lastTimeline = History.buildLastTimeline(workTask.getHistories());
        // 如果没有Before则lastTimeline（最新的历史记录之前的提交）
        String b = recall.hasBefore() ? (" --before " + FeatureUtils.escapeShell(workTask, recall.getBefore()) + " ") : (lastTimeline != null ? " --before " + FeatureUtils.escapeShell(workTask, GitMemoryService.DATE_FORMAT.format(lastTimeline)) : "");
        String a = recall.hasAfter() ? (" --after " + FeatureUtils.escapeShell(workTask, recall.getAfter()) + " ") : "";
        String g = this.buildRecallGrep(workTask, recall);
        // git log -n 20 --grep='' --format="%h" | xargs -r git show -p --pretty=format:''  | grep '^+' | grep -v '^+++' | cut -c 2-
        String l = new StringBuffer(FeatureUtils.escapeShell(workTask, this.buildGitApp(workTask))).append(" log -n ").append(this.buildRecall(workTask)).append(b).append(a).append(g).append(" ").toString();
        String s = new StringBuffer(l).append("--format=\"%h\" | xargs -r git show -p --pretty=format:''  | grep '^+' | grep -v '^+++' | cut -c 2-").toString();
        String c = new StringBuffer("cd ").append(FeatureUtils.escapeShell(workTask, this.buildGitPath(workTask))).append(" && ").append(s).toString();
        CliPubData pubData = this.cliSubFetcher.command(workTask, CliSubOps.builder()
                .r(List.of(FeatureUtils.escapeShell(workTask, this.buildGitPath(workTask))))
                .app(List.of("git"))
                .exempted(true)
                .build(), c, "");
        if (log.isInfoEnabled()) {
            log.info("The recalled memory={}, device={}", StringUtils.length(pubData.getCmd()), workTask.getDevice());
        }
        Assert.isTrue(pubData.isOk(), pubData.getCmd());
        return SecurityTruncater.truncate(this.buildRecallCompress(workTask, recall, pubData.getCmd()), this.truncate);
    }

    @Override
    public String init(WorkflowTask workTask) throws Exception {
        if (!this.allowedRead(workTask)) {
            return "";
        }
        this.notify(workTask, XmlResourceLang.get(GitMemoryService.LANG_KEY_INIT_MESSAGE));
        CliPubData pubData = this.buildInitData(workTask);
        if (!pubData.isOk()) {
            if (StringUtils.containsIgnoreCase(pubData.getCmd(), XmlResourceLang.get(MemoryService.LANG_KEY_MEMORY_INIT_FILE))) {
                // 尝试初始化
                CliPubData pubInit = this.buildInitFile(workTask);
                if (pubInit.isOk()) {
                    pubData = this.buildInitData(workTask);
                }
            }
        }
        if (!pubData.isOk()) {
            if (!StringUtils.containsIgnoreCase(pubData.getCmd(), XmlResourceLang.get(GitMemoryService.LANG_KEY_INIT))) {
                log.error("The inited memory failed={}", pubData.getCmd());
            }
            return "";
        }
        // 过滤import，节约上下文
        if (!StringUtils.isEmpty(pubData.getCmd())) {
            String memory = this.template4memory.replace("#memory", GitMemoryUtils.buildMarkdown(workTask, pubData.getCmd()));
            memory = memory.replace("#important", XmlResourceLang.get(GitMemoryService.LANG_KEY_IMPORTANT));
            if (log.isWarnEnabled() && !TemplateChecker.check(memory)) {
                log.warn("The init template contains unexpected characters; please check: {}", memory);
            }
            return SecurityTruncater.truncate(memory, this.truncate);
        } else {
            return "";
        }
    }

    @Override
    public void commit(WorkflowTask workTask) throws Exception {
        // Closeable=false, 只依赖CLI
        WorkflowTask copyTask = new HeartbeatWorkTask(this.routerService, workTask, false);
        // 继承Metadata
        SyncConfig syncConfig = SyncConfig.builder()
                // 异步回写Git
                .syncCallable(MemoryCommitCallable.builder()
                        // Git路径
                        .path(FeatureUtils.escapeShell(workTask, this.buildGitPath(workTask)))
                        .app(FeatureUtils.escapeShell(workTask, this.buildGitApp(workTask)))
                        .template(this.template4summaryStore)
                        .cliSubFetcher(this.cliSubFetcher)
                        .workTask(copyTask)
                        .gitPath(this)
                        .build())
                .metadata(this.buildMetadata(workTask))
                .reQuery(this.buildQuery(workTask))
                .notifier(Notifier.ENDPOINT)
                .workflow("memory@summary")
                .timeout(this.timeout)
                .workTask(copyTask)
                .build();
        SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig);
    }

    @Override
    public void refresh(WorkflowTask workTask, List<History> histories) throws Exception {
    }

    @Override
    public Boolean support(WorkflowTask workTask) throws Exception {
        return !StringUtils.isEmpty(this.buildGitApp(workTask));
    }


    @Override
    public String buildGitApp(WorkflowTask workTask) throws Exception {
        return MapUtils.getString(workTask.getMetadata(), GitMemoryService.KEY_GIT);
    }

    @Override
    public String buildGitPath(WorkflowTask workTask) throws Exception {
        // 设备维度
        return FeatureUtils.buildWorkspace(workTask) + FeatureUtils.buildFileSeparator(workTask) + "data";
    }

    @Override
    public String buildGitData(WorkflowTask workTask) throws Exception {
        return this.memory;
    }

    @Override
    // 初始化Git文件
    public CliPubData buildInitFile(WorkflowTask workTask) throws Exception {
        // no such file or directory
        StringBuffer buffer = new StringBuffer();
        String file = FeatureUtils.escapeShell(workTask, this.buildGitPath(workTask));
        buffer.append("mkdir -p ").append(file).append(" && ").append("cd ").append(file).append(" && ").append(FeatureUtils.escapeShell(workTask, this.buildGitApp(workTask))).append(" init");
        return this.cliSubFetcher.command(workTask, CliSubOps.builder()
                .app(List.of("mkdir", "cd"))
                .r(List.of(file))
                .exempted(true)
                .build(), buffer.toString(), "");
    }

    // 全量遍历，上游Query保证不能过大
    protected String buildRecallCompress(WorkflowTask workTask, MemoryRecall recall, String original) throws Exception {
        // 不存在KeyWord 或 长度字节数小于指定数量
        if (!recall.hasKeyword() || StringUtils.isEmpty(original) || BytesUtils.utf8Bytes(original) < this.limit) {
            return original;
        }
        String[] lines = original.split(FeatureUtils.buildLineSeparator(workTask));
        List<Line> hits = Unix4j.fromStrings(lines)
                .grep(Grep.Options.n, recall.buildKeywords())
                .toLineList();
        StringBuilder buffer = new StringBuilder();
        Set<Integer> keep = new TreeSet<Integer>();
        for (Line hit : hits) {
            String content = hit.getContent();
            int colon = content.indexOf(':');
            if (colon > 0) {
                try {
                    int idx = Integer.parseInt(content.substring(0, colon)) - 1;
                    for (int i = Math.max(0, idx - this.context); i <= Math.min(lines.length - 1, idx + this.context); i++) {
                        keep.add(i);
                    }
                } catch (NumberFormatException ex) {
                    if (log.isDebugEnabled()) {
                        log.warn("The recall bad line={}", content);
                    }
                }
            }
        }
        if (!keep.isEmpty()) {
            // 直接追加
            for (Integer idx : keep) {
                buffer.append(lines[idx]).append(FeatureUtils.buildLineSeparator((workTask)));
            }
        }
        return buffer.toString();
    }

    // 全量存储，但只召回重要的，grep '\[important=1\]'
    protected CliPubData buildInitData(WorkflowTask workTask) throws Exception {
        // --before="2026-03-26 11:25:23"
        String file = " -- " + this.memory;
        // 获取最旧的历史记录时间（与Short Memory做互补）
        Long lastTimeline = History.buildFirstTimeline(workTask.getHistories());
        String path = FeatureUtils.escapeShell(workTask, this.buildGitPath(workTask));
        String after = lastTimeline != null ? " --before " + FeatureUtils.escapeShell(workTask, GitMemoryService.DATE_FORMAT.format(lastTimeline)) + " " : "";
        StringBuffer buffer = new StringBuffer("cd ").append(path).append(" && ");
        // 如果没有LastTimeline表示没有短期记忆，就不需要过滤important
        buffer.append(FeatureUtils.escapeShell(workTask, this.buildGitApp(workTask))).append(" log -n ").append(this.buildFetch(workTask)).append(after).append(" --pretty=format:'%s' ").append(file).append(lastTimeline != null ? " | grep '\\[important=1\\]'" : "");
        return this.cliSubFetcher.command(workTask, CliSubOps.builder()
                .app(List.of("git"))
                .r(List.of(path))
                .exempted(true)
                .build(), buffer.toString(), "");
    }

    // 构建Grep部分
    protected String buildRecallGrep(WorkflowTask workTask, MemoryRecall recall) throws Exception {
        StringBuilder grep = new StringBuilder();
        if (!CollectionUtils.isEmpty(recall.getKeywords())) {
            grep.append(" --fixed-strings");
            for (String each : recall.getKeywords()) {
                if (!StringUtils.isBlank(each)) {
                    grep.append(" --grep=").append(FeatureUtils.escapeShell(workTask, each));
                }
            }
        }
        return grep.toString();
    }

    protected Map<String, Object> buildMetadata(WorkflowTask workTask) throws Exception {
        Map<String, Object> metadata = new HashMap<String, Object>(workTask.getMetadata());
        metadata.put(FeatureField.KEY_SILENT, true);
        return RequestModelSelect.transfer(workTask, metadata);
    }

    protected Integer buildRecall(WorkflowTask workTask) throws Exception {
        Integer recall = MapUtils.getInteger(workTask.getMetadata(), "__memoryRecall", this.recall);
        return (int) (recall * (ComplexityMode.FAST_REPLY.is(ComplexityUtils.result(workTask)) ? 1 : this.complexRate));

    }

    protected Integer buildFetch(WorkflowTask workTask) throws Exception {
        Integer fetch = MapUtils.getInteger(workTask.getMetadata(), "__memoryFetch", this.fetch);
        return (int) (fetch * (ComplexityMode.FAST_REPLY.is(ComplexityUtils.result(workTask)) ? 1 : this.complexRate));
    }

    protected String buildQuery(WorkflowTask workTask) throws Exception {
        String query = this.template4summaryQuery.replace("#query", workTask.getOriginal());
        query = query.replace("#answer", workTask.getQuery());
        if (log.isWarnEnabled() && !TemplateChecker.check(query)) {
            log.warn("The query template contains unexpected characters; please check: {}", query);
        }
        return query;
    }

    protected void notify(WorkflowTask workTask, String scene) throws Exception {
        if (!FeatureFlag.isSilent(workTask)) {
            this.source(workTask, CliPrinter.process(GitMemoryService.NAME), scene);
        }
    }

    protected Boolean allowedRead(WorkflowTask workTask) throws Exception {
        // 前提：当前没有服务端History召回时 客户端带了LastResponse（非新会话）或 非简单问题
        return CollectionUtils.isEmpty(History.getReferenceHistory(workTask.getHistories(), History.REFERENCE_SERVER)) && (FeatureUtils.buildLastResponse(workTask) != null || !ComplexityMode.FAST_REPLY.equals(ComplexityUtils.result(workTask)));
    }

    @Builder
    public static class MemoryCommitCallable extends BaseCallable {

        protected final StringBuffer buffer = new StringBuffer();

        protected CliSubFetcher cliSubFetcher;

        protected WorkflowTask workTask;

        protected GitPath gitPath;

        protected String template;

        protected String path;

        protected String app;

        @Override
        public void call(Segment segment) throws Exception {
            this.buffer.append(segment.getContent());
            if (segment.isFinished()) {
                String why = null;
                String digest = null;
                if (JsonUtils.like(this.buffer.toString())) {
                    Map<String, Object> memory = JsonUtils.read(this.buffer.toString(), Map.class);
                    Assert.notEmpty(memory, "The git memory commit can not be empty");
                    Integer important = MapUtils.getInteger(memory, "important", 0);
                    digest = "[datetime=" + GitMemoryService.DATE_FORMAT.format(this.workTask.getCreated()) + "][important=" + important + "][" + MapUtils.getString(memory, "needs") + "]";
                    why = MapUtils.getString(memory, "why_do_this");
                    if (log.isInfoEnabled()) {
                        log.info("The long-term memory refreshed, important={}, why={}", important, why);
                    }
                } else if (log.isWarnEnabled()) {
                    log.warn("The long-term memory refresh failed, the result is not JSON.");
                    digest = "[datetime=" + GitMemoryService.DATE_FORMAT.format(this.workTask.getCreated()) + "][important=0][" + this.workTask.getOriginal() + "]";
                }
                // 使用>覆盖，只保留Git
                StringBuffer buffer = new StringBuffer().append("cd ").append(this.path).append(" && ");
                buffer.append("echo ").append(FeatureUtils.escapeShell(this.workTask, this.buildContent(why))).append(" > ").append(this.gitPath.buildGitData(this.workTask)).append(" && ");
                buffer.append(this.app).append(" add ").append(this.gitPath.buildGitData(this.workTask)).append(" && ").append(this.app).append(" commit -m ").append(FeatureUtils.escapeShell(this.workTask, digest));
                CliPubData pubData = this.buildCommit(buffer.toString());
                if (!pubData.isOk()) {
                    if (StringUtils.containsIgnoreCase(pubData.getCmd(), XmlResourceLang.get(MemoryService.LANG_KEY_MEMORY_INIT_FILE))) {
                        // 尝试初始化
                        CliPubData pubInit = this.gitPath.buildInitFile(this.workTask);
                        if (pubInit.isOk()) {
                            pubData = this.buildCommit(buffer.toString());
                        }
                    }
                }
                if (!pubData.isOk()) {
                    throw new WorkflowException("The long-term memory refresh failed=" + pubData.getCmd());
                }
            }
        }

        protected String buildContent(String why) throws Exception {
            String store = this.template.replace("#query", this.workTask.getOriginal());
            store = store.replace("#answer", this.workTask.getQuery());
            store = store.replace("#why", StringUtils.defaultIfEmpty(why, ""));
            if (log.isWarnEnabled() && !TemplateChecker.check(store)) {
                log.warn("The store template contains unexpected characters; please check: {}", store);
            }
            return store;
        }

        protected CliPubData buildCommit(String commit) throws Exception {
            return this.cliSubFetcher.command(this.workTask, CliSubOps.builder()
                    .app(List.of("cd", "git", "echo"))
                    .w(List.of(this.path))
                    .exempted(true)
                    .build(), commit, "");
        }
    }

    @Configuration
    @Getter
    @Setter
    public static class InitConfig {

        @Autowired
        protected NotifierService notifierService;

        @Autowired
        protected ResourceService resourceService;

        @Autowired
        protected RouterService routerService;

        @Autowired
        protected CliSubFetcher cliSubFetcher;

        @Value("${memory.git.template.store:classpath:config/memory/summary_store.md}")
        protected String template4summaryStore;

        @Value("${memory.git.template.query:classpath:config/memory/summary_query.md}")
        protected String template4summaryQuery;

        @Value("${memory.git.template.init:classpath:config/memory/memory.md}")
        protected String template4memory;

        @Value("${memory.git.complex.rate:5}")
        protected Double complexRate;

        @Value("${memory.git.truncate:20480}")
        protected Integer truncate;

        @Value("${memory.git.timeout:60000}")
        protected Integer timeout;

        @Value("${memory.git.context:10}")
        protected Integer context;

        @Value("${memory.git.recall:5}")
        protected Integer recall;

        @Value("${memory.git.memory:memory}")
        protected String memory;

        @Value("${memory.git.fetch:5}")
        protected Integer fetch;

        // 超过限制需要压缩/过滤
        @Value("${memory.git.limit:102400}")
        protected Integer limit;

        @Bean(GitMemoryService.NAME)
        @ConditionalOnMissingBean(name = GitMemoryService.NAME)
        public GitMemoryService gitMemoryService() throws Exception {
            GitMemoryService gitMemoryService = new GitMemoryService();
            BeanUtils.copyProperties(this, gitMemoryService);
            log.info("GitMemoryService inited");
            return gitMemoryService;
        }
    }
}
