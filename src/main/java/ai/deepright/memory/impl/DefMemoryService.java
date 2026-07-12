package ai.deepright.memory.impl;

import ai.deepright.complex.ComplexityMode;
import ai.deepright.complex.ComplexityUtils;
import ai.deepright.feature.FeatureField;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.llm.provider.RequestModelSelect;
import ai.deepright.memory.MemoryRecall;
import ai.deepright.memory.MemoryService;
import ai.deepright.router.RouterService;
import ai.deepright.utils.SecurityTruncater;
import ai.deepright.utils.TemplateChecker;
import ai.deepright.workflow.worktask.HeartbeatWorkTask;
import ai.open.right.WorkflowException;
import ai.open.right.config.RedisConfig;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.resouce.ResourceService;
import ai.open.right.utils.SpinExec;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.function.impl.BaseFunction;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.store.history.HistoryTruncate;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import ai.open.right.workflow.sync.impl.BaseCallable;
import jakarta.annotation.PostConstruct;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.core.RedisTemplate;

import java.io.BufferedInputStream;
import java.nio.charset.StandardCharsets;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.TimeUnit;

@Slf4j
@Getter
@Setter
public class DefMemoryService extends BaseFunction implements MemoryService {

    protected RedisTemplate<String, Object> redis4array;

    protected List<MemoryService> memoryService;

    protected ResourceService resourceService;

    protected RouterService routerService;

    protected String template4refreshSummary;

    protected String template4refreshSession;

    protected Integer timeout4character;

    protected Integer timeout4redis;

    protected Integer truncate;

    // USER.md和SOUL.md加总超过该值则需要压缩
    protected Integer oversize;

    protected Integer circle;

    protected Integer expire;

    @PostConstruct
    public void init() throws Exception {
        // IOUtils/JsonUtils负责关闭资源
        this.template4refreshSummary = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4refreshSummary).openStream()), StandardCharsets.UTF_8);
        this.template4refreshSession = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4refreshSession).openStream()), StandardCharsets.UTF_8);
        // 覆盖（rewrite），不需要重入
        // 启动检测，必要资源
        WorkflowException.check(StringUtils.isEmpty(this.template4refreshSummary), "The template refresh summary must not be empty", ProtocolCode.C400);
        WorkflowException.check(StringUtils.isEmpty(this.template4refreshSession), "The template session summary must not be empty", ProtocolCode.C400);
    }


    @Override
    public Object call(FunctionContext functionContext) throws Exception {
        WorkflowTask workTask = functionContext.getWorkTask().printQuery();
        return this.recall(workTask, workTask.getObjectQuery(MemoryRecall.class));
    }

    @Override
    public String init(WorkflowTask workTask) throws Exception {
        if (this.allowedRead(workTask)) {
            MemoryService memoryService = this.fetchMemoryService(workTask);
            if (memoryService != null) {
                // 初始化长期记忆
                Object value = new MemoryInitExec(this.redis4array, memoryService, workTask, this.expire, this.timeout4redis, this.circle, this.getKey(workTask)).exec();
                if (value != null) {
                    if (log.isInfoEnabled()) {
                        log.info("The memory was inited");
                    }
                    return SecurityTruncater.truncate(String.class.cast(value), this.truncate);
                }
            }
        }
        return "";
    }

    @Override
    public String recall(WorkflowTask workTask, MemoryRecall recall) throws Exception {
        if (this.allowedRead(workTask)) {
            MemoryService memoryService = this.fetchMemoryService(workTask);
            if (memoryService != null) {
                return SecurityTruncater.truncate(memoryService.recall(workTask, recall), this.truncate);
            }
        }
        return "";
    }

    @Override
    public void commit(WorkflowTask workTask) throws Exception {
        if (this.allowedUpdate(workTask)) {
            this.updateCharacter(workTask, this.buildRefreshSession(workTask));
            MemoryService memoryService = this.fetchMemoryService(workTask);
            if (memoryService != null) {
                memoryService.commit(workTask);
            }
        } else if (log.isDebugEnabled()) {
            log.debug("The memory commit will be ignored for entropy");
        }
    }

    @Override
    public void refresh(WorkflowTask workTask, List<History> histories) throws Exception {
        if (this.allowedUpdate(workTask)) {
            String memory = History.buildMarkdown(histories, MemoryTruncate.builder()
                    .truncate(this.truncate)
                    .workTask(workTask)
                    .build());
            // 范围内则刷新（ComplexityUtils.score内置长文跳过）
            if (!ComplexityUtils.score(memory).is(ComplexityMode.FAST_REPLY)) {
                this.updateCharacter(workTask, this.buildRefreshSummary(workTask, memory));
                MemoryService memoryService = this.fetchMemoryService(workTask);
                if (memoryService != null) {
                    memoryService.refresh(workTask, histories);
                }
            } else if (log.isDebugEnabled()) {
                log.debug("The memory refresh will be ignored for entropy");
            }
        }
    }

    protected Boolean allowedUpdate(WorkflowTask workTask) throws Exception {
        // 不为后台任务（但Task除外）、不为归纳Skills、不为Profile整理
        // 知识库整理部分由KnowledgeService决策
        // Original为原始请求，Query为改写请求（等同Answer），不保存无价值记忆
        return (!FeatureFlag.isDaemon(workTask) || FeatureFlag.isTask(workTask)) && !FeatureFlag.isProfileCommit(workTask) && !FeatureFlag.isKnowledgeCommit(workTask) && !FeatureFlag.isSkillExtract(workTask) && ((!ComplexityUtils.score(workTask.getOriginal() + System.lineSeparator() + workTask.getQuery()).is(ComplexityMode.FAST_REPLY)));
    }

    protected Boolean allowedRead(WorkflowTask workTask) throws Exception {
        // 不为Profile（User.md/Soul.md）整理
        return !FeatureFlag.isProfileCommit(workTask);
    }

    @Override
    public Boolean support(WorkflowTask workTask) throws Exception {
        for (MemoryService memoryService : this.memoryService) {
            if (memoryService.support(workTask) && memoryService != this) {
                return true;
            }
        }
        return false;
    }

    protected void updateCharacter(WorkflowTask workTask, String query) throws Exception {
        SyncConfig syncConfig = SyncConfig.builder()
                // Closeable=false, 只依赖CLI
                .workTask(new HeartbeatWorkTask(this.routerService, workTask, false))
                .syncCallable(MemoryRefreshCallable.builder()
                        .build())
                .metadata(this.buildMetadata(workTask))
                .timeout(this.timeout4character)
                // 推送为Endpoint（上游）
                .notifier(Notifier.ENDPOINT)
                .workflow("memory@refresh")
                .reQuery(query)
                .build();
        SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig);
    }

    // Summary总结时
    protected String buildRefreshSummary(WorkflowTask workTask, String memory) throws Exception {
        String summary = this.template4refreshSummary.replace("#memory", memory);
        if (log.isWarnEnabled() && !TemplateChecker.check(summary)) {
            log.warn("The refresh summary template contains unexpected characters; please check: {}", summary);
        }
        return summary;
    }

    // 每次会话时
    protected String buildRefreshSession(WorkflowTask workTask) throws Exception {
        String summary = this.template4refreshSession.replace("#query", workTask.getOriginal());
        summary = summary.replace("#answer", workTask.getQuery());
        if (log.isWarnEnabled() && !TemplateChecker.check(summary)) {
            log.warn("The refresh session template contains unexpected characters; please check: {}", summary);
        }
        return summary;
    }

    protected Map<String, Object> buildMetadata(WorkflowTask workTask) throws Exception {
        Map<String, Object> metadata = new HashMap<String, Object>(workTask.getMetadata());
        // 开启静默模式和后台任务
        metadata.put(FeatureField.KEY_DAEMON, true);
        metadata.put(FeatureField.KEY_SILENT, true);
        return RequestModelSelect.transfer(workTask, metadata);
    }

    protected MemoryService fetchMemoryService(WorkflowTask workTask) throws Exception {
        MultiMemoryService multiMemoryService = new MultiMemoryService();
        for (MemoryService memoryService : this.memoryService) {
            if (memoryService.support(workTask) && memoryService != this) {
                if (log.isInfoEnabled()) {
                    log.info("The memory service={}", memoryService);
                }
                multiMemoryService.add(memoryService);
            }
        }
        return !multiMemoryService.isEmpty() ? multiMemoryService : null;
    }

    protected String getKey(WorkflowTask workTask) throws Exception {
        // 设备所有Agent共享长期记忆
        return RedisConfig.DOMAIN + DefMemoryService.class.getSimpleName() + workTask.getDevice();
    }

    public static class MemoryInitExec extends SpinExec {

        protected final RedisTemplate<String, Object> redis4array;

        protected final MemoryService memoryService;

        protected final WorkflowTask workTask;

        protected final Integer expired;

        protected final String key;

        public MemoryInitExec(RedisTemplate<String, Object> redis4array, MemoryService memoryService, WorkflowTask workTask, Integer expired, Integer timeout, Integer circle, String key) {
            super(timeout, circle);
            this.memoryService = memoryService;
            this.redis4array = redis4array;
            this.workTask = workTask;
            this.expired = expired;
            this.key = key;
        }

        @Override
        public Object doExec() throws Exception {
            try {
                Object data = this.redis4array.opsForValue().get(this.key);
                if (data == null) {
                    // 初始化
                    data = this.memoryService.init(this.workTask);
                    try {
                        // 被动更新
                        new MemorySetExec(this.redis4array, String.class.cast(data), this.expired, this.timeout, this.circle, this.key).doExec();
                    } catch (Exception e) {
                        WorkflowException.dolog(e);
                    }
                    return data;
                } else {
                    return new String(byte[].class.cast(data), StandardCharsets.UTF_8);
                }
            } catch (Exception e) {
                WorkflowException.dolog(e);
                return null;
            }
        }
    }

    public static class MemorySetExec extends SpinExec {

        protected final RedisTemplate<String, Object> redis4array;

        protected final Integer expired;

        protected final String content;

        protected final String key;

        public MemorySetExec(RedisTemplate<String, Object> redis4array, String content, Integer expired, Integer timeout, Integer circle, String key) {
            super(timeout, circle);
            this.redis4array = redis4array;
            this.content = content;
            this.expired = expired;
            this.key = key;
        }

        @Override
        public Object doExec() throws Exception {
            try {
                // 刷新长期记忆
                this.redis4array.opsForValue().set(this.key, this.content.getBytes(StandardCharsets.UTF_8), this.expired, TimeUnit.MILLISECONDS);
                return true;
            } catch (Exception e) {
                WorkflowException.dolog(e);
                return null;
            }
        }
    }

    @Builder
    public static class MemoryRefreshCallable extends BaseCallable {

        protected final StringBuffer buffer = new StringBuffer();

        @Override
        public void call(Segment segment) throws Exception {
            this.buffer.append(segment.getContent());
            if (segment.isFinished() && log.isDebugEnabled()) {
                log.debug("The memory refreshed device={}, content={}", segment.getDevice(), this.buffer);
            }
        }
    }

    @Builder
    @Getter
    public static class MemoryTruncate implements HistoryTruncate {

        protected WorkflowTask workTask;

        protected Integer truncate;

        @Override
        public String truncate(String histories) throws Exception {
            return SecurityTruncater.truncate(histories, this.truncate);
        }
    }

    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected RedisTemplate<String, Object> redis4array;

        @Autowired
        protected List<MemoryService> memoryService;

        @Autowired
        protected ResourceService resourceService;

        @Autowired
        protected RouterService routerService;

        @Value("${memory.template.refresh.summary:classpath:config/memory/refresh_summary.md}")
        protected String template4refreshSummary;

        @Value("${memory.template.refresh.summary:classpath:config/memory/refresh_session.md}")
        protected String template4refreshSession;

        @Value("${memory.timeout.character:5000}")
        protected Integer timeout4character;

        @Value("${memory.timeout.redis:5000}")
        protected Integer timeout4redis;

        @Value("${memory.truncate:128000}")
        protected Integer truncate;

        // USER.md和SOUL.md加总超过该值（字节）则需要压缩
        @Value("${memory.oversize:25600}")
        protected Integer oversize;

        @Value("${memory.circle:10}")
        protected Integer circle;

        @Value("${memory.expire:10}")
        protected Integer expire;

        @Bean(MemoryService.NAME)
        @ConditionalOnMissingBean(name = MemoryService.NAME)
        public DefMemoryService defMemoryService() throws Exception {
            DefMemoryService redisMemoryService = new DefMemoryService();
            BeanUtils.copyProperties(this, redisMemoryService);
            log.info("DefMemoryService inited");
            return redisMemoryService;
        }
    }
}
