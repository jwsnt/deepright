package ai.deepright.cli;

import ai.deepright.feature.FeatureField;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.feature.FeatureUtils;
import ai.deepright.router.RouterAgent;
import ai.deepright.utils.SecurityTruncater;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.ArrayUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;
import org.springframework.util.Assert;

import java.nio.file.Paths;

@Getter
@Setter
@Slf4j
public class CliRag extends RagCondition implements RagService {

    public static final String KEY_DEVICE = "device";

    public static final String KEY_CHAT = "chat";

    public static final String RAG_KEY = "rag_env";

    protected Integer truncate4soul;

    protected Integer truncate4user;

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        // 从CLi更新上报
        this.fetchAgentFromId(ragConfig, ragData);
        this.updateWorkspace(ragConfig, ragData);
        this.updateTerminal(ragConfig, ragData);
        this.updateProvider(ragConfig, ragData);
        this.updateExtract(ragConfig, ragData);
        this.updateDevice(ragConfig, ragData);
        this.updateAgent(ragConfig, ragData);
        this.updateChat(ragConfig, ragData);
        this.updateSoul(ragConfig, ragData);
        this.updateUser(ragConfig, ragData);
        this.updateSys(ragConfig, ragData);
        this.updateApp(ragConfig, ragData);
        this.updateDir(ragConfig, ragData);
        return new RagAtOnce(ragConfig);
    }

    protected void fetchAgentFromId(RagConfig ragConfig, RagData ragData) throws Exception {
        // 获取当前Agent信息
        RouterAgent agent = this.buildAgent(ragData.getQuery());
        if (agent != null) {
            ragData.getQuery().putMetadata(FeatureField.KEY_KNOWLEDGE_CONTENT, agent.getKnowledge());
            ragData.getQuery().putMetadata(FeatureField.KEY_WORKSPACE, agent.getWorkspace());
            ragData.getQuery().putMetadata(FeatureField.KEY_SKILLS, agent.getSkills());
            ragData.getQuery().putMetadata(FeatureField.KEY_MEDIA, agent.getMedia());
            ragData.getQuery().putMetadata(FeatureField.KEY_SOUL, agent.getSoul());
            ragData.getQuery().putMetadata(FeatureField.KEY_USER, agent.getUser());
        }
    }

    protected void updateWorkspace(RagConfig ragConfig, RagData ragData) throws Exception {
        String replace = MapUtils.getString(ragConfig.getGlobalConfig(), FeatureField.KEY_WORKSPACE);
        if (!StringUtils.isEmpty(replace)) {
            String workspace = FeatureUtils.buildWorkspace(ragData.getQuery());
            Assert.hasText(workspace, "The cli workspace can not be empty");
            RagService.updatePrompt(ragConfig, ragData, replace, workspace);
        }
    }

    protected void updateTerminal(RagConfig ragConfig, RagData ragData) throws Exception {
        String replace = MapUtils.getString(ragConfig.getGlobalConfig(), FeatureField.KEY_TERMINAL);
        if (!StringUtils.isEmpty(replace)) {
            String terminal = FeatureUtils.buildTerminal(ragData.getQuery());
            Assert.hasText(terminal, "The cli terminal can not be empty");
            RagService.updatePrompt(ragConfig, ragData, replace, terminal);
        }
    }

    protected void updateProvider(RagConfig ragConfig, RagData ragData) throws Exception {
        String replace = MapUtils.getString(ragConfig.getGlobalConfig(), ProviderRequestService.KEY_PROVIDER.replaceFirst(ProviderRequestService.KEY_INTERNAL, ""));
        if (!StringUtils.isEmpty(replace)) {
            String provider = FeatureUtils.buildTargetProvider(ragData.getQuery());
            Assert.hasText(provider, "The cli provider can not be empty");
            RagService.updatePrompt(ragConfig, ragData, replace, provider);
        }
    }

    protected void updateExtract(RagConfig ragConfig, RagData ragData) throws Exception {
        if (ragData.getQuery().isEntry() && FeatureFlag.isSkillExtract(ragData.getQuery())) {
            // Extract（SOUL/USER/Knowledge提炼）任务关闭上下文
            ragData.getQuery().putMetadata("__containHistories", false);
        }
    }

    protected void updateDevice(RagConfig ragConfig, RagData ragData) throws Exception {
        String replace = MapUtils.getString(ragConfig.getGlobalConfig(), CliRag.KEY_DEVICE);
        if (!StringUtils.isEmpty(replace)) {
            RagService.updatePrompt(ragConfig, ragData, replace, ragData.getQuery().getDevice());
        }
    }

    protected void updateAgent(RagConfig ragConfig, RagData ragData) throws Exception {
        String replace = MapUtils.getString(ragConfig.getGlobalConfig(), FeatureField.KEY_AGENTID);
        if (!StringUtils.isEmpty(replace)) {
            String agent = FeatureUtils.buildAgentId(ragData.getQuery());
            Assert.hasText(agent, "The cli agent can not be empty");
            RagService.updatePrompt(ragConfig, ragData, replace, agent);
        }
    }

    protected void updateChat(RagConfig ragConfig, RagData ragData) throws Exception {
        String replace = MapUtils.getString(ragConfig.getGlobalConfig(), CliRag.KEY_CHAT);
        if (!StringUtils.isEmpty(replace)) {
            RagService.updatePrompt(ragConfig, ragData, replace, ragData.getQuery().getChat());
        }
    }

    // SOUL.md
    protected void updateSoul(RagConfig ragConfig, RagData ragData) throws Exception {
        String replace = MapUtils.getString(ragConfig.getGlobalConfig(), FeatureField.KEY_SOUL);
        if (!StringUtils.isEmpty(replace)) {
            // 可能为空
            String soul = SecurityTruncater.truncate(FeatureUtils.buildSoul(ragData.getQuery()), this.truncate4soul);
            RagService.updatePrompt(ragConfig, ragData, replace, soul);
        }
    }

    // USER.md
    protected void updateUser(RagConfig ragConfig, RagData ragData) throws Exception {
        String replace = MapUtils.getString(ragConfig.getGlobalConfig(), FeatureField.KEY_USER);
        if (!StringUtils.isEmpty(replace)) {
            // 可能为空
            String user = SecurityTruncater.truncate(FeatureUtils.buildUser(ragData.getQuery()), this.truncate4user);
            RagService.updatePrompt(ragConfig, ragData, replace, user);
        }
    }

    protected void updateSys(RagConfig ragConfig, RagData ragData) throws Exception {
        String replace = MapUtils.getString(ragConfig.getGlobalConfig(), FeatureField.KEY_SYS);
        if (!StringUtils.isEmpty(replace)) {
            // 系统类型
            String sys = FeatureUtils.buildSys(ragData.getQuery());
            Assert.hasText(sys, "The cli sys can not be empty");
            RagService.updatePrompt(ragConfig, ragData, replace, sys);
        }
    }

    protected void updateApp(RagConfig ragConfig, RagData ragData) throws Exception {
        String replace = MapUtils.getString(ragConfig.getGlobalConfig(), FeatureField.KEY_APP);
        if (!StringUtils.isEmpty(replace)) {
            String app = FeatureUtils.buildApp(ragData.getQuery());
            Assert.hasText(app, "The cli app can not be empty");
            RagService.updatePrompt(ragConfig, ragData, replace, app);
        }
    }

    protected void updateDir(RagConfig ragConfig, RagData ragData) throws Exception {
        // 首次从App推导
        if (ragData.getQuery().isEntry()) {
            ragData.getQuery().putMetadata(FeatureField.KEY_DIR, StringUtils.defaultIfEmpty(Paths.get(StringUtils.defaultIfEmpty(FeatureUtils.buildApp(ragData.getQuery()), "")).getParent().toString(), ""));
        }
        String replace = MapUtils.getString(ragConfig.getGlobalConfig(), FeatureField.KEY_DIR);
        if (!StringUtils.isEmpty(replace)) {
            RagService.updatePrompt(ragConfig, ragData, replace, ragData.getQuery().getMetadata(FeatureField.KEY_DIR, String.class));
        }
    }

    public RouterAgent buildAgent(WorkflowTask workTask) throws Exception {
        RouterAgent[] agents = FeatureUtils.buildRouterAgents(workTask);
        String agent = FeatureUtils.buildAgentId(workTask);
        if (!ArrayUtils.isEmpty(agents)) {
            for (RouterAgent each : agents) {
                // 对比ID
                if (StringUtils.equalsIgnoreCase(agent, each.getAgent())) {
                    return each;
                }
            }
        }
        return null;
    }

    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Value("${cli.truncate.soul:5120}")
        protected Integer truncate4soul;

        @Value("${cli.truncate.user:5120}")
        protected Integer truncate4user;

        @Bean(CliRag.RAG_KEY)
        @ConditionalOnMissingBean(name = CliRag.RAG_KEY)
        public CliRag cliRag() throws Exception {
            CliRag cliRag = new CliRag();
            BeanUtils.copyProperties(this, cliRag);
            log.info("CliRag inited, timeout4Condition={}", cliRag.getTimeout4Condition());
            return cliRag;
        }
    }
}
