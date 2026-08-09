package ai.deepright.plan.function;

import ai.deepright.cli.CliPrinter;
import ai.deepright.feature.FeatureField;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.lang.XmlResourceLang;
import ai.deepright.llm.provider.RequestModelSelect;
import ai.deepright.plan.PlanUtils;
import ai.deepright.router.RouterService;
import ai.deepright.utils.TemplateChecker;
import ai.deepright.workflow.worktask.HeartbeatWorkTask;
import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.resouce.ResourceService;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.function.impl.BaseFunction;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
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
import java.util.HashMap;
import java.util.Map;

@Slf4j
@Getter
@Setter
// 仅创建，更新覆盖使用PlanUpdateFunction
public class PlanCreateFunction extends BaseFunction {

    public static final String LANG_KEY_PLAN_CREATE = "plan.create";

    public static final String NAME = "fun_plan_create";

    public static final Integer CLOSE = 2;

    public static final Integer OPEN = 1;

    protected ResourceService resourceService;

    protected RouterService routerService;

    protected String template4directly;

    protected String template4answer;

    protected String template4query;

    protected Integer timeout;

    protected String update;

    @PostConstruct
    public void init() throws Exception {
        // IOUtils/JsonUtils负责关闭资源
        this.template4directly = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4directly).openStream()), StandardCharsets.UTF_8);
        this.template4answer = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4answer).openStream()), StandardCharsets.UTF_8);
        this.template4query = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4query).openStream()), StandardCharsets.UTF_8);
        // 覆盖（rewrite），不需要重入
        // 启动检测，必要资源
        WorkflowException.checkCondition(StringUtils.isEmpty(this.template4directly), "The template directly must not be empty");
        WorkflowException.checkCondition(StringUtils.isEmpty(this.template4answer), "The template answer must not be empty");
        WorkflowException.checkCondition(StringUtils.isEmpty(this.template4query), "The template query must not be empty");
    }

    @Override
    public Object call(FunctionContext functionContext) throws Exception {
        WorkflowTask workTask = functionContext.getWorkTask().printQuery();
        Map<String, Object> data = workTask.getObjectQuery(Map.class);
        // 构建Workflow类型
        String workflow = this.buildWorkflow(workTask, MapUtils.getInteger(data, "type"));
        Boolean necessary = MapUtils.getBoolean(data, "necessary", true);
        if (!necessary) {
            // 不需要规划
            PlanUtils.disablePlan(workTask);
            return this.buildDirectly(workTask, workflow);
        }
        String why = MapUtils.getString(data, "why_do_this");
        WorkflowException.checkCondition(StringUtils.isEmpty(why), "The why_do_this must not be empty");
        String query = this.buildQuery(workTask, why);
        this.notify(workTask);
        this.source(workTask, CliPrinter.format(why, CliPrinter.SIZE_N));
        // 初始化
        String plan = this.buildPlan(functionContext.getWorkTask(), workflow, query);
        String key = PlanUtils.storePlan(workTask, plan);
        if (log.isInfoEnabled()) {
            log.info("The plan key={}", key);
        }
        return this.buildAnswer(workTask, plan);
    }

    @Override
    public void source(WorkflowTask workTask, String content) throws Exception {
        if (!FeatureFlag.isSilent(workTask)) {
            super.source(workTask, content);
        }
    }

    public void notify(WorkflowTask workTask) throws Exception {
        if (!FeatureFlag.isSilent(workTask)) {
            super.source(workTask, CliPrinter.process(PlanCreateFunction.NAME), XmlResourceLang.get(PlanCreateFunction.LANG_KEY_PLAN_CREATE));
        }
    }

    // 构建Plan
    protected String buildPlan(WorkflowTask workTask, String workflow, String query) throws Exception {
        SyncConfig syncConfig = SyncConfig.builder()
                // 依赖主线程和CLI
                .workTask(new HeartbeatWorkTask(this.routerService, workTask, true))
                .metadata(this.buildMetadata(workTask))
                .notifier(Notifier.ENDPOINT)
                .timeout(this.timeout)
                .workflow(workflow)
                .reQuery(query).build();
        return SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get();
    }

    protected String buildWorkflow(WorkflowTask workTask, Integer type) throws Exception {
        // 1=Open-ended questions，2=Closed-ended questions
        WorkflowException.checkCondition(type == null, "The plan type cannot be empty, please set it to 1 for open-ended questions and 2 for closed-ended questions.");
        WorkflowException.checkCondition(type > PlanCreateFunction.CLOSE, "The plan type must be 1 or 2");
        return PlanCreateFunction.OPEN.equals(type) ? "plan@open" : "plan@close";
    }

    protected Map<String, Object> buildMetadata(WorkflowTask workTask) throws Exception {
        Map<String, Object> metadata = new HashMap<String, Object>(workTask.getMetadata());
        // 主动切到思考模式
        metadata.put(FeatureField.KEY_THINKING, true);
        metadata.put(FeatureField.KEY_DAEMON, true);
        metadata.put(FeatureField.KEY_SILENT, true);
        return RequestModelSelect.transfer(workTask, metadata);
    }

    protected String buildDirectly(WorkflowTask workTask, String plan) throws Exception {
        return this.template4directly;
    }

    protected String buildAnswer(WorkflowTask workTask, String plan) throws Exception {
        // 精确替换
        String answer = this.template4answer.replace("#tools", this.update);
        if (log.isWarnEnabled() && !TemplateChecker.check(answer)) {
            log.warn("The answer template contains unexpected characters; please check: {}", answer);
        }
        return answer;
    }

    protected String buildQuery(WorkflowTask workTask, String why) throws Exception {
        String query = this.template4query.replace("#query", workTask.getOriginal());
        query = query.replace("#why", why);
        if (log.isWarnEnabled() && !TemplateChecker.check(query)) {
            log.warn("The query template contains unexpected characters; please check: {}", query);
        }
        return query;
    }

    @Configuration
    @Getter
    @Setter
    public static class InitConfig {

        @Autowired
        protected ResourceService resourceService;

        @Autowired
        protected RouterService routerService;

        @Value("${plan.create.template.directly:classpath:config/plan/directly.md}")
        protected String template4directly;

        @Value("${plan.create.template.answer:classpath:config/plan/answer.md}")
        protected String template4answer;

        @Value("${plan.create.template.query:classpath:config/plan/query.md}")
        protected String template4query;

        @Value("${plan.create.timeout:120000}")
        protected Integer timeout;

        @Value("${plan.create.tools:plan-update}")
        protected String update;

        @Bean(PlanCreateFunction.NAME)
        @ConditionalOnMissingBean(name = PlanCreateFunction.NAME)
        public PlanCreateFunction planCreateFunction() throws Exception {
            PlanCreateFunction planCreateFunction = new PlanCreateFunction();
            BeanUtils.copyProperties(this, planCreateFunction);
            log.info("PlanCreateFunction inited");
            return planCreateFunction;
        }
    }
}
