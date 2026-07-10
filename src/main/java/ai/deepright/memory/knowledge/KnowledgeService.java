package ai.deepright.memory.knowledge;

import ai.deepright.cli.CliPrinter;
import ai.deepright.cli.CliPubData;
import ai.deepright.cli.CliSubFetcher;
import ai.deepright.cli.CliSubOps;
import ai.deepright.feature.FeatureField;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.feature.FeatureUtils;
import ai.deepright.lang.XmlResourceLang;
import ai.deepright.llm.notifier.MultiSourceNotifier;
import ai.deepright.llm.provider.RequestModelSelect;
import ai.deepright.memory.MemoryRecall;
import ai.deepright.memory.MemoryService;
import ai.deepright.memory.git.GitPath;
import ai.deepright.router.RouterService;
import ai.deepright.utils.SecurityTruncater;
import ai.deepright.utils.TemplateChecker;
import ai.deepright.workflow.worktask.HeartbeatWorkTask;
import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.resouce.ResourceService;
import ai.open.right.workflow.flow.WorkflowTask;
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
import org.apache.commons.collections.MapUtils;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.nio.charset.StandardCharsets;
import java.nio.file.Paths;
import java.time.Instant;
import java.time.ZoneId;
import java.time.format.DateTimeFormatter;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.TimeUnit;

@Slf4j
@Getter
@Setter
public class KnowledgeService implements MemoryService {

    public static final String LANG_KEY_RECALL_MESSAGE = "knowledge.recall.message";

    public static final String LANG_KEY_INIT_MESSAGE = "knowledge.init.message";

    public static final String KEY_KNOWLEDGE_COMMIT = "knowledge_commit";

    public static final String KEY_KNOWLEDGE = "knowledge";

    public static final String DATE_FORMAT = "yyyy-MM-dd HH:mm:ss";

    public static final String NAME = "knowledge";

    protected NotifierService notifierService;

    protected ResourceService resourceService;

    protected RouterService routerService;

    protected CliSubFetcher cliSubFetcher;

    protected String template4recall;

    protected String template4commit;

    protected String template4prefix;

    protected String template4init;

    protected Integer truncate;

    protected GitPath gitPath;

    protected Integer timeout;

    protected Long interval;

    protected String index;

    protected Integer days;

    @PostConstruct
    public void init() throws Exception {
        this.template4prefix = IOUtils.toString(this.resourceService.url(this.template4prefix).openStream(), StandardCharsets.UTF_8);
        this.template4recall = IOUtils.toString(this.resourceService.url(this.template4recall).openStream(), StandardCharsets.UTF_8);
        this.template4commit = IOUtils.toString(this.resourceService.url(this.template4commit).openStream(), StandardCharsets.UTF_8);
        this.template4init = IOUtils.toString(this.resourceService.url(this.template4init).openStream(), StandardCharsets.UTF_8);
        // IOUtils/JsonUtils负责关闭资源
        // 覆盖（rewrite），不需要重入
        // 启动检测，必要资源
        WorkflowException.check(StringUtils.isEmpty(this.template4prefix), "The template prefix must not be empty", ProtocolCode.C400);
        WorkflowException.check(StringUtils.isEmpty(this.template4recall), "The template recall must not be empty", ProtocolCode.C400);
        WorkflowException.check(StringUtils.isEmpty(this.template4commit), "The template commit must not be empty", ProtocolCode.C400);
        WorkflowException.check(StringUtils.isEmpty(this.template4init), "The template init must not be empty", ProtocolCode.C400);
    }

    @Override
    public String init(WorkflowTask workTask) throws Exception {
        if (FeatureFlag.isKnowledgeCommit(workTask)) {
            return this.buildQuery(workTask);
        } else {
            return this.template4init.replace("#knowledge", this.buildPath(workTask));
        }
    }

    @Override
    public String recall(WorkflowTask workTask, MemoryRecall recall) throws Exception {
        String keywords = recall.buildKeywords();
        if (StringUtils.isEmpty(keywords)) {
            // 知识库召回必须有有效关键字
            return "";
        }
        this.notify(workTask, XmlResourceLang.get(KnowledgeService.LANG_KEY_RECALL_MESSAGE));
        // grep -Ern "|||" /index | cut -d: -f1 | uniq
        Knowledge knowledge = this.buildKnowledge(workTask);
        StringBuffer buffer = new StringBuffer("grep -Ern '").append(keywords).append("' ").append(FeatureUtils.escapeShell(workTask, this.buildPath(workTask, knowledge))).append(" | cut -d: -f1 | uniq");
        CliPubData pubData = this.cliSubFetcher.command(workTask, CliSubOps.builder()
                .app(List.of("grep", "cut", "uniq"))
                .r(List.of(knowledge.getPath()))
                .exempted(true)
                .build(), buffer.toString(), "");
        WorkflowException.check(!(pubData.isOk()), pubData.getCmd(), ProtocolCode.C400);
        if (!StringUtils.isEmpty(pubData.getCmd())) {
            String query = this.template4recall.replace("#init", this.template4init.replace("#knowledge", this.buildPath(workTask, knowledge))).replace("#recall", pubData.getCmd());
            if (log.isWarnEnabled() && !TemplateChecker.check(query)) {
                log.warn("The recall template contains unexpected characters, please check: {}", query);
            }
            return SecurityTruncater.truncate(query, this.truncate);
        } else {
            return "";
        }
    }

    @Override
    public void commit(WorkflowTask workTask) throws Exception {
        if (this.isCommit(workTask) && this.updateTime(workTask) && log.isInfoEnabled()) {
            log.info("The knowledge positive commit has been success, device={}, agent={}", workTask.getDevice(), FeatureUtils.buildAgentId(workTask));
            return;
        }
        if (this.allowedUpdate(workTask) && this.updateTime(workTask)) {
            // 客户端时间间隔
            // Closeable=false, 只依赖CLI
            WorkflowTask copyTask = new HeartbeatWorkTask(this.routerService, workTask, false);
            // 继承Metadata
            SyncConfig syncConfig = SyncConfig.builder()
                    .syncCallable(KnowledgeCallable.builder()
                            .cliSubFetcher(this.cliSubFetcher)
                            .workTask(copyTask)
                            .build())
                    .metadata(this.buildMetadata(workTask))
                    .reQuery(this.buildQuery(workTask))
                    .workflow(MultiSourceNotifier.MAIN)
                    .notifier(Notifier.ENDPOINT)
                    .timeout(this.timeout)
                    .workTask(copyTask)
                    .build();
            SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig);
        }
    }

    @Override
    public void refresh(WorkflowTask workTask, List<History> histories) throws Exception {

    }

    @Override
    public Boolean support(WorkflowTask workTask) throws Exception {
        return this.buildKnowledge(workTask) != null;
    }

    protected Map<String, Object> buildMetadata(WorkflowTask workTask) throws Exception {
        Map<String, Object> metadata = new HashMap<String, Object>(workTask.getMetadata());
        // 标记知识库清洗
        metadata.put(KnowledgeService.KEY_KNOWLEDGE_COMMIT, true);
        // 关闭持久化上下文、关闭团队
        metadata.put(FeatureField.KEY_ROUTER_DISABLE, true);
        metadata.put(FeatureField.KEY_DAEMON, true);
        metadata.put(FeatureField.KEY_SILENT, true);
        metadata.put("__containHistories", false);
        return RequestModelSelect.transfer(workTask, metadata);
    }

    protected Knowledge buildKnowledge(WorkflowTask workTask) throws Exception {
        return workTask.getMetadata(KnowledgeService.KEY_KNOWLEDGE, Knowledge.class);
    }

    // Knowledge Path / Agent
    protected String buildPath(WorkflowTask workTask, Knowledge knowledge) throws Exception {
        return knowledge.getPath();
    }

    protected String buildPath(WorkflowTask workTask) throws Exception {
        return this.buildPath(workTask, this.buildKnowledge(workTask));
    }

    protected String buildQuery(WorkflowTask workTask) throws Exception {
        Knowledge knowledge = this.buildKnowledge(workTask);
        String prefix = this.template4prefix;
        DateTimeFormatter formatter = DateTimeFormatter.ofPattern(KnowledgeService.DATE_FORMAT);
        ZoneId zoneId = this.buildZoneId(workTask);
        // lastUpdate使用客户端时间，如果不存在则使用请求创建时间-days
        // limitUpdate使用请求创建时间
        String query = prefix.replace("#lastUpdate", Instant.ofEpochMilli(knowledge.getLastUpdate() != null ? knowledge.getLastUpdate() : (workTask.getCreated() - TimeUnit.MILLISECONDS.convert(this.days, TimeUnit.DAYS))).atZone(zoneId).format(formatter));
        query = query.replace("#limitUpdate", Instant.ofEpochMilli(workTask.getCreated()).atZone(zoneId).format(formatter));
        query = query.replace("#knowledge", this.buildPath(workTask, knowledge));
        query = query.replace("#git", this.gitPath.buildGitPath(workTask));
        query = query.replace("#index", this.index);
        if (log.isWarnEnabled() && !TemplateChecker.check(query)) {
            log.warn("The query template contains unexpected characters, please check: {}", query);
        }
        return query + System.lineSeparator() + StringUtils.defaultIfEmpty(FeatureUtils.buildKnowledge(workTask), this.template4commit);
    }

    protected void notify(WorkflowTask workTask, String scene) throws Exception {
        if (!FeatureFlag.isSilent(workTask)) {
            this.notifierService.notify(Segment.build(workTask, Segment.SegmentConfig.builder()
                    .content(new StringBuffer(XmlResourceLang.get(KnowledgeService.LANG_KEY_INIT_MESSAGE)))
                    .metadata(CliPrinter.process(scene))
                    .notifier(Notifier.SOURCE)
                    .build()), workTask);
        }
    }

    protected Boolean allowedUpdate(WorkflowTask workTask, Knowledge knowledge) throws Exception {
        // 如果是主动更新，则关闭以免二次更新
        if (this.isCommit(workTask)) {
            return false;
        }
        // 不传knowledge不更新，knowledge存在时仅在未显式关闭且达到更新时间间隔后允许更新
        return !knowledge.getDisable() && knowledge.shouldUpdate(this.interval);
    }

    protected Boolean allowedUpdate(WorkflowTask workTask) throws Exception {
        Knowledge knowledge = this.buildKnowledge(workTask);
        return knowledge != null && this.allowedUpdate(workTask, knowledge);
    }

    protected Boolean updateTime(WorkflowTask workTask) throws Exception {
        String path = FeatureUtils.buildApp(workTask);
        CliPubData pubData = this.cliSubFetcher.command(workTask, CliSubOps.builder()
                .app(List.of(Paths.get(path).getFileName().toString()))
                .w(List.of(path))
                .exempted(true)
                .build(), new StringBuffer(FeatureUtils.escapeShell(workTask, path)).append(" knowledge update-time --timestamp ").append(System.currentTimeMillis()).append(" --agentId ").append(FeatureUtils.buildAgentId(workTask)).toString(), "");
        if (!pubData.isOk() && log.isWarnEnabled()) {
            log.warn("The knowledge timestamp update failed={}", pubData.getCmd());
        }
        return pubData.isOk();
    }

    protected ZoneId buildZoneId(WorkflowTask workTask) throws Exception {
        String timezone = FeatureUtils.buildTimezone(workTask);
        return StringUtils.isBlank(timezone) ? ZoneId.systemDefault() : ZoneId.of(timezone);
    }

    // 是否为主动更新
    protected Boolean isCommit(WorkflowTask workTask) throws Exception {
        return MapUtils.getBoolean(workTask.getMetadata(), KnowledgeService.KEY_KNOWLEDGE_COMMIT, false);
    }

    @Builder
    @Slf4j
    public static class KnowledgeCallable extends BaseCallable {

        protected final StringBuffer buffer = new StringBuffer();

        protected CliSubFetcher cliSubFetcher;

        protected WorkflowTask workTask;

        @Override
        public void call(Segment segment) throws Exception {
            this.buffer.append(segment.getContent());
            if (segment.isFinished() && log.isInfoEnabled()) {
                log.info("The knowledge passive commit has been success, device={}, agent={}", this.workTask.getDevice(), FeatureUtils.buildAgentId(this.workTask));
            }
        }
    }

    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected NotifierService notifierService;

        @Autowired
        protected ResourceService resourceService;

        @Autowired
        protected RouterService routerService;

        @Autowired
        protected CliSubFetcher cliSubFetcher;

        @Autowired
        protected GitPath gitPath;

        @Value("${memory.knowledge.template.commit:classpath:config/memory/knowledge_commit.md}")
        protected String template4commit;

        @Value("${memory.knowledge.template.recall:classpath:config/memory/knowledge_recall.md}")
        protected String template4recall;

        @Value("${memory.knowledge.template.prefix:classpath:config/memory/knowledge_prefix.md}")
        protected String template4prefix;

        @Value("${memory.knowledge.template.init:classpath:config/memory/knowledge_init.md}")
        protected String template4init;

        @Value("${memory.knowledge.truncate:20480}")
        protected Integer truncate;

        @Value("${memory.knowledge.timeout:600000}")
        protected Integer timeout;

        @Value("${memory.knowledge.interval:3600000}")
        protected Long interval;

        @Value("${memory.knowledge.index:index.md}")
        protected String index;

        @Value("${memory.knowledge.days:3}")
        protected Integer days;

        @Bean(KnowledgeService.NAME)
        @ConditionalOnMissingBean(name = KnowledgeService.NAME)
        public KnowledgeService knowledgeService() throws Exception {
            KnowledgeService knowledgeService = new KnowledgeService();
            BeanUtils.copyProperties(this, knowledgeService);
            log.info("KnowledgeServiceImpl inited");
            return knowledgeService;
        }
    }
}
