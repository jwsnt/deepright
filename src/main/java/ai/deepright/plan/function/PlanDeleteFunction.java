package ai.deepright.plan.function;

import static org.springframework.util.ObjectUtils.isEmpty;

import static org.springframework.util.StringUtils.hasText;




import ai.open.right.protocol.ProtocolCode;

import ai.open.right.WorkflowException;

import ai.deepright.cli.CliPrinter;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.lang.XmlResourceLang;
import ai.deepright.plan.PlanUtils;
import ai.deepright.utils.TemplateChecker;
import ai.open.right.resouce.ResourceService;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.function.impl.BaseFunction;
import jakarta.annotation.PostConstruct;
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

import java.io.BufferedInputStream;
import java.nio.charset.StandardCharsets;

@Slf4j
@Getter
@Setter
public class PlanDeleteFunction extends BaseFunction {

    public static final String LANG_KEY_PLAN_DELETE = "plan.delete";

    public static final String NAME = "fun_plan_delete";

    protected ResourceService resourceService;

    protected String template4delete;

    @PostConstruct
    public void init() throws Exception {
        // IOUtils/JsonUtils负责关闭资源
        this.template4delete = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4delete).openStream()), StandardCharsets.UTF_8);
        // 覆盖（rewrite），不需要重入
        // 启动检测，必要资源
        WorkflowException.check(StringUtils.isEmpty(this.template4delete), "The template delete must not be empty", ProtocolCode.C400);
    }

    @Override
    public Object call(FunctionContext functionContext) throws Exception {
        WorkflowTask workTask = functionContext.getWorkTask().printQuery();
        if (StringUtils.isEmpty(PlanUtils.deletePlan(workTask)) && log.isWarnEnabled()) {
            log.warn("The expected plan was not deleted.");
        }
        this.notify(workTask);
        return this.buildAnswer(workTask);
    }

    public void notify(WorkflowTask workTask) throws Exception {
        if (!FeatureFlag.isSilent(workTask)) {
            super.source(workTask, CliPrinter.process(PlanDeleteFunction.NAME), XmlResourceLang.get(PlanDeleteFunction.LANG_KEY_PLAN_DELETE));
        }
    }

    protected String buildAnswer(WorkflowTask workTask) throws Exception {
        // 精确替换
        String answer = this.template4delete;
        if (log.isWarnEnabled() && !TemplateChecker.check(answer)) {
            log.warn("The answer template contains unexpected characters; please check: {}", answer);
        }
        return answer;
    }

    @Configuration
    @Getter
    @Setter
    public static class InitConfig {

        @Autowired
        protected ResourceService resourceService;

        @Value("${plan.delete.template.delete:classpath:config/plan/delete.md}")
        protected String template4delete;

        @Bean(PlanDeleteFunction.NAME)
        @ConditionalOnMissingBean(name = PlanDeleteFunction.NAME)
        public PlanDeleteFunction planDeleteFunction() throws Exception {
            PlanDeleteFunction planDeleteFunction = new PlanDeleteFunction();
            BeanUtils.copyProperties(this, planDeleteFunction);
            log.info("PlanDeleteFunction inited");
            return planDeleteFunction;
        }
    }
}
