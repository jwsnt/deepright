package ai.deepright.plan;

import ai.deepright.llm.provider.RequestContextBuilder;
import ai.open.right.resouce.ResourceService;
import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
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
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;
import org.springframework.util.Assert;

import java.io.BufferedInputStream;
import java.nio.charset.StandardCharsets;

@Slf4j
@Getter
@Setter
public class PlanRag extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_plan";

    protected ResourceService resourceService;

    protected String template4create;

    protected String template4update;

    @PostConstruct
    public void init() throws Exception {
        this.template4create = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4create).openStream()), StandardCharsets.UTF_8);
        this.template4update = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4update).openStream()), StandardCharsets.UTF_8);
        Assert.hasText(this.template4create, "The plan create template can not be empty");
        Assert.hasText(this.template4update, "The plan update template can not be empty");
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
                ragData.getRequest().getMessage().getHistories().add(RequestContextBuilder.buildContext(ragData.getRequest(), this.template4update.replace("#plan", plan), PlanUtils.fetchTime(ragData.getQuery())));
            } else {
                // 提示创建（Gemini特殊处理）
                ragData.getRequest().getMessage().getHistories().add(RequestContextBuilder.buildContext(ragData.getRequest(), this.template4create, ragData.getRequest().getMessage().getCreated() + RequestContextBuilder.NEXT));
            }
        }
        return new RagAtOnce(ragConfig);
    }

    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Autowired
        protected ResourceService resourceService;

        @Value("${plan.rag.template.create:classpath:config/plan/create.md}")
        protected String template4create;

        @Value("${plan.rag.template.update:classpath:config/plan/update.md}")
        protected String template4update;

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
