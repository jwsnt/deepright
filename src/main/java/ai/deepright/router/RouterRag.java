package ai.deepright.router;

import ai.deepright.feature.FeatureField;
import ai.deepright.feature.FeatureUtils;
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
import org.apache.commons.collections.CollectionUtils;
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
import java.util.List;

@Slf4j
@Getter
@Setter
public class RouterRag extends RagCondition implements RagService {

    public static final String NAME = "rag_router";

    protected ResourceService resourceService;

    protected RouterService routerService;

    protected String template4offline;

    protected String template4online;

    protected String template4main;

    @PostConstruct
    public void init() throws Exception {
        // IOUtils/JsonUtils负责关闭资源
        this.template4offline = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4offline).openStream()), StandardCharsets.UTF_8);
        this.template4online = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4online).openStream()), StandardCharsets.UTF_8);
        this.template4main = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4main).openStream()), StandardCharsets.UTF_8);
        // 覆盖（rewrite），不需要重入
        // 启动检测，必要资源
        Assert.hasText(this.template4offline, "The template offline must not be empty");
        Assert.hasText(this.template4online, "The template online must not be empty");
        Assert.hasText(this.template4main, "The template main must not be empty");
    }

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        // 当上下文超过10K tokens时，模型对尾部指令的注意力权重下降
        RagService.updatePrompt(ragConfig, ragData, ragConfig.getReplace(), this.buildRouter(ragConfig, ragData));
        return new RagAtOnce(ragConfig);
    }

    protected List<RouterDevice> fetchRouter(RagConfig ragConfig, RagData ragData) throws Exception {
        return this.routerService.router(ragData.getQuery());
    }

    protected String buildRouter(RagConfig ragConfig, RagData ragData) throws Exception {
        List<RouterDevice> routerDevice = this.fetchRouter(ragConfig, ragData);
        String router = "";
        if (!CollectionUtils.isEmpty(routerDevice)) {
            // 启动Router
            router = this.buildOnlineHint(ragConfig, ragData) + this.template4main.replace(ragConfig.getReplace(), RouterDevice.buildMarkdown(routerDevice)).replace("#device", ragData.getQuery().getDevice()).replace("#agentId", FeatureUtils.buildAgentId(ragData.getQuery()));
            ragData.getQuery().putMetadata(FeatureField.KEY_ROUTER_STARTUP, true);
        } else {
            router = this.buildOfflineHint(ragConfig, ragData);
        }
        return router;
    }

    protected String buildOfflineHint(RagConfig ragConfig, RagData ragData) throws Exception {
        return this.template4offline;
    }

    protected String buildOnlineHint(RagConfig ragConfig, RagData ragData) throws Exception {
        return this.template4online;
    }

    @Configuration
    @Setter
    @Getter
    public static class RouterInitConfig extends ConditionInitConfig {

        @Autowired
        protected ResourceService resourceService;

        @Autowired
        protected RouterService routerService;

        @Value("${router.template.offline:classpath:config/router/offline.md}")
        protected String template4offline;

        @Value("${router.template.online:classpath:config/router/online.md}")
        protected String template4online;

        @Value("${router.template.main:classpath:config/router/main.md}")
        protected String template4main;

        @Bean(RouterRag.NAME)
        @ConditionalOnMissingBean(name = RouterRag.NAME)
        public RouterRag ragRouter() throws Exception {
            RouterRag routerRag = new RouterRag();
            BeanUtils.copyProperties(this, routerRag);
            log.info("RouterRag inited. timeout4Condition={}", routerRag.getTimeout4Condition());
            return routerRag;
        }
    }
}
