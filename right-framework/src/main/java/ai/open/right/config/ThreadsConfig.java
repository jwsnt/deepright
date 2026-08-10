package ai.open.right.config;

import ai.open.right.WorkflowException;
import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.ApplicationContext;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.scheduling.annotation.AsyncAnnotationBeanPostProcessor;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.scheduling.concurrent.CustomizableThreadFactory;
import org.springframework.util.Assert;

import java.lang.Thread.UncaughtExceptionHandler;
import java.lang.management.ManagementFactory;
import java.lang.management.OperatingSystemMXBean;
import java.util.List;
import java.util.concurrent.*;

/**
 * @author shenjiawei
 */
@Configuration
@Slf4j
@Setter
@Getter
public class ThreadsConfig {

    // 虚拟线程模式
    public static final Integer MODE_VIRTUAL = 0;

    // 经典线程模式
    public static final Integer MODE_CLASSIC = 1;

    @Autowired
    protected ApplicationContext context;

    // 用于接管Loop流量
    protected ExecutorService executor;

    @Value("${threads.inheritInheritableThreadLocals:false}")
    protected Boolean inheritInheritableThreadLocals;

    @Value("${threads.shutdowninterval:1000}")
    protected Integer shutdowninterval;

    // 经典线程模式时KeepAlive时间
    @Value("${threads.keepalive:60000}")
    protected Integer keepalive;

    // 经典线程模式时的队列数量
    @Value("${threads.queue:500}")
    protected Integer queue;

    // 经典线程模式时的核心线程数量
    @Value("${threads.core:100}")
    protected Integer core;

    // 经典线程模式时最大线程数量
    @Value("${threads.max:100}")
    protected Integer max;

    // 线程模式，虚线程=0，经典线程=1
    @Value("${threads.mode:0}")
    protected Integer mode;

    @PostConstruct
    public void init() {
        Assert.isTrue(!this.context.getBeansOfType(AsyncAnnotationBeanPostProcessor.class).isEmpty(), "The asynchronous configuration must be enabled");
    }

    @Bean("executor")
    @ConditionalOnMissingBean(name = "executor")
    public ExecutorService executor() throws Exception {
        if (ThreadsConfig.MODE_CLASSIC.equals(this.mode)) {
            log.info("Classic threads init, max={}, core={}, queue={}, keep alive={} ", this.max, this.core, this.queue, this.keepalive);
            ThreadFactory base = new CustomizableThreadFactory("MASTER");
            ThreadFactory withHandler = r -> {
                Thread thread = base.newThread(r);
                thread.setUncaughtExceptionHandler(CustomizableUncaughtExceptionHandler.INSTANCE);
                return thread;
            };
            return (this.executor = new ThreadPoolExecutor(this.core, this.max, this.keepalive, TimeUnit.MILLISECONDS, new ArrayBlockingQueue<Runnable>(this.queue), withHandler, new ThreadPoolExecutor.AbortPolicy()));
        } else {
            // 如果虚拟线程执行的任务中存在synchronized、原生方法调用（JNI）、未适配NIO的阻塞 IO Socket（如RestTemplate的HttpURLConnection）会触发虚拟线程钉住（Pinning）
            // 详细模式：打印发生 Pinning 的堆栈
            // -Djdk.tracePinnedThreads=full
            // 简易模式：只打印发生 Pinning 的位置
            // -Djdk.tracePinnedThreads = short
            log.info("Virtual threads init, maximum={}, core={}", ForkJoinPool.commonPool().getPoolSize(), ForkJoinPool.commonPool().getParallelism());
            // 开启 inheritInheritableThreadLocals 会导致虚拟线程在创建时进行大量内存拷贝，高并发下极易引发OOM
            return (this.executor = Executors.newThreadPerTaskExecutor(Thread.ofVirtual().name("Virtual").uncaughtExceptionHandler(CustomizableUncaughtExceptionHandler.INSTANCE).inheritInheritableThreadLocals(this.inheritInheritableThreadLocals).factory()));
        }
    }

    @PreDestroy
    public void destroy() throws Exception {
        if (this.executor != null) {
            List<Runnable> tasks = this.executor.shutdownNow();
            log.info("The thread executor has been closed, discarding {} tasks", tasks.size());
        }
    }

    @Scheduled(initialDelayString = "${monitor.threads.initialDelay:30000}", fixedRateString = "${monitor.threads.fixedRate:30000}")
    public String monitor() throws Exception {
        StringBuffer content = new StringBuffer();
        OperatingSystemMXBean runtime = ManagementFactory.getOperatingSystemMXBean();
        Integer processors = runtime.getAvailableProcessors();
        Double loadAverage = runtime.getSystemLoadAverage();
        Double sys = loadAverage / processors;
        content.append("Exec: ").append(this.executor).append(System.lineSeparator());
        content.append("Sys: ").append(sys).append(System.lineSeparator());
        if (log.isInfoEnabled()) {
            log.info("Threads status={}", content);
        }
        return content.toString();
    }

    public static class CustomizableUncaughtExceptionHandler implements UncaughtExceptionHandler {

       public static final UncaughtExceptionHandler INSTANCE = new CustomizableUncaughtExceptionHandler();

        @Override
        public void uncaughtException(Thread t, Throwable e) {
            if (Exception.class.isAssignableFrom(e.getClass())) {
                WorkflowException.dolog(Exception.class.cast(e));
            } else {
                log.error(e.getMessage(), e);
            }
        }
    }
}
