package ai.deepright.plan.function;

import ai.deepright.cli.CliPrinter;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.lang.XmlResourceLang;
import ai.deepright.plan.PlanPattern;
import ai.deepright.plan.PlanUtils;
import ai.deepright.utils.TemplateChecker;
import ai.open.right.resouce.ResourceService;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.function.impl.BaseFunction;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.core.JsonParseException;
import com.fasterxml.jackson.databind.exc.MismatchedInputException;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.io.IOUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.io.BufferedInputStream;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

@Slf4j
@Getter
@Setter
// 仅更新覆盖，创建使用PlanCreateFunction
public class PlanUpdateFunction extends BaseFunction {

    public static final String LANG_KEY_PLAN_UPDATE = "plan.update";

    public static final String NAME = "fun_plan_update";

    protected ResourceService resourceService;

    protected String template4success;

    protected String template4schema;

    protected String template4failed;

    protected Integer timeout;

    @PostConstruct
    public void init() throws Exception {
        // IOUtils/JsonUtils负责关闭资源
        this.template4schema = JsonUtils.write(MapUtils.getMap(MapUtils.getMap(JsonUtils.read(IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4schema).openStream()), StandardCharsets.UTF_8), Map.class), "update"), "funCall"));
        this.template4success = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4success).openStream()), StandardCharsets.UTF_8);
        this.template4failed = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4failed).openStream()), StandardCharsets.UTF_8);
        this.template4failed = this.template4failed.replace("#schema", this.template4schema);
        // 覆盖（rewrite），不需要重入
        // 启动检测，必要资源
        Assert.hasText(this.template4success, "The template success must not be empty");
        Assert.hasText(this.template4schema, "The template schema must not be empty");
        Assert.hasText(this.template4failed, "The template failed must not be empty");
    }

    @Override
    public Object call(FunctionContext functionContext) throws Exception {
        WorkflowTask workTask = functionContext.getWorkTask().printQuery();
        try {
            PlanData planData = this.buildData(workTask).check(this.template4failed);
            Assert.notEmpty(planData.getData(), "The plan update pattern can not be empty");
            this.notify(workTask);
            this.source(workTask, CliPrinter.format(planData.getWhy(), CliPrinter.SIZE_N));
            String plan = PlanUtils.fetchPlan(workTask);
            Assert.hasText(plan, "The plan can not be empty");
            String replace = PlanUtils.replace(workTask, plan, planData.getData());
            PlanUtils.storePlan(workTask, replace);
            return this.buildSuccess(workTask, replace);
        } catch (Exception e) {
            return this.buildFailed(workTask, e.getMessage());
        }
    }

    @Override
    public void source(WorkflowTask workTask, String content) throws Exception {
        if (!FeatureFlag.isSilent(workTask)) {
            super.source(workTask, content);
        }
    }

    public void notify(WorkflowTask workTask) throws Exception {
        if (!FeatureFlag.isSilent(workTask)) {
            super.source(workTask, CliPrinter.process(PlanUpdateFunction.NAME), XmlResourceLang.get(PlanUpdateFunction.LANG_KEY_PLAN_UPDATE));
        }
    }

    protected String buildSuccess(WorkflowTask workTask, String content) throws Exception {
        return this.template4success;
    }

    protected String buildFailed(WorkflowTask workTask, String failed) throws Exception {
        String template = this.template4failed.replace("#content", failed);
        if (log.isWarnEnabled() && !TemplateChecker.check(template)) {
            log.warn("The query template contains unexpected characters; please check: {}", template);
        }
        return template;
    }

    protected String buildSchema(WorkflowTask workTask) throws Exception {
        return "The pattern must not be empty, please strictly follow the schema: " + this.template4schema;
    }

    protected PlanData buildData(WorkflowTask workTask) throws Exception {
        try {
            return workTask.getObjectQuery(PlanUpdate.class);
        } catch (JsonParseException | MismatchedInputException e) {
            if (log.isDebugEnabled()) {
                log.debug(e.getMessage(), e);
            }
            try {
                return workTask.getObjectQuery(PlanDrawBack.class);
            } catch (JsonParseException | MismatchedInputException ex) {
                if (log.isDebugEnabled()) {
                    log.debug(e.getMessage(), e);
                }
                return workTask.getObjectQuery(PlanDeepDive.class);
            }
        }
    }

    public interface PlanData {

        public List<PlanPattern> getData();

        public String getWhy();

        public PlanData init() throws Exception;

        public PlanData check(String schema) throws Exception;
    }

    @Getter
    @Setter
    public static class PlanUpdate implements PlanData {

        protected List<PlanPattern> pattern;

        // Gemini
        protected List<PlanPattern> task;

        @JsonProperty("why_do_this")
        protected String why;

        @Override
        public List<PlanPattern> getData() {
            return this.pattern;
        }

        public PlanUpdate init() throws Exception {
            this.pattern = !CollectionUtils.isEmpty(this.pattern) ? this.pattern : this.task;
            if (!CollectionUtils.isEmpty(this.pattern)) {
                this.pattern = this.pattern.stream()
                        .filter(p -> p != null && p.getPattern() != null && p.getReplacement() != null)
                        .collect(Collectors.toList());
            }
            return this;
        }

        public PlanUpdate check(String schema) throws Exception {
            this.init();
            Assert.notEmpty(this.pattern, schema);
            return this;
        }
    }

    @Getter
    @Setter
    public static class PlanDrawBack implements PlanData {

        protected List<PlanPattern> planData;

        protected List<String> replacement;

        protected List<String> pattern;

        @JsonProperty("why_do_this")
        protected String why;

        public PlanDrawBack check(String schema) throws Exception {
            this.init();
            Assert.notEmpty(this.planData, schema);
            return this;
        }

        @Override
        public PlanData init() throws Exception {
            this.planData = new ArrayList<PlanPattern>(this.pattern.size());
            for (int index = 0; index < this.replacement.size(); index++) {
                String replacement = this.replacement.get(index);
                String pattern = this.pattern.get(index);
                if (replacement != null && pattern != null) {
                    PlanPattern planPattern = new PlanPattern();
                    planPattern.setReplacement(replacement);
                    planPattern.setPattern(pattern);
                    this.planData.add(planPattern);
                }
            }
            return this;
        }

        @Override
        public List<PlanPattern> getData() {
            return this.planData;
        }
    }

    @Getter
    @Setter
    // MiniMax
    public static class PlanDeepDive implements PlanData {

        protected List<PlanPattern> planData;

        protected Map<String, Object> item;

        protected Object replacement;

        protected Object pattern;

        @JsonProperty("why_do_this")
        protected String why;

        public PlanDeepDive check(String schema) throws Exception {
            this.init();
            Assert.notEmpty(this.planData, schema);
            return this;
        }

        @Override
        public PlanData init() throws Exception {
            if (!MapUtils.isEmpty(this.item)) {
                // 尝试从Item解析
                this.planData = new ArrayList<PlanPattern>();
                PlanPattern planPattern = new PlanPattern();
                planPattern.setReplacement(MapUtils.getString(this.item, "replacement"));
                planPattern.setPattern(MapUtils.getString(this.item, "pattern"));
                this.planData.add(planPattern);
            }
            return this;
        }

        @Override
        public List<PlanPattern> getData() {
            return this.planData;
        }
    }

    @Configuration
    @Getter
    @Setter
    public static class InitConfig {

        @Autowired
        protected ResourceService resourceService;

        @Value("${plan.update.template.success:classpath:config/plan/success.md}")
        protected String template4success;

        @Value("${plan.update.template.failed:classpath:config/plan/failed.md}")
        protected String template4failed;

        @Value("${plan.update.template.schema:classpath:config/plan.json}")
        protected String template4schema;

        @Bean(PlanUpdateFunction.NAME)
        @ConditionalOnMissingBean(name = PlanUpdateFunction.NAME)
        public PlanUpdateFunction planUpdateFunction() throws Exception {
            PlanUpdateFunction planUpdateFunction = new PlanUpdateFunction();
            BeanUtils.copyProperties(this, planUpdateFunction);
            log.info("PlanUpdateFunction inited");
            return planUpdateFunction;
        }
    }
}
