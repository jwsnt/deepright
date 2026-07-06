package ai.deepright.workflow.assistant;

import ai.deepright.cli.CliPrinter;
import ai.deepright.cli.insert.CliRecall;
import ai.deepright.complex.ComplexityUtils;
import ai.deepright.feature.FeatureField;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.lang.XmlResourceLang;
import ai.deepright.llm.notifier.MultiSourceFlag;
import ai.deepright.skills.SkillsChecker;
import ai.deepright.utils.TemplateChecker;
import ai.deepright.workflow.worktask.ResetStateWorkTask;
import ai.open.right.WorkflowException;
import ai.open.right.resouce.ResourceService;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowQueue;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.assistant.DefaultAssistant;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
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
import org.springframework.util.Assert;

import java.io.BufferedInputStream;
import java.nio.charset.StandardCharsets;
import java.util.*;

@Slf4j
@Getter
@Setter
public class LogicVerifyAssistant extends DefaultAssistant {

    public static final String LANG_KEY_ASSISTANT_LOGIC = "assistant.logic.hint";

    public static final String WORKFLOW_NAME = "logic";

    protected Set<String> excludedProvider = new HashSet<String>();

    protected ResourceService resourceService;

    protected WorkflowQueue workflowQueue;

    protected CliRecall cliRecall;

    protected String template4verify;

    protected String template4query;

    protected Integer maxTimes;

    protected Integer timeout;

    protected Boolean enabled;

    @PostConstruct
    public void init() throws Exception {
        this.template4verify = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4verify).openStream()), StandardCharsets.UTF_8);
        this.template4query = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4query).openStream()), StandardCharsets.UTF_8);
        Assert.hasText(this.template4verify, "The template verify must not be empty");
        Assert.hasText(this.template4query, "The template query must not be empty");
    }

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        // 没开logic：直接放行
        // 命中skip条件：直接放行
        // 其余情况执行isGo()，通过才放行，不通过就走重写/重试逻辑
        if (!this.enabled || this.skipLogic(workflowConfig, workTask) || this.isGo(workflowConfig, workTask)) {
            this.chainOr2Endpoint(workflowConfig, workTask, workTask.getQuery());
        } else if (MapUtils.getInteger(workTask.getMetadata(), LogicVerifyAssistant.WORKFLOW_NAME) == null) {
            // 断言，不应该存在死分支
            throw new WorkflowException("The system entered an exception branch");
        }
    }

    protected void notify(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        if (!FeatureFlag.isSilent(workTask)) {
            // __RESET__有值时立即将当前SSE响应框中的内容收入思考气泡，重置当前SSE思考气泡的状态，重新开始计算和分片
            this.notify(workTask, CliPrinter.process(LogicVerifyAssistant.WORKFLOW_NAME, MultiSourceFlag.RESET, LogicVerifyAssistant.WORKFLOW_NAME), Notifier.SOURCE, XmlResourceLang.get(LogicVerifyAssistant.LANG_KEY_ASSISTANT_LOGIC));
        }
    }

    protected String buildVerify(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        StringBuffer userQuery = new StringBuffer(workTask.getOriginal()).append(System.lineSeparator());
        // 合并中途插入的Query
        List<History> histories = this.cliRecall.recall(workTask);
        if (!CollectionUtils.isEmpty(histories)) {
            for (History history : histories) {
                userQuery.append(history.getContent()).append(System.lineSeparator());
            }
        }
        String verify = this.template4verify.replace("#query", userQuery.toString());
        verify = verify.replace("#answer", workTask.getQuery());
        if (log.isWarnEnabled() && !TemplateChecker.check(verify)) {
            log.warn("The error template contains unexpected characters, please check: {}", verify);
        }
        return verify;
    }

    protected String buildQuery(WorkflowConfig workflowConfig, WorkflowTask workTask, String improve) throws Exception {
        String query = this.template4query.replace("#query", workTask.getOriginal());
        query = query.replace("#improve", improve);
        if (log.isWarnEnabled() && !TemplateChecker.check(query)) {
            log.warn("The error template contains unexpected characters, please check: {}", query);
        }
        return query;
    }

    protected Boolean skipLogic(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        // 客户端任务 或 Task 或  后台任务 或 没有主动开启思考 或 供应商不支持 或 超过最大次数 则跳过
        return FeatureFlag.isCron(workTask) || FeatureFlag.isTask(workTask) || FeatureFlag.isDaemon(workTask) || !ComplexityUtils.isThinking(workTask) || this.excludedProvider.contains(workTask.getMetadata(ProviderRequestService.KEY_PROVIDER, String.class)) || this.isMaxTime(workflowConfig, workTask);
    }

    protected Boolean isMaxTime(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        return MapUtils.getInteger(workTask.getMetadata(), LogicVerifyAssistant.WORKFLOW_NAME, 0) >= this.maxTimes;
    }

    protected Boolean isGo(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        try {
            String response = this.commit(workTask, "logic", this.buildVerify(workflowConfig, workTask));
            if (log.isDebugEnabled()) {
                log.debug("The logic response is={}", response);
            }
            Map<String, Object> result = this.buildResult(workTask, response);
            if (!MapUtils.getBoolean(result, "passed", true)) {
                // 直接更新Workflow
                String improve = MapUtils.getString(result, "how_to_improve");
                if (!StringUtils.isEmpty(improve)) {
                    this.notify(workflowConfig, workTask);
                    this.workflowQueue.put(this.reConfig(workflowConfig, workTask, improve));
                    return false;
                } else {
                    // 没有建议直接通过
                    return true;
                }
            } else {
                return true;
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
            return true;
        } finally {
            workTask.putMetadata(LogicVerifyAssistant.WORKFLOW_NAME, MapUtils.getInteger(workTask.getMetadata(), LogicVerifyAssistant.WORKFLOW_NAME, 0) + 1);
        }
    }

    protected Map<String, Object> buildResult(WorkflowTask workTask, String response) throws Exception {
        response = StringUtils.trim(response);
        response = JsonUtils.like(response) ? response : JsonUtils.extract(response);
        try {
            Map<String, Object> result = JsonUtils.read(response, Map.class);
            Assert.notNull(MapUtils.getBoolean(result, "passed"), "The passed must not be empty");
            return result;
        } catch (Exception e) {
            if (log.isDebugEnabled()) {
                log.debug(e.getMessage(), e);
            }
        }
        try {
            // 模型解析
            String extract = StringUtils.trim(this.commit(workTask, "extract", response));
            extract = JsonUtils.like(extract) ? extract : JsonUtils.extract(extract);
            Map<String, Object> result = JsonUtils.read(extract, Map.class);
            Assert.notNull(MapUtils.getBoolean(result, "passed"), "The passed must not be empty");
            return result;
        } catch (Exception inner) {
            if (log.isDebugEnabled()) {
                log.debug(inner.getMessage(), inner);
            }
        }
        // 无法解析，直接通过
        return ImmutableMap.of("passed", true);
    }

    protected String commit(WorkflowTask workTask, String workflow, String query) throws Exception {
        SyncConfig syncConfig = SyncConfig.builder()
                // 不能破坏现有输出结构
                .metadata(ImmutableMap.of(FeatureField.KEY_DAEMON, true, FeatureField.KEY_SILENT, true, FeatureField.KEY_LOGIC, true))
                .notifier(Notifier.ENDPOINT)
                .chat(workTask.getChat())
                .timeout(this.timeout)
                .workflow(workflow)
                .workTask(workTask)
                .reQuery(query)
                .biz("logic")
                .build();
        return SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get();
    }

    // 重置时间
    protected WorkflowTask reConfig(WorkflowConfig workflowConfig, WorkflowTask workTask, String query) throws Exception {
        ResetStateWorkTask workWrap = new ResetStateWorkTask(workTask, this.buildQuery(workflowConfig, workTask, query));
        // CliInsertRag.KEY_RECALL不需要保留，会从缓存召回
        workWrap.getUserContext().getMetadata().clear();
        // 不存储Query
        workWrap.getMetadata().put("__storeQuery", false);
        workWrap.setWorkflow("main");
        workWrap.setBiz("main");
        return workWrap;
    }

    @Configuration
    @Setter
    @Getter
    public static class DefaultInitConfig extends InitConfig {

        @Autowired
        protected ResourceService resourceService;

        @Autowired
        protected WorkflowQueue workflowQueue;

        @Autowired
        protected SkillsChecker skillsChecker;

        @Autowired
        protected CliRecall cliRecall;

        @Value("${logic.template.verify:classpath:config/logic/verify.md}")
        protected String template4verify;

        @Value("${logic.template.query:classpath:config/logic/query.md}")
        protected String template4query;

        @Value("${logic.maxTimes:3}")
        protected Integer maxTimes;

        @Value("${logic.timeout:300000}")
        protected Integer timeout;

        @Value("${logic.excluded:}")
        protected String excluded;

        @Value("${logic.enabled:true}")
        protected Boolean enabled;

        @Bean(LogicVerifyAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = LogicVerifyAssistant.WORKFLOW_NAME)
        public LogicVerifyAssistant logicVerifyAssistant() throws Exception {
            LogicVerifyAssistant logicVerifyAssistant = new LogicVerifyAssistant();
            BeanUtils.copyProperties(this, logicVerifyAssistant);
            this.excluded(logicVerifyAssistant);
            log.info("LogicVerifyAssistant inited");
            return logicVerifyAssistant;
        }

        protected void excluded(LogicVerifyAssistant logicVerifyAssistant) {
            if (!StringUtils.isEmpty(this.excluded)) {
                Arrays.stream(this.excluded.split(","))
                        .map(StringUtils::trim)
                        .filter(StringUtils::isNotEmpty)
                        .forEach(logicVerifyAssistant.getExcludedProvider()::add);
            }
        }
    }
}
