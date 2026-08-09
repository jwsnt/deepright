package ai.deepright.test;

import ai.deepright.feature.FeatureFlag;
import ai.open.right.WorkflowException;
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

import java.io.BufferedInputStream;
import java.nio.charset.StandardCharsets;

@Slf4j
@Getter
@Setter
public class TestRag extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_test";

    protected ResourceService resourceService;

    protected String template;

    @PostConstruct
    public void init() throws Exception {
        this.template = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template).openStream()), StandardCharsets.UTF_8);
        WorkflowException.checkCondition(StringUtils.isEmpty(this.template), "The template must not be empty");
    }


    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        this.tiny(ragData);
        return new RagAtOnce(ragConfig);
    }

    protected void tiny(RagData ragData) throws Exception {
        if (FeatureFlag.isTest(ragData.getQuery())) {
            ragData.getRequest().getFunCalls().clear();
            ragData.setPrompt(this.template);
        }
    }

    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Autowired
        protected ResourceService resourceService;

        @Value("${test.rag.template:classpath:config/main/test.md}")
        protected String template;

        @Bean(TestRag.RAG_KEY)
        @ConditionalOnMissingBean(name = TestRag.RAG_KEY)
        public TestRag testRag() throws Exception {
            TestRag testRag = new TestRag();
            BeanUtils.copyProperties(this, testRag);
            log.info("TestRag inited, timeout4Condition={}", testRag.getTimeout4Condition());
            return testRag;
        }
    }
}
