package ai.open.right.integration.impl;

import ai.open.right.integration.RightConfig;
import ai.open.right.integration.RightService;
import ai.open.right.integration.RightTask;
import ai.open.right.trace.TraceService;
import ai.open.right.workflow.flow.Workflow;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.sync.SyncCallable;
import ai.open.right.workflow.sync.SyncWriteBack;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.concurrent.ExecutionException;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;

@Setter
@Getter
@Slf4j
public class RightServiceImpl implements RightService {

    protected TraceService traceService;

    protected Workflow workflow;

    protected Integer timeout;

    @Override
    public Future<String> get(RightConfig rightConfig) throws Exception {
        rightConfig.setTrace(this.traceService.getTrace(rightConfig.getTrace()));
        RightFuture rightFuture = this.buildRightFuture(rightConfig);
        RightTask rightTask = this.buildRightTask(rightConfig, rightFuture);
        this.workflow.async(rightTask);
        return rightFuture;
    }

    protected RightTask buildRightTask(RightConfig rightConfig, RightFuture rightFuture) {
        RightTask rightTask = new RightTask(rightConfig, rightFuture).init();
        RightTask.RightTaskChecker.check(rightTask);
        return rightTask;
    }

    protected RightFuture buildRightFuture(RightConfig rightConfig) {
        return new RightFuture(rightConfig.getNotifierWriteBack(), rightConfig.getSyncCallable(), rightConfig.getInterval(), rightConfig.getTimeout(this.timeout));
    }

    @Getter
    @Setter
    @Slf4j
    public static class RightFuture extends SyncWriteBack implements Future<String> {

        protected volatile Boolean done = false;

        public RightFuture(NotifierWriteBack notifierWriteBack, SyncCallable syncCallable, Integer interval, Integer timeout) {
            super(notifierWriteBack, syncCallable, null, interval, timeout, System.currentTimeMillis());
        }

        @Override
        public boolean cancel(boolean mayInterruptIfRunning) {
            return false;
        }

        @Override
        public boolean isCancelled() {
            return false;
        }

        @Override
        public boolean isDone() {
            return this.done;
        }

        @Override
        public String get() throws ExecutionException {
            try {
                // 成功、超时、异常
                return super.get();
            } catch (Exception e) {
                throw new ExecutionException(e);
            } finally {
                this.done = true;
            }
        }

        @Override
        public String get(long timeout, TimeUnit unit) throws ExecutionException {
            if (log.isWarnEnabled()) {
                log.warn("Right integration can not support this method={}-{}", timeout, unit);
            }
            return this.get();
        }
    }

    @ConditionalOnProperty(name = "integration.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected TraceService traceService;

        @Autowired
        protected Workflow workflow;

        @Value("${integration.timeout:1800000}")
        protected Integer timeout;

        @Bean
        @ConditionalOnMissingBean(value = RightService.class)
        public RightService rightService() throws Exception {
            RightServiceImpl rightService = new RightServiceImpl();
            BeanUtils.copyProperties(this, rightService);
            log.info("RightServiceImpl inited: timeout={}", rightService.getTimeout());
            return rightService;
        }
    }
}