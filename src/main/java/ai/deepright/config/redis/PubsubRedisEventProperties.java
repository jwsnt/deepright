package ai.deepright.config.redis;

import lombok.Getter;
import lombok.Setter;
import org.springframework.boot.context.properties.ConfigurationProperties;

import java.time.Duration;

@Getter
@Setter
@ConfigurationProperties(prefix = "pubsub.redis.event")
public class PubsubRedisEventProperties {

    protected Duration timeout = Duration.ofSeconds(10);

    protected Lettuce lettuce = new Lettuce();

    protected String host = "localhost";

    protected Integer database = 0;

    protected Boolean ssl = false;

    protected Integer port = 6379;

    protected String clientName;

    protected String username;

    protected String password;

    @Getter
    @Setter
    public static class Lettuce {

        protected Duration shutdownTimeout = Duration.ofSeconds(10);
    }
}
