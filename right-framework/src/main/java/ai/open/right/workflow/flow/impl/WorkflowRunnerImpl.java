package ai.open.right.workflow.flow.impl;

import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.Workflow;
import ai.open.right.workflow.flow.WorkflowQueue;
import ai.open.right.workflow.flow.WorkflowRunner;
import ai.open.right.workflow.flow.WorkflowTask;
import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.concurrent.ExecutorService;

@Slf4j
@Setter
@Getter
public class WorkflowRunnerImpl implements WorkflowRunner {

    protected ExecutorService executorService;

    protected WorkflowQueue workflowQueue;

    protected Workflow workflow;

    // 处理任务队列的线程数量
    protected Integer threads;

    protected volatile Boolean shutdown = false;

    @PostConstruct
    public void init() {
        for (int i = 0; i < this.threads; i++) {
            this.executorService.execute(this);
        }
        if (log.isDebugEnabled()) {
            log.debug("WorkflowRunnerImpl start={}", this.threads);
        }
    }

    @PreDestroy
    public void destroy() {
        this.shutdown = true;
    }

    @Override
    public void run() {
        while (!this.shutdown) {
            try {
                // Get超时由WorkflowQueue控制
                WorkflowTask task = this.workflowQueue.get();
                if (task != null) {
                    if (log.isDebugEnabled()) {
                        WorkflowRunnerImpl.log.debug("Worker is running={}", task.getConversation());
                    }
                    // WorkflowRunnerImpl仅做Queue调度，由Workflow异步处理任务
                    this.workflow.async(task);
                }
            } catch (Exception e) {
                WorkflowException.dolog(e);
            }
        }
        if (log.isDebugEnabled()) {
            log.debug("WorkflowRunnerImpl shutdown");
        }
    }
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        @Qualifier("executor")
        protected ExecutorService executorService;

        @Autowired
        protected WorkflowQueue workflowQueue;

        @Autowired
        protected Workflow workflow;

        @Value("${queue.threads:5}")
        // 处理任务队列的线程数量
        protected Integer threads;

        @Bean
        @ConditionalOnMissingBean(value = WorkflowRunner.class)
        public WorkflowRunner workflowRunner() throws Exception {
            WorkflowRunnerImpl workflowRunner = new WorkflowRunnerImpl();
            BeanUtils.copyProperties(this, workflowRunner);
            log.info("WorkflowRunnerImpl inited: threads={}", workflowRunner.getThreads());
            return workflowRunner;
        }
    }
}
