package ai.open.right.workflow.a2a.server.cmd;

import ai.open.right.resouce.ResourceService;
import ai.open.right.utils.IPUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.a2a.A2ARequest;
import ai.open.right.workflow.a2a.protocol.AgentCard;
import ai.open.right.workflow.a2a.protocol.AgentSkill;
import ai.open.right.workflow.a2a.server.A2ACmdExportService;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.config.WorkflowConfigService;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.io.InputStream;
import java.util.List;

// 获取Agent Card
@Setter
@Getter
@Slf4j
public class A2ACmdExportAgentCard implements A2ACmdExportService {

    // 发现指定Agent
    public static final String URL_BASE = "/.well-known/agent-card.json";

    protected WorkflowConfigService workflowConfigService;

    protected ResourceService resourceService;

    protected AgentCard[] agentCards;

    // Agent Card对应的URL Pattern
    protected String[] patterns;

    // 服务Endpoint Server
    protected String server;

    // 加载A2A配置的URI
    protected String uri;

    @PostConstruct
    public void init() throws Exception {
        try (InputStream input = this.resourceService.url(this.uri).openStream()) {
            // 读取配置
            this.agentCards = JsonUtils.read(input, AgentCard[].class);
            Assert.notEmpty(this.agentCards, "Agent card can not be empty");
            this.patterns = new String[this.agentCards.length];
            for (int index = 0; index < this.agentCards.length; index++) {
                AgentCard agentCard = this.agentCards[index];
                String[] pair = SplitUtils.split(agentCard.getName());
                Assert.isTrue(pair.length == 2, "A2A export(" + agentCard.getName() + ") must conform to the format of Biz@workflow");
                // 更新AgentCard和Pattern的配置
                WorkflowConfig workflowConfig = this.findWorkflowConfig(pair);
                this.updateAgentCard(index, agentCard, workflowConfig);
                if (log.isInfoEnabled()) {
                    log.info("A2A Agent card inited={}-{}", agentCard.getName(), agentCard.getUrl());
                }
            }
        }
    }

    // 从思考链（Workflow）配置更新AgentSkill
    protected void updateAgentSkill(Integer index, AgentCard agentCard, WorkflowConfig workflowConfig) throws Exception {
        if (!agentCard.hasSkill()) {
            // 填充默认Skill
            agentCard.setSkills(List.of(AgentSkill.builder().build()));
        }
    }

    // 从思考链（Workflow）配置更新AgentCard
    protected void updateAgentCard(Integer index, AgentCard agentCard, WorkflowConfig workflowConfig) throws Exception {
        // 更新Endpoint，不同BIZ@Workflow不同Path
        this.patterns[index] = agentCard.getName() + A2ACmdExportAgentCard.URL_BASE;
        agentCard.getCapabilities().setStreaming(workflowConfig.getLlmConfig().getStream());
        agentCard.setDescription(workflowConfig.getDescription());
        agentCard.setUrl(this.server + "/" + agentCard.getName());
        this.updateAgentSkill(index, agentCard, workflowConfig);
    }

    protected WorkflowConfig findWorkflowConfig(String[] pair) throws Exception {
        // 查询WorkflowConfig
        return this.workflowConfigService.config(pair[0], pair[1]);
    }

    // 获取匹配请求的Agent
    protected AgentCard findAgentCard(A2ARequest a2aRequest) throws Exception {
        for (int index = 0; index < this.patterns.length; index++) {
            // 不存在Method，且命中Path
            if (StringUtils.isEmpty(a2aRequest.getMethod()) && StringUtils.endsWithIgnoreCase(a2aRequest.getPath(), this.patterns[index])) {
                return this.agentCards[index];
            }
        }
        return null;
    }

    @Override
    public Boolean support(A2ARequest a2aRequest) throws Exception {
        // 找到了匹配模式
        return this.findAgentCard(a2aRequest) != null;
    }

    @Override
    public void cmd(A2ARequest a2aRequest) throws Exception {
        AgentCard agentCard = this.findAgentCard(a2aRequest);
        Assert.notNull(agentCard, "Agent card can not be empty: " + a2aRequest.getMethod());
        a2aRequest.writeOnce(agentCard);
    }

    @ConditionalOnProperty(name = "a2a.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        public static final String LOCALHOST = "localhost";

        @Autowired
        protected WorkflowConfigService workflowConfigService;

        @Autowired
        protected ResourceService resourceService;

        @Value("${a2a.server:}")
        protected String server;

        @Value("${a2a.port:9996}")
        protected Integer port;

        @Value("${a2a.uri:classpath:a2a.json}")
        protected String uri;

        @Bean(name = A2ACmdExportAgentCard.URL_BASE)
        @ConditionalOnMissingBean(name = A2ACmdExportAgentCard.URL_BASE)
        public A2ACmdExportAgentCard a2aCmdExportAgentCard() throws Exception {
            this.server = StringUtils.defaultIfBlank(this.server, "http://" + StringUtils.defaultIfBlank(this.buildIP(), InitConfig.LOCALHOST) + ":" + this.port);
            A2ACmdExportAgentCard a2aCmdExportAgentCard = new A2ACmdExportAgentCard();
            BeanUtils.copyProperties(this, a2aCmdExportAgentCard);
            log.info("A2ACmdExportAgentCard inited: uri={},server={}", a2aCmdExportAgentCard.getUri(), a2aCmdExportAgentCard.getServer());
            return a2aCmdExportAgentCard;
        }

        protected String buildIP() throws Exception {
            return IPUtils.getIP();
        }
    }
}
