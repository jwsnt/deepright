package ai.open.right.workflow.flow.assistant;

import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.config.WorkflowConfigService;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.util.Map;

@Getter
@Setter
@Slf4j
public class ProxyAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-proxy";

    protected WorkflowConfigService workflowConfigService;

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        ProxyTask proxyTask = this.buildProxyTask(workflowConfig, workTask);
        workflowConfig.setChain(SplitUtils.join(proxyTask.getWorkflow(), proxyTask.getBiz()));
        workTask = this.buildWorkTask(workflowConfig, workTask, proxyTask);
        this.chainOr2Endpoint(workflowConfig, workTask, workTask.getQuery());
    }

    protected WorkflowTask buildWorkTask(WorkflowConfig workflowConfig, WorkflowTask workTask, ProxyTask proxyTask) {
        if (!MapUtils.isEmpty(proxyTask.getMetadata())) {
            workTask.getMetadata().putAll(proxyTask.getMetadata());
        }
        workTask.setQuery(proxyTask.getQuery());
        return workTask;
    }

    protected ProxyTask buildProxyTask(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        ProxyTask proxyTask = workTask.getObjectQuery(ProxyTask.class);
        Assert.notNull(proxyTask, "The proxy task can not be empty");
        Assert.hasText(proxyTask.getBiz(), "The biz can not be empty");
        Assert.hasText(proxyTask.getWorkflow(), "The workflow can not be empty");
        Assert.notNull(this.workflowConfigService.config(proxyTask.getBiz(), proxyTask.getWorkflow()), "The workflow can not be empty: " + SplitUtils.join(proxyTask.getWorkflow(), proxyTask.getBiz()));
        return proxyTask;
    }

    @Getter
    @Setter
    public static class ProxyTask {

        protected Map<String, Object> metadata;

        @JsonProperty("why_do_this")
        protected String whyDoThis;

        protected String workflow;

        protected String query;

        protected String biz;
    }

    @ConditionalOnProperty(name = "proxy.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected WorkflowConfigService workflowConfigService;

        @Bean(ProxyAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = ProxyAssistant.WORKFLOW_NAME)
        public ProxyAssistant proxyAssistant() throws Exception {
            ProxyAssistant proxyAssistant = new ProxyAssistant();
            BeanUtils.copyProperties(this, proxyAssistant);
            log.info("ProxyAssistant inited");
            return proxyAssistant;
        }
    }
}
