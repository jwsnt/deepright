package ai.open.right.workflow.flow.impl;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.Workflow;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.assistant.Assistant;
import ai.open.right.workflow.flow.block.BlockService;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.config.WorkflowConfigService;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.trigger.WorkflowTriggerService;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.ratelimit.RateLimitService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.slf4j.MDC;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.scheduling.annotation.Async;
import org.springframework.util.Assert;

import java.util.Map;

@Setter
@Getter
@Slf4j
public class WorkflowImpl implements Workflow {

    protected WorkflowTriggerService workflowTriggerService;

    protected WorkflowConfigService workflowConfigService;

    // 限流（@limit）
    protected RateLimitService rateLimitService;

    protected Map<String, Assistant> assistant;

    protected NotifierService notifierService;

    protected BlockService blockService;

    // 发生异常时是否推送消息
    protected Boolean messageOnFailed;

    // 单任务最大Loop深度
    protected Integer deepness;

    @Async("executor")
    @Override
    public void async(WorkflowTask workTask) throws Exception {
        this.sync(workTask);
    }

    @Override
    public void sync(WorkflowTask workTask) throws Exception {
        try {
            if (log.isDebugEnabled()) {
                log.debug("The workTask took {} milliseconds before starting execution", workTask.getConsuming());
            }
            this.into(workTask);
            this.rateLimit(workTask);
            // MDC（每次覆盖，不做clear/remove）
            MDC.put("trace", workTask.getTrace());
            MDC.put("dimension", workTask.getDimension());
            WorkflowConfig workflowConfig = this.workflowConfigService.config(workTask);
            if (log.isDebugEnabled()) {
                log.debug("The workTask took {} milliseconds to retrieve the workflow config", workTask.getConsuming());
            }
            Assert.notNull(workflowConfig, "The task config can not be empty");
            // 重写初始化WorkflowTask
            workTask = this.reInitial(workflowConfig, workTask);
            if (this.workflowTriggerService != null) {
                // 触发Workflow监听（准备开始处理指定Workflow）
                this.workflowTriggerService.before(workflowConfig, workTask);
            }
            String name = this.reConfig(workflowConfig, workTask).getAssistant();
            Assistant assistant = this.assistant.get(name);
            Assert.notNull(assistant, "The task assistant can not be empty, please config `xxx.enable`: " + name);
            if (this.allowed(workflowConfig, workTask)) {
                assistant.config(workflowConfig, workTask);
                assistant.execute(workflowConfig, workTask);
            } else {
                // 调用深度检查
                throw new WorkflowException("The task has been rejected, please check for cycles in the workflow", ProtocolCode.C400);
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
            if (this.messageOnFailed) {
                try {
                    // 是否发送失败消息
                    int code = WorkflowException.code(e);
                    // 错误码小于0，需要推端并立即关闭
                    this.notifierService.notify(Segment.failed(workTask, e, code <= 0 ? Notifier.SOURCE : workTask.getNotifier(), code), workTask, workTask);
                } catch (Exception re) {
                    WorkflowException.dolog(re);
                }
            } else {
                throw e;
            }
        } finally {
            Thread.yield();
        }
    }

    protected WorkflowConfig reConfig(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        // 是否开启Track Chat
        if (workflowConfig.hasChatTrack()) {
            if (log.isDebugEnabled()) {
                log.debug("The workflow open track chat");
            }
            workTask.beginChatTrack();
        }
        this.reConfigProvider(workflowConfig, workTask);
        return workflowConfig;
    }

    // 子类覆盖，包装或重写。@See WorkflowTaskWrap
    protected WorkflowTask reInitial(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        workTask.incrDeepness();
        if (log.isDebugEnabled()) {
            log.debug("The workflow deepness={}", workTask.getDeepness());
        }
        return workTask;
    }

    protected void reConfigProvider(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        // 从客户端获取LLM供应商
        String provider = workTask.getMetadata(ProviderRequestService.KEY_PROVIDER, String.class);
        if (!StringUtils.isEmpty(provider)) {
            if (log.isDebugEnabled()) {
                log.debug("The workflow update from customer client: provider={}", provider);
            }
            workflowConfig.getLlmConfig().replaceProvider(provider);
        }
    }

    protected Integer deepness(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        return workflowConfig.getDeepness() != null ? workflowConfig.getDeepness() : this.deepness;
    }

    // 调用深度检查
    protected Boolean allowed(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        this.block(workTask);
        this.closed(workTask);
        // 递增 & 返回
        Integer limit = this.deepness(workflowConfig, workTask);
        Boolean allowed = workTask.getDeepness() < limit;
        if (!allowed && log.isWarnEnabled()) {
            log.warn("The workflow deepness={}, limit={}", workTask.getDeepness(), limit);
        }
        return allowed;
    }

    protected void rateLimit(WorkflowTask workTask) throws Exception {
        // 检查是否限流
        if (this.rateLimitService != null) {
            this.rateLimitService.checkLimit(workTask);
        }
    }

    protected void closed(WorkflowTask workTask) throws Exception {
        workTask.checkClosed();
    }

    protected void block(WorkflowTask workTask) throws Exception {
        if (this.blockService != null) {
            this.blockService.block(workTask);
        }
    }

    protected void into(WorkflowTask workTask) throws Exception {
        if (log.isInfoEnabled()) {
            log.info("The workflow's workTask={} waiting time is={}, deepness={}", SplitUtils.join(workTask.getWorkflow(), workTask.getBiz()), System.currentTimeMillis() - workTask.getCreated(), workTask.getDeepness());
        }
    }
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired(required = false)
        protected WorkflowTriggerService workflowTriggerService;

        @Autowired
        protected WorkflowConfigService workflowConfigService;

        @Autowired(required = false)
        protected RateLimitService rateLimitService;

        @Autowired
        protected Map<String, Assistant> assistant;

        @Autowired
        protected NotifierService notifierService;

        @Autowired(required = false)
        protected BlockService blockService;

        // 发生异常时是否推送消息
        @Value("${message.failed:true}")
        protected Boolean messageOnFailed;

        @Value("${deepness.max:99}")
        // 单任务最大Loop深度
        protected Integer deepness;

        @Bean
        @ConditionalOnMissingBean(value = Workflow.class)
        public Workflow workflow() throws Exception {
            WorkflowImpl workflow = new WorkflowImpl();
            BeanUtils.copyProperties(this, workflow);
            log.info("WorkflowImpl inited: deepness={}, messageOnFailed={}", workflow.getDeepness(), workflow.getMessageOnFailed());
            return workflow;
        }
    }
}
