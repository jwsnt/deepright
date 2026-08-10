package ai.open.right.workflow.flow.impl;

import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.WorkflowQueue;
import ai.open.right.workflow.flow.WorkflowTask;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.scheduling.annotation.Scheduled;

import java.util.concurrent.ArrayBlockingQueue;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.TimeUnit;

@Slf4j
@Setter
@Getter
public class WorkflowQueueImpl implements WorkflowQueue {

    protected BlockingQueue<WorkflowTask> workflowTasks;

    // 任务队列扫描间隔
    protected Integer timeout;

    // 任务最大队列数量
    protected Integer queue;

    @PostConstruct
    public void init() {
        this.workflowTasks = new ArrayBlockingQueue<WorkflowTask>(this.queue);
    }

    public void put(WorkflowTask workTask) throws Exception {
        if (!this.workflowTasks.offer(workTask)) {
            throw new WorkflowException("Workflow queue failed(put): chat=" + workTask.getChat());
        }
    }

    public WorkflowTask get() throws Exception {
        try {
            return this.workflowTasks.poll(this.timeout, TimeUnit.SECONDS);
        } catch (Exception e) {
            WorkflowException.dolog(e);
            return null;
        }
    }

    @Scheduled(initialDelayString = "${monitor.workflowqueue.initialDelay:30000}", fixedRateString = "${monitor.workflowqueue.fixedRate:30000}")
    public String monitor() {
        StringBuffer content = new StringBuffer();
        content.append("WorkflowQueue size=").append(this.workflowTasks.size());
        if (log.isInfoEnabled()) {
            log.info(content.toString());
        }
        return content.toString();
    }
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Value("${queue.timeout:20}")
        // 任务队列扫描间隔
        protected Integer timeout;

        @Value("${queue.size:100}")
        // 任务最大队列数量
        protected Integer queue;

        @Bean
        @ConditionalOnMissingBean(value = WorkflowQueue.class)
        public WorkflowQueue workflowQueue() throws Exception {
            WorkflowQueueImpl workflowQueue = new WorkflowQueueImpl();
            BeanUtils.copyProperties(this, workflowQueue);
            log.info("WorkflowQueueImpl inited: timeout={}, queue={}", workflowQueue.getTimeout(), workflowQueue.getQueue());
            return workflowQueue;
        }
    }
}
