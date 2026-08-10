package ai.open.right.pid.impl;

import ai.open.right.pid.PIDService;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.FileUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.io.File;
import java.lang.management.ManagementFactory;

@Slf4j
@Setter
@Getter
public class PIDServiceImpl implements PIDService {

    // PID文件存储位置
    protected String file;

    protected String pid;

    @PostConstruct
    public void init() throws Exception {
        this.pid = ManagementFactory.getRuntimeMXBean().getName().split("@")[0];
        if (!StringUtils.isEmpty(this.file)) {
            FileUtils.write(new File(this.file), this.pid, "UTF-8");
        }
        if (log.isInfoEnabled()) {
            log.info("App starts with the PID={}", this.pid);
        }
    }

    @Override
    public String pid() throws Exception {
        return this.pid;
    }

    @ConditionalOnProperty(name = "pid.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        // PID文件存储位置
        @Value("${pid.file:}")
        protected String file;

        @Bean
        @ConditionalOnMissingBean(value = PIDService.class)
        public PIDService pidService() throws Exception {
            PIDServiceImpl pidService = new PIDServiceImpl();
            BeanUtils.copyProperties(this, pidService);
            log.info("PIDServiceImpl inited, file={}", pidService.getFile());
            return pidService;
        }
    }
}
