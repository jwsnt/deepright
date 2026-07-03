package ai.deepright.workflow;

import ai.deepright.auth.AuthService;
import ai.deepright.feature.FeatureUtils;
import ai.deepright.llm.notifier.MultiSourceNotifier;
import ai.deepright.router.RouterDevice;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.Workflow;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.impl.WorkflowImpl;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;

@Slf4j
@Setter
@Getter
public class ReConfigWorkflow extends WorkflowImpl {

    public static final String NAME = "workflow";

    protected AuthService authService;

    @Override
    protected WorkflowTask reInitial(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        // 原有逻辑
        workTask = super.reInitial(workflowConfig, workTask);
        this.provider(workTask);
        this.block(workTask);
        return workTask;
    }

    @Override
    protected void rateLimit(WorkflowTask workTask) throws Exception {
        if (StringUtils.equalsIgnoreCase(MultiSourceNotifier.MAIN, SplitUtils.join(workTask))) {
            super.rateLimit(workTask);
        }
    }

    protected void provider(WorkflowTask workTask) throws Exception {
        String provider = FeatureUtils.buildTargetProvider(workTask);
        if (this.authService.support(provider)) {
            this.authService.auth(workTask, provider, MapUtils.getString(workTask.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN));
        }
    }

    protected void block(WorkflowTask workTask) throws Exception {
        if (workTask.isEntry()) {
            // @See CliSubBlocker 终止之前同设备同会话并行任务
            this.blockService.submit("main", workTask.getChat(), RouterDevice.key(workTask), workTask, workTask.getCreated() - 1);
        }
    }

    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    @Getter
    @Setter
    public static class ReConfigInitConfig extends InitConfig {

        @Autowired
        protected AuthService authService;

        @Override
        @Bean(ReConfigWorkflow.NAME)
        @ConditionalOnMissingBean(value = Workflow.class)
        public Workflow workflow() throws Exception {
            ReConfigWorkflow workflow = new ReConfigWorkflow();
            BeanUtils.copyProperties(this, workflow);
            log.info("ReConfigWorkflow inited: deepness={}, messageOnFailed={}", workflow.getDeepness(), workflow.getMessageOnFailed());
            return workflow;
        }
    }
}
