package ai.deepright.task;

import ai.deepright.cli.*;
import ai.deepright.cli.function.CliSubFunction;
import ai.deepright.feature.FeatureField;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.feature.FeatureUtils;
import ai.deepright.lang.XmlResourceLang;
import ai.deepright.llm.RetryUtils;
import ai.deepright.llm.notifier.MultiSourceFlag;
import ai.deepright.llm.notifier.MultiSourceNotifier;
import ai.deepright.llm.provider.RequestModelSelect;
import ai.deepright.plan.PlanUtils;
import ai.deepright.router.RouterDevice;
import ai.deepright.router.RouterService;
import ai.deepright.utils.TemplateChecker;
import ai.deepright.workflow.worktask.HeartbeatWorkTask;
import ai.open.right.WorkflowException;
import ai.open.right.context.UserContext;
import ai.open.right.resouce.ResourceService;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.file.DefStore;
import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.function.impl.BaseFunction;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.impl.ShortcutNotifier;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.google.common.collect.ImmutableMap;
import jakarta.annotation.PostConstruct;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.beanutils.PropertyUtils;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.io.BufferedInputStream;
import java.nio.charset.StandardCharsets;
import java.nio.file.Paths;
import java.util.*;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.TimeUnit;

@Slf4j
@Getter
@Setter
// CLI生成指定任务
public class TaskFunction extends BaseFunction implements TaskResult {

    public static final String LANG_KEY_START = "task.start";

    public static final String LANG_KEY_CLOSE = "task.close";

    // DEBUG
    public static final String NAME = "fun_task";

    protected Map<String, Object> responseSchema;

    protected ResourceService resourceService;

    protected ExecutorService executorService;

    protected RouterService routerService;

    protected CliSubFetcher cliSubFetcher;

    protected CliTransfer cliTransfer;

    protected Boolean allowedLocalhost;

    protected String template4artifact;

    protected String template4schema;

    protected String template4answer;

    protected String template4query;

    protected String template4error;

    protected String template4async;

    protected DefStore defStore;

    protected Integer oversize;

    protected Integer interval;

    protected Integer extract;

    protected Integer timeout;

    protected Integer expire;

    @PostConstruct
    public void init() throws Exception {
        this.responseSchema = Map.class.cast(PropertyUtils.getNestedProperty(JsonUtils.read(new BufferedInputStream(this.resourceService.url(this.template4schema).openStream()), Map.class), "extract.llm.additional.response_schema"));
        this.template4artifact = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4artifact).openStream()), StandardCharsets.UTF_8);
        this.template4answer = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4answer).openStream()), StandardCharsets.UTF_8);
        this.template4error = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4error).openStream()), StandardCharsets.UTF_8);
        this.template4query = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4query).openStream()), StandardCharsets.UTF_8);
        this.template4async = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4async).openStream()), StandardCharsets.UTF_8);
        Assert.hasText(this.template4artifact, "The template artifact must not be empty");
        Assert.hasText(this.template4answer, "The template answer must not be empty");
        Assert.notEmpty(this.responseSchema, "The response schema can not be empty");
        Assert.hasText(this.template4async, "The template asnyc must not be empty");
        Assert.hasText(this.template4query, "The template query must not be empty");
        Assert.hasText(this.template4error, "The template error must not be empty");
        this.timeout = (int) TimeUnit.MILLISECONDS.convert(this.timeout, TimeUnit.SECONDS);
    }

    @Override
    public Object call(FunctionContext functionContext) throws Exception {
        WorkflowTask workTask = functionContext.getWorkTask().printQuery();
        TaskData[] taskArray = this.buildTaskData(workTask);
        Assert.notEmpty(taskArray, "The assigned task cannot be empty, please check again.");
        RouterDevice sourceDevice = new RouterDevice(workTask).printRouter();
        Integer timeout = this.buildTimeout(workTask, taskArray);
        List<TaskSync> syncTasks = this.buildSyncTasks(workTask, sourceDevice, taskArray, timeout);
        return this.isAsync(workTask, syncTasks) ? this.execAsync(workTask, syncTasks, timeout) : this.execSync(workTask, syncTasks);
    }

    @Override
    public String buildAnswer(WorkflowTask workTask, TaskSync syncTask) throws Exception {
        TaskData taskData = this.buildExtract(workTask, syncTask);
        // 回传
        String artifacts = taskData.hasArtifacts() ? this.buildArtifacts(workTask, taskData, this.buildTransfer(workTask, syncTask.getTargetDevice(), syncTask.getSourceDevice(), taskData, false)) : "";
        String answer = this.template4answer.replace("#target", syncTask.getTargetDevice().key());
        answer = answer.replace("#content", taskData.getContent());
        answer = answer.replace("#artifact", artifacts);
        if (log.isWarnEnabled() && !TemplateChecker.check(answer)) {
            log.warn("The answer template contains unexpected characters, please check: {}", answer);
        }
        this.finishAndClean(workTask, syncTask);
        return answer;
    }

    @Override
    public String buildError(WorkflowTask workTask, TaskSync syncTask) throws Exception {
        String error = this.template4error.replace("#target", syncTask.getTargetDevice().key());
        error = error.replace("#content", syncTask.getError());
        if (log.isWarnEnabled() && !TemplateChecker.check(error)) {
            log.warn("The error template contains unexpected characters, please check: {}", error);
        }
        this.finishAndClean(workTask, syncTask);
        return error;
    }

    protected String buildAsync(WorkflowTask workTask, String filename, String deadline) throws Exception {
        String async = this.template4async.replace("#filename", filename);
        async = async.replace("#deadline", deadline);
        if (log.isWarnEnabled() && !TemplateChecker.check(async)) {
            log.warn("The error template contains unexpected characters, please check: {}", async);
        }
        return async;
    }

    protected String execAsync(WorkflowTask workTask, List<TaskSync> syncTasks, Integer timeout) throws Exception {
        TaskExec exec = TaskExec.builder()
                .cliSubFetcher(this.cliSubFetcher)
                .defStore(this.defStore)
                .oversize(this.oversize)
                .taskSyncs(syncTasks)
                .workTask(workTask)
                .taskResult(this)
                .build();
        this.executorService.execute(exec = exec.init(Math.min(timeout, this.timeout)));
        return this.buildAsync(workTask, exec.getFilename(), exec.getDeadline());
    }

    protected String execSync(WorkflowTask workTask, List<TaskSync> syncTasks) throws Exception {
        StringBuffer answer = new StringBuffer();
        for (TaskSync eachTask : syncTasks) {
            if (StringUtils.isEmpty(eachTask.getError())) {
                answer.append(this.buildAnswer(workTask, eachTask));
            } else {
                answer.append(this.buildError(workTask, eachTask));
            }
        }
        return answer.toString();
    }

    // 是否为异步任务（任一）
    protected Boolean isAsync(WorkflowTask workTask, List<TaskSync> syncTasks) throws Exception {
        return syncTasks.stream().anyMatch(task -> task.getTaskData().isAsync());
    }

    @Override
    public void source(WorkflowTask workTask, String content) throws Exception {
        if (!FeatureFlag.isSilent(workTask) && !StringUtils.isEmpty(content)) {
            super.source(workTask, content);
        }
    }

    protected TaskData[] buildTaskData(WorkflowTask workTask) throws Exception {
        String query = StringUtils.trim(workTask.printQuery().getQuery());
        query = JsonUtils.like(query) ? query : JsonUtils.extract(query);
        Assert.isTrue(JsonUtils.like(query), "The response cannot be parsed as JSON due to unexpected formatting: " + workTask.getQuery());
        TaskData[] taskData = null;
        // 适配多模型结果
        if (JsonUtils.map(workTask.getQuery())) {
            try {
                // 多个嵌套
                taskData = JsonUtils.transfer(MapUtils.getObject(workTask.getObjectQuery(Map.class), "task"), TaskData[].class);
            } catch (Exception e) {
                if (log.isDebugEnabled()) {
                    log.debug(e.getMessage(), e);
                }
            }
            // 单个
            taskData = taskData != null ? taskData : new TaskData[]{workTask.getObjectQuery(TaskData.class)};
        } else {
            taskData = workTask.getObjectQuery(TaskData[].class);
        }
        Assert.notNull(taskData, "The task data cannot be empty: " + workTask.getQuery());
        return taskData;
    }

    protected String buildQuery(WorkflowTask workTask, RouterDevice sourceDevice, RouterDevice targetDevice, TaskData taskData, List<TaskTransfer> transferData) throws Exception {
        String artifacts = !CollectionUtils.isEmpty(taskData.getArtifacts()) ? this.buildArtifacts(workTask, taskData, transferData) : "";
        String query = this.template4query.replace("#source", RouterDevice.agent(workTask));
        query = query.replace("#content", taskData.getContent().replace(sourceDevice.getWorkspace(), targetDevice.getWorkspace()));
        query = query.replace("#artifact", artifacts);
        if (log.isWarnEnabled() && !TemplateChecker.check(query)) {
            log.warn("The query template contains unexpected characters; please check: {}", query);
        }
        return query;
    }

    protected List<TaskTransfer> buildTransfer(WorkflowTask workTask, RouterDevice sourceDevice, RouterDevice targetDevice, TaskData taskData, Boolean check) throws Exception {
        List<TaskTransfer> taskTransfer = null;
        if (taskData.hasArtifacts() && this.allowedTransfer(workTask, sourceDevice, targetDevice)) {
            taskTransfer = new ArrayList<TaskTransfer>(taskData.getArtifacts().size());
            for (TaskArtifact artifact : taskData.getArtifacts()) {
                try {
                    CliTransferData transferData = this.cliTransfer.transfer(workTask, sourceDevice, targetDevice, artifact.getPath(), artifact.getWhy());
                    taskTransfer.add(TaskTransfer.builder()
                            .cliTransferData(transferData)
                            .taskArtifact(artifact)
                            .build());
                } catch (Exception e) {
                    if (!check) {
                        if (log.isInfoEnabled()) {
                            log.info(e.getMessage(), e);
                        }
                    } else {
                        throw e;
                    }
                }
            }
        }
        return taskTransfer;
    }

    protected String unmaskQuery(WorkflowTask workTask, RouterDevice sourceDevice, TaskData taskData, RouterDevice targetDevice, List<TaskTransfer> taskTransfers) throws Exception {
        String query = this.buildQuery(workTask, sourceDevice, targetDevice, taskData, taskTransfers);
        return query.replace(targetDevice.maskWorkspace().getWorkspace(), targetDevice.resetWorkspace().getWorkspace());
    }

    protected List<TaskSync> buildSyncTasks(WorkflowTask workTask, RouterDevice sourceDevice, TaskData[] taskArray, Integer timeout) throws Exception {
        List<TaskSync> syncTask = new ArrayList<TaskSync>();
        for (TaskData taskData : taskArray) {
            RouterDevice targetDevice = null;
            try {
                targetDevice = this.buildTargetDevice(workTask, taskData.check()).printRouter();
                this.source(workTask, taskData.getWhy());
                this.notify(workTask, targetDevice, MultiSourceFlag.KEY_START, TaskFunction.LANG_KEY_START);
                List<TaskTransfer> taskTransfers = this.buildTransfer(workTask, sourceDevice, targetDevice, taskData, true);
                String query = this.unmaskQuery(workTask, sourceDevice, taskData, targetDevice, taskTransfers);
                Map<String, Object> metadata = this.buildMetadata(workTask, targetDevice);
                syncTask.add(TaskSync.builder().syncWorkflowTask(this.commit(workTask, targetDevice, metadata, MultiSourceNotifier.MAIN, timeout, query))
                        .sourceDevice(sourceDevice)
                        .targetDevice(targetDevice)
                        .taskData(taskData)
                        .build());
                if (log.isInfoEnabled()) {
                    log.info("The task submit request, router={}", targetDevice.key());
                }
            } catch (Exception e) {
                WorkflowException.dolog(e);
                // 异常会抛出给模型，需要可读
                syncTask.add(TaskSync.builder()
                        .targetDevice(targetDevice != null ? targetDevice : new RouterDevice(workTask, taskData.getDevice(), taskData.getAgent()))
                        .sourceDevice(sourceDevice)
                        .error(e.getMessage())
                        .taskData(taskData)
                        .build());
            }
        }
        return syncTask;
    }

    protected String buildArtifacts(WorkflowTask workTask, TaskData taskData, List<TaskTransfer> transferData) throws Exception {
        StringBuffer buffer = new StringBuffer();
        if (!CollectionUtils.isEmpty(transferData)) {
            // 文件转存
            for (TaskTransfer t : transferData) {
                buffer.append("|").append(Paths.get(t.getCliTransferData().getSource()).getFileName()).append("|").append(t.getCliTransferData().getTarget()).append("|").append(t.getTaskArtifact().getDesc()).append("|").append(System.lineSeparator());
            }
        } else {
            // 同机文件
            for (TaskArtifact t : taskData.getArtifacts()) {
                buffer.append("|").append(Paths.get(t.getPath()).getFileName()).append("|").append(t.getPath()).append("|").append(t.getDesc()).append("|").append(System.lineSeparator());
            }
        }
        String artifact = this.template4artifact.replace("#artifact", buffer.toString());
        if (log.isWarnEnabled() && !TemplateChecker.check(artifact)) {
            log.warn("The artifact template contains unexpected characters; please check: {}", artifact);
        }
        return artifact;
    }

    protected Boolean allowedTransfer(WorkflowTask workTask, RouterDevice sourceDevice, RouterDevice targetDevice) throws Exception {
        return this.allowedLocalhost || !StringUtils.equalsIgnoreCase(sourceDevice.getDevice(), targetDevice.getDevice());
    }

    protected Map<String, Object> buildMetadata(WorkflowTask workTask, RouterDevice routerDevice) throws Exception {
        Map<String, Object> metadata = this.buildModelAndToken(workTask, routerDevice, new HashMap<String, Object>(routerDevice.getMetadata()));
        // @See CliRag
        metadata.put(FeatureField.KEY_MEDIA, MapUtils.getMap(routerDevice.getMetadata(), FeatureField.KEY_MEDIA));
        metadata.put(FeatureField.KEY_USER, MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_USER));
        metadata.put(FeatureField.KEY_WORKSPACE, routerDevice.getWorkspace());
        metadata.put(FeatureField.KEY_SILENT, FeatureFlag.isSilent(workTask));
        metadata.put(FeatureField.KEY_TERMINAL, routerDevice.getTerminal());
        metadata.put(FeatureField.KEY_ROUTER_UPSTREAM, routerDevice.key());
        metadata.put(FeatureField.KEY_OUTPUT, this.responseSchema);
        metadata.put(FeatureField.KEY_SYS, routerDevice.getSys());
        metadata.put(FeatureField.KEY_ROUTER_DISABLE, true);
        metadata.put(FeatureField.KEY_DAEMON, true);
        metadata.put(FeatureField.KEY_TASK, true);
        return metadata;
    }

    protected Map<String, Object> buildModelAndToken(WorkflowTask workTask, RouterDevice routerDevice, Map<String, Object> metadata) throws Exception {
        Assert.hasText(routerDevice.getProvider(), "The router provider can not be empty");
        String app = FeatureUtils.buildApp(routerDevice.getMetadata());
        String path = FeatureUtils.escapePath(FeatureFlag.isWindows(routerDevice.getSys()), app + " token --provider " + routerDevice.getProvider());
        CliPubData pubData = this.cliSubFetcher.command(workTask, routerDevice, CliSubOps.builder()
                .app(List.of(Paths.get(app).getFileName().toString()))
                .r(List.of(path))
                .exempted(true)
                .build(), path, "");
        Assert.isTrue(pubData.isOk(), pubData.getCmd());
        Map<String, String> provider = MapUtils.getMap(JsonUtils.read(pubData.getCmd(), Map.class), routerDevice.getProvider());
        String multiOutput = MapUtils.getString(provider, RequestModelSelect.KEY_MODEL_MULTI_OUTPUT);
        String multiInput = MapUtils.getString(provider, RequestModelSelect.KEY_MODEL_MULTI_INPUT);
        String thinking = MapUtils.getString(provider, RequestModelSelect.KEY_MODEL_THINKING);
        String base = MapUtils.getString(provider, RequestModelSelect.KEY_MODEL_BASE);
        String fast = MapUtils.getString(provider, RequestModelSelect.KEY_MODEL_FAST);
        String token = MapUtils.getString(provider, "token");
        String url = MapUtils.getString(provider, "__url");
        Assert.hasText(token, "The router provider token can not be empty: " + routerDevice.getProvider());
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN, token);
        metadata.put(ProviderRequestService.KEY_PROVIDER, routerDevice.getProvider());
        if (!StringUtils.isEmpty(multiOutput)) {
            metadata.put(RequestModelSelect.KEY_MODEL_MULTI_OUTPUT, multiOutput);
        }
        if (!StringUtils.isEmpty(multiInput)) {
            metadata.put(RequestModelSelect.KEY_MODEL_MULTI_INPUT, multiInput);
        }
        if (!StringUtils.isEmpty(thinking)) {
            metadata.put(RequestModelSelect.KEY_MODEL_THINKING, thinking);
        }
        if (!StringUtils.isEmpty(base)) {
            metadata.put(RequestModelSelect.KEY_MODEL_BASE, base);
        }
        if (!StringUtils.isEmpty(fast)) {
            metadata.put(RequestModelSelect.KEY_MODEL_FAST, fast);
        }
        if (!StringUtils.isEmpty(url)) {
            metadata.put(RequestModelSelect.KEY_MODEL_URL, url);
        }
        return metadata;
    }

    protected RouterDevice buildTargetDevice(WorkflowTask workTask, TaskData taskData) throws Exception {
        RouterDevice targetDevice = this.routerService.fetch(workTask, taskData.getDevice(), taskData.getAgent());
        Assert.isTrue(targetDevice != null && !targetDevice.isExpired(this.expire), "The router " + taskData.getAgent() + " is offline, so the assigned task cannot be completed.");
        Assert.isTrue(!targetDevice.isSame(workTask), "The router " + taskData.getAgent() + " must not assign tasks to itself");
        return targetDevice;
    }

    protected Integer buildTimeout(WorkflowTask workTask, TaskData[] taskArray) throws Exception {
        // seconds to mills
        int timeout = Math.min(this.timeout, (int) TimeUnit.MILLISECONDS.convert(Arrays.stream(taskArray)
                .filter(task -> task != null && task.getTimeout() != null)
                .mapToInt(TaskData::getTimeout)
                .max()
                .orElse(0), TimeUnit.SECONDS));
        return timeout > 0 ? timeout : this.timeout;
    }

    protected TaskData buildExtract(WorkflowTask workTask, TaskSync syncTask) throws Exception {
        String answer = syncTask.getSyncWorkflowTask().get();
        if (log.isInfoEnabled()) {
            log.info("The task answer is={}", answer);
        }
        if (StringUtils.isEmpty(answer)) {
            TaskData taskData = new TaskData();
            taskData.setContent("The internal error occurred and there was no response, please try again.");
            return taskData;
        }
        answer = StringUtils.trim(answer);
        answer = JsonUtils.like(answer) ? answer : JsonUtils.extract(answer);
        if (!JsonUtils.like(answer)) {
            TaskData taskData = new TaskData();
            taskData.setContent("The internal error occurred and response cannot be parsed as JSON due to unexpected formatting: " + answer + ", please try again.");
            return taskData;
        }
        TaskData taskData = null;
        try {
            // Json解析
            taskData = JsonUtils.read(answer, TaskData.class);
            if (taskData != null && taskData.hasAnyBody()) {
                return taskData;
            }
        } catch (Exception e) {
            if (log.isDebugEnabled()) {
                log.debug(e.getMessage(), e);
            }
        }
        try {
            // 模型解析
            String extract = StringUtils.trim(this.commit(workTask, syncTask.getTargetDevice(), this.buildMetadata(workTask, syncTask.getTargetDevice()), "extract", this.extract, answer).get());
            extract = JsonUtils.like(extract) ? extract : JsonUtils.extract(extract);
            taskData = JsonUtils.read(extract, TaskData.class);
        } catch (Exception e) {
            if (log.isDebugEnabled()) {
                log.debug(e.getMessage(), e);
            }
            taskData = new TaskData();
            taskData.setContent(answer);
        }
        return taskData;
    }

    protected void finishAndClean(WorkflowTask workTask, TaskSync syncTask) throws Exception {
        this.notify(workTask, syncTask.getTargetDevice(), MultiSourceFlag.KEY_CLOSE, TaskFunction.LANG_KEY_CLOSE);
        // 结束前清理任务Plan
        String plan = PlanUtils.deletePlan(workTask, syncTask.getTargetDevice().key());
        if (!StringUtils.isEmpty(plan) && log.isInfoEnabled()) {
            log.info("The task clean the plan");
        }
    }

    protected void notify(WorkflowTask workTask, RouterDevice targetDevice, String scene, String lang) throws Exception {
        if (!FeatureFlag.isSilent(workTask)) {
            super.source(workTask, CliPrinter.process(TaskFunction.NAME, targetDevice.getAgent()), XmlResourceLang.get(lang).replace("#agent", targetDevice.getAgent()));
            super.source(workTask, ImmutableMap.of(scene, targetDevice.getAgent()), targetDevice.getAgent());
        }
    }

    protected SyncWorkflowTask commit(WorkflowTask workTask, RouterDevice targetDevice, Map<String, Object> metadata, String workflow, Integer timeout, String query) throws Exception {
        SyncConfig syncConfig = SyncConfig.builder()
                .workTask(new TaskWorkflow(workTask, this.routerService, this, targetDevice, this.interval, this.timeout, FeatureFlag.isCron(workTask)))
                .notifier(Notifier.ENDPOINT)
                .chat(workTask.getChat())
                .metadata(metadata)
                .workflow(workflow)
                .timeout(timeout)
                .reQuery(query)
                // 不继承Metadata
                .pure(true)
                .build();
        return SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig);
    }

    @Getter
    public static class TaskWorkflow extends HeartbeatWorkTask {

        public static final Set<String> IGNORE = new HashSet<String>();

        static {
            TaskWorkflow.IGNORE.add("task@extract");
            TaskWorkflow.IGNORE.add("base@close");
        }

        protected final ShortcutNotifier shortcutNotifier;

        protected final RouterDevice targetDevice;

        protected final UserContext userContext;

        protected final WorkflowTask workTask;

        protected final Integer interval;

        // 是否来自计划任务
        protected final Boolean cron;

        protected final Long ddl;

        public TaskWorkflow(WorkflowTask workTask, RouterService routerService, ShortcutNotifier notifier, RouterDevice targetDevice, Integer interval, Integer timeout, Boolean cron) throws Exception {
            // Closeable=true 依赖主线程和CLI
            super(routerService, workTask, true);
            this.userContext = (this.targetDevice = targetDevice).isLoop(workTask) ? workTask.getUserContext() : UserContext.copyWithDevice(workTask.getUserContext(), targetDevice.getDevice());
            this.ddl = System.currentTimeMillis() + timeout;
            this.shortcutNotifier = notifier;
            this.workTask = workTask;
            this.interval = interval;
            this.cron = cron;
        }

        public Boolean isAllowed(Segment segment) throws Exception {
            // 不为计划任务（计划任务会特定结构）且不为特定节点 且为本机时
            return !this.cron && !TaskWorkflow.IGNORE.contains(SplitUtils.join(segment)) && this.targetDevice.isLoop(this.workTask);
        }

        @Override
        public void writeSource(Segment segment) throws Exception {
            try {
                String content = segment.getContent();
                content = this.isAllowed(segment) && !StringUtils.isEmpty(content) ? "[" + this.targetDevice.getAgent() + "]: " + content : "";
                // 过滤CLI_SUB消息
                if (!StringUtils.isEmpty(content) && !StringUtils.equalsIgnoreCase(SplitUtils.join(segment), "cli@sub") && !StringUtils.equalsIgnoreCase(MapUtils.getString(segment.getMetadata(), MultiSourceFlag.PROCESS), CliSubFunction.NAME)) {
                    Map<String, Object> metadata = new HashMap<String, Object>(segment.getMetadata());
                    if (MapUtils.getInteger(segment.getMetadata(), RetryUtils.RETRY) == null) {
                        // 事件
                        metadata.putAll(CliPrinter.process(TaskFunction.NAME));
                    }
                    metadata.put(MultiSourceFlag.TARGET, this.targetDevice.getAgent());
                    this.shortcutNotifier.source(this.workTask, metadata, content);
                }
            } catch (Exception e) {
                // 不能抛出异常
                WorkflowException.dolog(e);
            } finally {
                // 标记消费
                segment.mark();
            }
            // 不传递super.source
        }

        @Override
        public void writeBack(Segment segment) throws Exception {
            super.writeBack(segment);
        }

        @Override
        public void checkClosed() throws Exception {
            super.checkClosed();
            // 超时检查
            Assert.isTrue(this.ddl > System.currentTimeMillis(), "The task was timeout: " + this.ddl);
        }

        @Override
        public UserContext getUserContext() {
            return this.userContext;
        }

        @Override
        public String getDevice() {
            return this.targetDevice.getDevice();
        }
    }


    @Builder
    @Getter
    public static class TaskTransfer {

        protected CliTransferData cliTransferData;

        protected TaskArtifact taskArtifact;
    }

    @Getter
    @Setter
    public static class TaskArtifact {

        protected String path;

        protected String desc;

        @JsonProperty("why_do_this")
        protected String why;
    }

    @Configuration
    @Getter
    @Setter
    public static class InitConfig {

        @Autowired
        protected ResourceService resourceService;

        @Autowired
        @Qualifier("executor")
        protected ExecutorService executorService;

        @Autowired
        protected RouterService routerService;

        @Autowired
        protected CliSubFetcher cliSubFetcher;

        @Autowired
        protected CliTransfer cliTransfer;

        // 本机是否传输文件还是直接访问
        @Value("${task.allowedLocalhost:false}")
        protected Boolean allowedLocalhost;

        @Value("${task.template.artifact:classpath:config/task/artifact.md}")
        protected String template4artifact;

        // 提示下游使用的转换Schema
        @Value("${task.template.schema:classpath:config/task.json}")
        protected String template4schema;

        @Value("${task.template.answer:classpath:config/task/answer.md}")
        protected String template4answer;

        @Value("${task.template.query:classpath:config/task/query.md}")
        protected String template4query;

        @Value("${task.template.error:classpath:config/task/error.md}")
        protected String template4error;

        @Value("${task.template.async:classpath:config/task/async.md}")
        protected String template4async;

        @Autowired
        protected DefStore defStore;

        @Value("${cli.push.oversize:1048576}")
        protected Integer oversize;

        // 通知间隔（毫秒）
        @Value("${task.template.interval:30000}")
        protected Integer interval;

        // 子任务超时（秒），单位与模型传递对齐
        @Value("${task.timeout:300}")
        protected Integer timeout;

        // 提取任务超时（毫秒）
        @Value("${task.extract:120000}")
        protected Integer extract;

        @Value("${router.expire:120000}")
        protected Integer expire;

        @Bean(TaskFunction.NAME)
        @ConditionalOnMissingBean(name = TaskFunction.NAME)
        public TaskFunction taskFunction() throws Exception {
            TaskFunction taskFunction = new TaskFunction();
            BeanUtils.copyProperties(this, taskFunction);
            log.info("TaskFunction inited");
            return taskFunction;
        }
    }
}
