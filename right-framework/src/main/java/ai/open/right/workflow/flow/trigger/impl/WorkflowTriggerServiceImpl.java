package ai.open.right.workflow.flow.trigger.impl;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.trigger.WorkflowTrigger;
import ai.open.right.workflow.flow.trigger.WorkflowTriggerService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.util.Map;

@Setter
@Getter
@Slf4j
public class WorkflowTriggerServiceImpl implements WorkflowTriggerService {

    protected Map<String, WorkflowTrigger> triggers;

    protected WorkflowTrigger global;

    @Override
    public void before(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        if (this.global != null) {
            // 全局
            if (log.isDebugEnabled()) {
                log.debug("Workflow trigger=global");
            }
            this.global.before(workflowConfig, workTask);
        }
        if (workflowConfig.hasTrigger()) {
            String key = workflowConfig.getTrigger();
            WorkflowTrigger trigger = this.triggers.get(key);
            if (log.isInfoEnabled()) {
                log.info("Workflow trigger: key={},trigger={}", key, trigger);
            }
            Assert.notNull(trigger, "Workflow trigger can not be empty: " + key);
            trigger.before(workflowConfig, workTask);
        }
    }

    @ConditionalOnProperty(name = "trigger.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired(required = false)
        protected Map<String, WorkflowTrigger> triggers;

        @Autowired(required = false)
        @Qualifier(WorkflowTrigger.NAME)
        protected WorkflowTrigger global;

        @Bean
        @ConditionalOnMissingBean(value = WorkflowTriggerService.class)
        public WorkflowTriggerService workflowTriggerService() throws Exception {
            WorkflowTriggerServiceImpl workflowTriggerService = new WorkflowTriggerServiceImpl();
            if (!CollectionUtils.isEmpty(this.triggers)) {
                this.triggers.remove(WorkflowTrigger.NAME);
            }
            BeanUtils.copyProperties(this, workflowTriggerService);
            log.info("WorkflowTriggerServiceImpl inited, trigger={}, global={}", workflowTriggerService.getTriggers(), workflowTriggerService.getGlobal());
            return workflowTriggerService;
        }
    }
}
