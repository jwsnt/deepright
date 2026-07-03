package ai.deepright.config;

import jakarta.annotation.PreDestroy;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.List;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;

@Configuration
@Slf4j
public class ScheduledConfig {

    protected ScheduledExecutorService scheduled;

    @Value("${scheduled.thread:1}")
    protected Integer thread;

    @Bean({"scheduled"})
    @ConditionalOnMissingBean(name = "scheduled")
    public ScheduledExecutorService scheduled() throws Exception {
       this.scheduled = Executors.newScheduledThreadPool(this.thread);
       return this.scheduled;
    }

    @PreDestroy
    public void destroy() throws Exception {
        if (this.scheduled != null) {
            List<Runnable> tasks = this.scheduled.shutdownNow();
            log.info("The scheduled executor has been closed, discarding {} tasks", tasks.size());
        }
    }
}
