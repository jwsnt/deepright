package ai.deepright.module;

import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.FileUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.PropertySource;
import redis.embedded.RedisServer;
import redis.embedded.core.RedisServerBuilder;

import java.io.File;
import java.nio.file.Paths;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

@Slf4j
@Getter
@Setter
// 嵌入式Redis
public class RedisService {

    public static final String NAME = "embedded.redis";

    protected ExecutorService executorService;

    protected RedisServer redisServer;

    protected String dbfilename;

    protected Integer port;

    protected String save;

    protected String dir;

    @PostConstruct
    public void init() throws Exception {
        this.redisServer = this.build(RedisServer.newRedisServer()
                        .port(this.port))
                .build();
        this.executorService = Executors.newSingleThreadExecutor();
        this.executorService.execute(new RedisRunnable(this.redisServer, this.port));
    }

    @PreDestroy
    public void destroy() throws Exception {
        this.redisServer.stop();
        this.executorService.shutdown();
    }

    protected RedisServerBuilder build(RedisServerBuilder redisServerBuilder) throws Exception {
        String userHome = System.getProperty("user.home");
        String dir = Paths.get(this.dir).isAbsolute() ? this.dir : userHome + File.separator + this.dir;
        FileUtils.forceMkdir(new File(dir));
        if (log.isInfoEnabled()) {
            log.info("The redis server config, dbfilename={}, save={}, dir={}", this.dbfilename, this.save, dir);
        }
        redisServerBuilder.setting("dbfilename " + this.dbfilename);
        redisServerBuilder.setting("save " + this.save);
        redisServerBuilder.setting("dir " + dir);
        return redisServerBuilder;
    }

    public static class RedisRunnable implements Runnable {

        protected final RedisServer redisServer;

        protected final Integer port;

        public RedisRunnable(RedisServer redisServer, Integer port) {
            this.redisServer = redisServer;
            this.port = port;
        }

        @Override
        public void run() {
            try {
                if (log.isInfoEnabled()) {
                    log.info("The redis server will be started, port={}", this.port);
                }
                this.redisServer.start();
            } catch (Exception e) {
                log.error("The redis service start failed, please check if the port is in use (lsof -i :{}), message={}", this.port, e.getMessage(), e);
                System.exit(1);
            }
        }
    }

    @PropertySource({"classpath:application.properties", "classpath:right-global.properties", "classpath:right-thread.properties"})
    @ConditionalOnProperty(name = "embedded.redis.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class RedisInitConfig {

        @Value("${embedded.redis.dbfilename:dump.rdb}")
        protected String dbfilename;

        @Value("${spring.data.redis.port:6379}")
        protected Integer port;

        @Value("${embedded.redis.save:60 1}")
        protected String save;

        @Value("${embedded.redis.dir:redis-data}")
        protected String dir;

        @Bean(RedisService.NAME)
        @ConditionalOnMissingBean(name = RedisService.NAME)
        public RedisService redisService() throws Exception {
            RedisService redisService = new RedisService();
            BeanUtils.copyProperties(this, redisService);
            log.info("RedisService inited");
            return redisService;
        }
    }
}
