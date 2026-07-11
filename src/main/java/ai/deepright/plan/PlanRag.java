package ai.deepright.plan;

import ai.deepright.feature.FeatureFlag;
import ai.deepright.llm.provider.RequestContextBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.resouce.ResourceService;
import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.flow.llm.store.history.History;
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
import java.util.List;

@Slf4j
@Getter
@Setter
public class PlanRag extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_plan";

    protected ResourceService resourceService;

    protected String template4appendVerify;

    protected String template4updateVerify;

    protected String template4appendShort;

    protected String template4create;

    protected String template4update;

    @PostConstruct
    public void init() throws Exception {
        this.template4updateVerify = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4updateVerify).openStream()), StandardCharsets.UTF_8);
        this.template4appendVerify = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4appendVerify).openStream()), StandardCharsets.UTF_8);
        this.template4appendShort = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4appendShort).openStream()), StandardCharsets.UTF_8);
        this.template4create = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4create).openStream()), StandardCharsets.UTF_8);
        this.template4update = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4update).openStream()), StandardCharsets.UTF_8);
        WorkflowException.check(StringUtils.isEmpty(this.template4updateVerify), "The plan update verify template can not be empty", ProtocolCode.C400);
        WorkflowException.check(StringUtils.isEmpty(this.template4appendVerify), "The plan append verify template can not be empty", ProtocolCode.C400);
        WorkflowException.check(StringUtils.isEmpty(this.template4appendShort), "The plan append short template can not be empty", ProtocolCode.C400);
        WorkflowException.check(StringUtils.isEmpty(this.template4create), "The plan create template can not be empty", ProtocolCode.C400);
        WorkflowException.check(StringUtils.isEmpty(this.template4update), "The plan update template can not be empty", ProtocolCode.C400);
    }

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        // 常量标记，是否需要规划 @See RequestFunCall / PlanCreateFunction
        if (PlanUtils.shouldPlan(ragData.getQuery())) {
            String plan = PlanUtils.fetchPlan(ragData.getQuery());
            if (!StringUtils.isEmpty(plan)) {
                // 提示更新
                History assistant = RequestContextBuilder.buildContext(ragData.getRequest(), this.template4update.replace("#verify", FeatureFlag.isVerify(ragData.getQuery()) ? this.template4updateVerify : "").replace("#plan", plan), History.ROLE_ASSISTANT, PlanUtils.fetchTime(ragData.getQuery()));
                History user = RequestContextBuilder.buildContext(ragData.getRequest(), FeatureFlag.isVerify(ragData.getQuery()) ? this.template4appendVerify : this.template4appendShort, assistant.getCreated() + RequestContextBuilder.NEXT);
                ragData.getRequest().getMessage().getHistories().addAll(List.of(assistant, user));
            } else {
                // 提示创建（Gemini特殊处理）
                ragData.getRequest().getMessage().getHistories().add(RequestContextBuilder.buildContext(ragData.getRequest(), this.template4create, ragData.getRequest().getMessage().getCreated() + RequestContextBuilder.NEXT));
            }
        }
        return new RagAtOnce(ragConfig);
    }

    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Autowired
        protected ResourceService resourceService;

        @Value("${plan.rag.template.append.verify:classpath:config/plan/append_verify.md}")
        protected String template4appendVerify;

        @Value("${plan.rag.template.update.verify:classpath:config/plan/update_verify.md}")
        protected String template4updateVerify;

        @Value("${plan.rag.template.append.short:classpath:config/plan/append_short.md}")
        protected String template4appendShort;

        @Value("${plan.rag.template.update:classpath:config/plan/update.md}")
        protected String template4update;

        @Value("${plan.rag.template.create:classpath:config/plan/create.md}")
        protected String template4create;

        @Bean(PlanRag.RAG_KEY)
        @ConditionalOnMissingBean(name = PlanRag.RAG_KEY)
        public PlanRag planRag() throws Exception {
            PlanRag planRag = new PlanRag();
            BeanUtils.copyProperties(this, planRag);
            log.info("PlanRag inited, timeout4Condition={}", planRag.getTimeout4Condition());
            return planRag;
        }
    }
}
