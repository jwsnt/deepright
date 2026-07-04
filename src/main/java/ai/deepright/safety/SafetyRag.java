package ai.deepright.safety;

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
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.io.BufferedInputStream;
import java.nio.charset.StandardCharsets;

@Slf4j
@Getter
@Setter
public class SafetyRag extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_safety";

    protected ResourceService resourceService;

    protected String template;

    @PostConstruct
    public void init() throws Exception {
        // IOUtils/JsonUtils负责关闭资源
        this.template = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template).openStream()), StandardCharsets.UTF_8);
        // 覆盖（rewrite），不需要重入
        // 启动检测，必要资源
        Assert.hasText(this.template, "The template must not be empty");
    }

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        if (log.isDebugEnabled()) {
            log.debug("Rag safety start");
        }
        this.updatePrompt(ragConfig, ragData);
        return new RagAtOnce(ragConfig);
    }

    protected void updatePrompt(RagConfig ragConfig, RagData ragData) throws Exception {
        RagService.updatePrompt(ragConfig, ragData, ragConfig.getReplace(), this.template);
    }

    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Autowired
        protected ResourceService resourceService;

        @Value("${safety.template:classpath:config/safety/main.md}")
        protected String template;

        @Bean(SafetyRag.RAG_KEY)
        @ConditionalOnMissingBean(name = SafetyRag.RAG_KEY)
        public SafetyRag ragSafety() throws Exception {
            SafetyRag safetyRag = new SafetyRag();
            BeanUtils.copyProperties(this, safetyRag);
            log.info("SafetyRag inited. timeout4Condition={}", safetyRag.getTimeout4Condition());
            return safetyRag;
        }
    }
}
