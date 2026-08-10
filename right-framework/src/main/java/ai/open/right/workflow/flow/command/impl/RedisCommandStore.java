package ai.open.right.workflow.flow.command.impl;

import ai.open.right.WorkflowException;
import ai.open.right.config.RedisConfig;
import ai.open.right.utils.GzipUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.command.QuickCommand;
import ai.open.right.workflow.flow.command.QuickCommandStore;
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
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.ZSetOperations;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;
import java.util.Set;

@Slf4j
@Setter
@Getter
public class RedisCommandStore implements QuickCommandStore {

    protected RedisTemplate<String, Object> redis4array;

    // SECONDS
    // Quick Command缓存时间
    protected Integer expire;

    @Override
    public List<QuickCommand> restore(String biz, String chat, String device) {
        try {
            Assert.notNull(this.redis4array, "Redis4array can not be empty");
            String key = this.getKey(biz, chat, device);
            if (log.isDebugEnabled()) {
                log.debug("Quick command restore key={}", key);
            }
            ZSetOperations<String, Object> zSetOps = this.redis4array.opsForZSet();
            Set<Object> members = zSetOps.range(key, 0, -1);
            if (CollectionUtils.isEmpty(members)) {
                if (log.isDebugEnabled()) {
                    log.debug("Quick command is empty while restoring");
                }
                return null;
            }
            List<QuickCommand> commands = new ArrayList<QuickCommand>();
            for (Object each : members) {
                QuickCommand command = JsonUtils.read(GzipUtils.decompress((byte[]) each), QuickCommand.class);
                commands.add(command);
            }
            if (log.isDebugEnabled()) {
                log.debug("Restore quick command={}-{}", key, commands.size());
            }
            return commands;
        } catch (Exception e) {
            WorkflowException.dolog(e);
            return null;
        }
    }

    @Override
    public void store(List<QuickCommand> commands, Integer expire, String biz, String chat, String device) {
        try {
            if (CollectionUtils.isEmpty(commands)) {
                if (log.isDebugEnabled()) {
                    log.debug("Quick command is empty while storing");
                }
                return;
            }
            Assert.notNull(this.redis4array, "Redis4array can not be empty");
            String key = this.getKey(biz, chat, device);
            if (log.isDebugEnabled()) {
                log.debug("Quick command store key={}", key);
            }
            List<Object> result = this.redis4array.executePipelined((new RedisCallbackImpl(commands, expire != null ? expire : this.expire, key.getBytes(StandardCharsets.UTF_8))));
            if (log.isDebugEnabled()) {
                log.debug("Quick command store result={}-{}", key, result);
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
    }

    @Override
    public void store(List<QuickCommand> commands, String biz, String chat, String device) {
        this.store(commands, this.expire, biz, chat, device);
    }

    @Override
    public void clear(String biz, String chat, String device) {
        try {
            Assert.notNull(this.redis4array, "Redis4array can not be empty");
            String key = this.getKey(biz, chat, device);
            if (log.isInfoEnabled()) {
                log.info("Quick command clear key={}", key);
            }
            this.redis4array.delete(key);
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
    }

    protected String getKey(String biz, String chat, String device) {
        return RedisConfig.DOMAIN + RedisCommandStore.class.getSimpleName() + biz + chat + device;
    }

    @ConditionalOnProperty(name = "command.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected RedisTemplate<String, Object> redis4array;

        // SECONDS
        @Value("${command.expire:300}")
        // Quick Command缓存时间
        protected Integer expire;

        @Bean
        @ConditionalOnMissingBean(value = QuickCommandStore.class)
        public QuickCommandStore redisCommandStore() throws Exception {
            RedisCommandStore redisCommandStore = new RedisCommandStore();
            BeanUtils.copyProperties(this, redisCommandStore);
            log.info("RedisCommandStore inited: expire={}", redisCommandStore.getExpire());
            return redisCommandStore;
        }
    }
}
