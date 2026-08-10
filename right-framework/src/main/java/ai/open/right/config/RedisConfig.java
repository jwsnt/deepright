package ai.open.right.config;

/// ////////////////////////////////////// //////////////
/// 使用虚拟线程时需要考虑Redis堵塞
/// spring.data.redis.lettuce.pool.max-active=200
/// spring.data.redis.lettuce.pool.max-idle=200
/// spring.data.redis.lettuce.pool.min-idle=200
/// spring.data.redis.lettuce.pool.max-wait=500ms
/// spring.data.redis.lettuce.shutdown-timeout=500ms
/// ////////////////////////////////////// //////////////

import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.AnyNestedCondition;
import org.springframework.boot.autoconfigure.condition.ConditionalOnClass;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.boot.autoconfigure.data.redis.RedisProperties;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Conditional;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.DependsOn;
import org.springframework.data.redis.connection.lettuce.LettuceConnectionFactory;
import org.springframework.data.redis.core.RedisOperations;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.serializer.RedisSerializer;

@ConditionalOnClass(RedisOperations.class)
@EnableConfigurationProperties(RedisProperties.class)
@Configuration
@Setter
@Getter
@Slf4j
public class RedisConfig {

    public static final String DOMAIN = "right";

    protected RedisTemplate<String, Object> redis4funCall;

    protected RedisTemplate<String, Object> redis4event;

    protected RedisTemplate<String, Object> redis4array;

    protected RedisTemplate<String, Object> redis4chat;

    @Value("${extreme.enable:false}")
    protected Boolean extreme;

    @Bean("redis4funCall")
    @DependsOn("redis4array")
    @ConditionalOnMissingBean(name = "redis4funCall")
    @ConditionalOnProperty(name = "track.enable", havingValue = "true", matchIfMissing = false)
    public RedisTemplate<String, Object> redis4funCall(LettuceConnectionFactory connectionFactory) throws Exception {
        // connectionFactory.clientConfiguration.poolConfig
        // connectionFactory.configuration(host/port)
        // connectionFactory.client.defaultTimeout
        // connectionFactory.clientConfiguration..timeout
        if (!this.extreme) {
            this.redis4funCall = new RedisTemplate<>();
            this.redis4funCall.setConnectionFactory(connectionFactory);
            this.redis4funCall.setKeySerializer(RedisSerializer.string());
            this.redis4funCall.setValueSerializer(RedisSerializer.byteArray());
            this.redis4funCall.setHashKeySerializer(RedisSerializer.string());
            this.redis4funCall.setHashValueSerializer(RedisSerializer.byteArray());
            this.redis4funCall.afterPropertiesSet();
            return this.redis4funCall;
        } else {
            log.info("The redis funCall bean reuses redis array client (extreme.enable=true) ...");
            return this.redis4array;
        }
    }

    @Bean("redis4array")
    @Conditional(RedisConfigCondition.class)
    @ConditionalOnMissingBean(name = "redis4array")
    public RedisTemplate<String, Object> redis4array(LettuceConnectionFactory connectionFactory) throws Exception {
        this.redis4array = new RedisTemplate<>();
        this.redis4array.setConnectionFactory(connectionFactory);
        this.redis4array.setKeySerializer(RedisSerializer.string());
        this.redis4array.setValueSerializer(RedisSerializer.byteArray());
        this.redis4array.setHashKeySerializer(RedisSerializer.string());
        this.redis4array.setHashValueSerializer(RedisSerializer.byteArray());
        this.redis4array.afterPropertiesSet();
        return this.redis4array;
    }

    @Bean("redis4event")
    @DependsOn("redis4array")
    @ConditionalOnMissingBean(name = "redis4event")
    @ConditionalOnProperty(name = "pubsub.enable", havingValue = "true", matchIfMissing = false)
    public RedisTemplate<String, Object> redis4event(LettuceConnectionFactory connectionFactory) throws Exception {
        if (!this.extreme) {
            this.redis4event = new RedisTemplate<>();
            this.redis4event.setConnectionFactory(connectionFactory);
            this.redis4event.setKeySerializer(RedisSerializer.string());
            this.redis4event.setValueSerializer(RedisSerializer.byteArray());
            this.redis4event.setHashKeySerializer(RedisSerializer.string());
            this.redis4event.setHashValueSerializer(RedisSerializer.byteArray());
            this.redis4event.afterPropertiesSet();
            return this.redis4event;
        } else {
            log.info("The redis event bean reuses redis array client (extreme.enable=true) ...");
            return this.redis4array;
        }
    }

    @Bean("redis4chat")
    @DependsOn("redis4array")
    @ConditionalOnMissingBean(name = "redis4chat")
    @ConditionalOnProperty(name = "track.enable", havingValue = "true", matchIfMissing = false)
    public RedisTemplate<String, Object> redis4chat(LettuceConnectionFactory connectionFactory) throws Exception {
        // connectionFactory.clientConfiguration.poolConfig
        // connectionFactory.configuration(host/port)
        // connectionFactory.client.defaultTimeout
        // connectionFactory.clientConfiguration..timeout
        if (!this.extreme) {
            this.redis4chat = new RedisTemplate<>();
            this.redis4chat.setConnectionFactory(connectionFactory);
            this.redis4chat.setKeySerializer(RedisSerializer.string());
            this.redis4chat.setValueSerializer(RedisSerializer.byteArray());
            this.redis4chat.setHashKeySerializer(RedisSerializer.string());
            this.redis4chat.setHashValueSerializer(RedisSerializer.byteArray());
            this.redis4chat.afterPropertiesSet();
            return this.redis4chat;
        } else {
            log.info("The redis chat bean reuses redis array client (extreme.enable=true) ...");
            return this.redis4array;
        }
    }

    public static class RedisConfigCondition extends AnyNestedCondition {

        // event.listener.redis.enable RedisEventListener
        // command.enable RedisCommandStore
        // history.enable RedisHistoryStore
        // token.enable RedisTokenStatistic
        // block.enable RedisBlockServiceImpl
        public RedisConfigCondition() {
            super(ConfigurationPhase.REGISTER_BEAN);
        }

        @ConditionalOnProperty(name = "event.listener.redis.enable", havingValue = "true")
        static class OnListenerEnable {
        }

        @ConditionalOnProperty(name = "history.enable", havingValue = "true")
        static class OnHistoryEnable {
        }

        @ConditionalOnProperty(name = "command.enable", havingValue = "true")
        static class OnCommandEnable {
        }

        @ConditionalOnProperty(name = "token.enable", havingValue = "true")
        static class OnTokenEnable {
        }

        @ConditionalOnProperty(name = "block.enable", havingValue = "true")
        static class OnBlockEnable {
        }
    }
}
