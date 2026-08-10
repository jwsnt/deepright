package ai.open.right.workflow.flow.llm.store.digest.impl;

import ai.open.right.WorkflowException;
import ai.open.right.config.RedisConfig;
import ai.open.right.utils.GzipUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.store.Dimension;
import ai.open.right.workflow.flow.llm.store.digest.Digest;
import ai.open.right.workflow.flow.llm.store.digest.DigestStore;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.util.StringUtils;

import java.util.Map;
import java.util.concurrent.TimeUnit;

@Slf4j
@Setter
@Getter
public class RedisDigestStore implements DigestStore {

    protected RedisTemplate<String, Object> redis;

    // SECONDS
    // Redis Digest持久化时间
    protected Integer expire;


    @Override
    public Digest upsert(Dimension dimension, String scene, Digest digest) {
        try {
            String key = this.getKey(dimension, scene);
            Object value = this.redis.opsForValue().get(key);
            if (log.isDebugEnabled()) {
                log.debug("Digest data restore from redis: key={},value={}", key, value);
            }
            if (value == null) {
                // Insert
                if (digest.hasDigest()) {
                    this.updateDigest(key, JsonUtils.write(digest.getDigest()));
                }
                return digest;
            }
            // Update
            Map<String, Object> last = JsonUtils.read(GzipUtils.decompress((byte[]) value), Map.class);
            this.updateDigest(key, JsonUtils.write(digest.merge(last).getDigest()));
            return digest;
        } catch (Exception e) {
            WorkflowException.dolog(e);
            return digest;
        }
    }

    protected void updateDigest(String key, String value) throws Exception {
        if (StringUtils.hasText(value)) {
            this.redis.opsForValue().set(key, GzipUtils.compress(value), this.expire, TimeUnit.SECONDS);
            if (log.isDebugEnabled()) {
                log.debug("Digest data update to redis: key={},value={}", key, value);
            }
        }
    }

    protected String getKey(Dimension dimension, String scene) {
        return RedisConfig.DOMAIN + RedisDigestStore.class.getSimpleName() + dimension.getBiz() + dimension.getChat() + scene + dimension.getDevice();
    }

    @ConditionalOnProperty(name = "digest.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        @Qualifier("redis4array")
        protected RedisTemplate<String, Object> redis;

        // SECONDS
        @Value("${digest.expire:300}")
        // Redis Digest持久化时间
        protected Integer expire;

        @Bean
        @ConditionalOnMissingBean(value = DigestStore.class)
        public DigestStore digestStore() throws Exception {
            RedisDigestStore digestStore = new RedisDigestStore();
            BeanUtils.copyProperties(this, digestStore);
            log.info("RedisDigestStore inited: expire={}", digestStore.getExpire());
            return digestStore;
        }
    }
}
