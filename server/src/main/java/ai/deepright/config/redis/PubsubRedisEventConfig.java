package ai.deepright.config.redis;

import static org.springframework.util.ObjectUtils.isEmpty;

import org.apache.commons.lang3.StringUtils;
import org.apache.commons.pool2.impl.GenericObjectPoolConfig;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.connection.RedisPassword;
import org.springframework.data.redis.connection.RedisStandaloneConfiguration;
import org.springframework.data.redis.connection.lettuce.LettuceConnectionFactory;
import org.springframework.data.redis.connection.lettuce.LettucePoolingClientConfiguration;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.serializer.RedisSerializer;

import java.time.Duration;

@Configuration
@EnableConfigurationProperties(PubsubRedisEventProperties.class)
public class PubsubRedisEventConfig {

    public static final String NAME_FACTORY = "redis4eventConnectionFactory";

    public static final String NAME = "redis4event";

    @Value("${pubsub.redis.event.lettuce.pool.max-active:200}")
    protected Integer maxActive;

    @Value("${pubsub.redis.event.lettuce.pool.max-wait:30000ms}")
    protected Duration maxWait;

    @Value("${pubsub.redis.event.lettuce.pool.max-idle:32}")
    protected Integer maxIdle;

    @Value("${pubsub.redis.event.lettuce.pool.min-idle:8}")
    protected Integer minIdle;

    @Bean(PubsubRedisEventConfig.NAME_FACTORY)
    @ConditionalOnMissingBean(name = PubsubRedisEventConfig.NAME_FACTORY)
    public LettuceConnectionFactory redis4eventConnectionFactory(PubsubRedisEventProperties pubsubEventProperties) {
        RedisStandaloneConfiguration redis = new RedisStandaloneConfiguration();
        redis.setHostName(!StringUtils.isEmpty(pubsubEventProperties.getHost()) ? pubsubEventProperties.getHost() : "localhost");
        redis.setPort(pubsubEventProperties.getPort());
        redis.setDatabase(pubsubEventProperties.getDatabase());
        if (!StringUtils.isEmpty(pubsubEventProperties.getPassword())) {
            redis.setPassword(RedisPassword.of(pubsubEventProperties.getPassword()));
        }
        if (!StringUtils.isEmpty(pubsubEventProperties.getUsername())) {
            redis.setUsername(pubsubEventProperties.getUsername());
        }
        GenericObjectPoolConfig<?> pool = new GenericObjectPoolConfig<>();
        pool.setMaxTotal(this.maxActive);
        pool.setMaxWait(this.maxWait);
        pool.setMaxIdle(this.maxIdle);
        pool.setMinIdle(this.minIdle);
        LettucePoolingClientConfiguration.LettucePoolingClientConfigurationBuilder clientBuilder = LettucePoolingClientConfiguration.builder().poolConfig(pool);
        if (pubsubEventProperties.getLettuce().getShutdownTimeout() != null) {
            clientBuilder.shutdownTimeout(pubsubEventProperties.getLettuce().getShutdownTimeout());
        }
        if (!StringUtils.isEmpty(pubsubEventProperties.getClientName())) {
            clientBuilder.clientName(pubsubEventProperties.getClientName());
        }
        if (pubsubEventProperties.getTimeout() != null) {
            clientBuilder.commandTimeout(pubsubEventProperties.getTimeout());
        }
        if (pubsubEventProperties.getSsl()) {
            clientBuilder.useSsl();
        }
        return new LettuceConnectionFactory(redis, clientBuilder.build());
    }

    @Bean(PubsubRedisEventConfig.NAME)
    @ConditionalOnMissingBean(name = PubsubRedisEventConfig.NAME)
    public RedisTemplate<String, Object> redis4event(@Qualifier("redis4eventConnectionFactory") LettuceConnectionFactory connectionFactory) {
        RedisTemplate<String, Object> redisTemplate = new RedisTemplate<>();
        redisTemplate.setHashValueSerializer(RedisSerializer.byteArray());
        redisTemplate.setHashKeySerializer(RedisSerializer.string());
        redisTemplate.setKeySerializer(RedisSerializer.string());
        redisTemplate.setValueSerializer(RedisSerializer.byteArray());
        redisTemplate.setConnectionFactory(connectionFactory);
        return redisTemplate;
    }
}
