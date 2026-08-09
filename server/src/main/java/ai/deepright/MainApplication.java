package ai.deepright;

import lombok.extern.slf4j.Slf4j;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.autoconfigure.jdbc.DataSourceAutoConfiguration;
import org.springframework.scheduling.annotation.EnableAsync;

@SpringBootApplication(exclude = {DataSourceAutoConfiguration.class})
@EnableAsync(proxyTargetClass = true)
@Slf4j
public class MainApplication {

    public static void main(String[] args) throws Exception {
        if (log.isWarnEnabled()) {
            log.warn("Before starting, please ensure the application has sufficient permissions. For example: sudo chown -R ubuntu:ubuntu /home/ubuntu. The java runtime path={}", System.getProperty("java.home"));
        }
        try {
            SpringApplication.run(MainApplication.class, args);
        } catch (Exception e) {
            // 显式调用System.exit()强制终 JVM，释放资源
            log.error("Application startup failed", e);
            System.exit(1);
        }
    }
}
