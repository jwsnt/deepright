package ai.open.right.workflow.flow.track.impl;

import ai.open.right.WorkflowException;
import ai.open.right.config.RedisConfig;
import ai.open.right.utils.GzipUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.store.Dimension;
import ai.open.right.workflow.flow.track.TrackChat;
import ai.open.right.workflow.flow.track.TrackChatBody;
import ai.open.right.workflow.flow.track.TrackChatService;
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
import org.springframework.dao.DataAccessException;
import org.springframework.data.redis.connection.RedisConnection;
import org.springframework.data.redis.core.RedisCallback;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.ZSetOperations;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.Set;

@Slf4j
@Setter
@Getter
//Chat + Device纬度的任务回放
public class RedisTrackChatService implements TrackChatService {

    private static final List<TrackChatBody> EMPTY = Collections.unmodifiableList(new ArrayList<TrackChatBody>());

    protected RedisTemplate<String, Object> redis4chat;

    // SECONDS
    // Track持久化时间
    protected Integer expire;

    // Track存储条数
    protected Integer max;

    @Override
    public List<TrackChatBody> restore(WorkflowTask workTask) throws Exception {
        try {
            Assert.notNull(this.redis4chat, "Redis4chat can not be empty");
            String key = this.getKey(workTask);
            // 获取并不销毁。只随expire过期
            ZSetOperations<String, Object> zSetOps = this.redis4chat.opsForZSet();
            Set<Object> chats = zSetOps.range(key, 0, this.max);
            if (log.isDebugEnabled()) {
                log.debug("Track chat restore from redis: key={},size={}", key, CollectionUtils.isEmpty(chats) ? 0 : chats.size());
            }
            if (CollectionUtils.isEmpty(chats)) {
                return RedisTrackChatService.EMPTY;
            }
            List<TrackChatBody> chatBodies = new ArrayList<TrackChatBody>();
            for (Object each : chats) {
                try {
                    chatBodies.add(JsonUtils.read(GzipUtils.decompress((byte[]) each), TrackChatBody.class));
                } catch (Exception e) {
                    WorkflowException.dolog(e);
                }
            }
            if (CollectionUtils.isEmpty(chatBodies)) {
                return RedisTrackChatService.EMPTY;
            } else {
                Collections.reverse(chatBodies);
                return chatBodies;
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
            return RedisTrackChatService.EMPTY;
        }
    }

    @Override
    public void store(TrackChat trackChat) throws Exception {
        try {
            Assert.notNull(this.redis4chat, "Redis4chat can not be empty");
            // Init初始化
            String key = this.getKey(trackChat.getDimension());
            byte[] kBytes = key.getBytes(StandardCharsets.UTF_8);
            byte[] vBytes = GzipUtils.compress(JsonUtils.write(trackChat.getTrackChatBody()).getBytes(StandardCharsets.UTF_8));
            List<Object> result = this.redis4chat.executePipelined((new RedisStoreCallback(kBytes, vBytes, this.expire, this.max, trackChat.getTrackChatBody().getTimestamp())));
            if (log.isDebugEnabled()) {
                log.debug("Track chat store to redis: key={},result={}", key, result);
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
    }

    // Chat + Device
    protected String getKey(Dimension dimension) {
        return RedisConfig.DOMAIN + RedisTrackChatService.class.getSimpleName() + dimension.getChat() + dimension.getDevice();
    }

    public static class RedisStoreCallback implements RedisCallback<Void> {

        protected final byte[] kBytes;

        protected final byte[] vBytes;

        protected final Integer expire;

        protected final Integer num;

        protected final Long now;

        public RedisStoreCallback(byte[] kBytes, byte[] vBytes, Integer expire, Integer num, Long now) {
            this.vBytes = vBytes;
            this.expire = expire;
            this.kBytes = kBytes;
            this.num = num;
            this.now = now;
        }

        @Override
        public Void doInRedis(RedisConnection connection) throws DataAccessException {
            connection.zAdd(this.kBytes, -this.now, this.vBytes);
            connection.zSetCommands().zRemRange(kBytes, this.num, -1);
            connection.keyCommands().expire(this.kBytes, this.expire);
            return null;
        }
    }

    @ConditionalOnProperty(name = "track.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired(required = false)
        protected RedisTemplate<String, Object> redis4chat;

        // SECONDS
        @Value("${track.chat.expire:3600000}")
        // Track持久化时间
        protected Integer expire;

        @Value("${track.chat.max:1000}")
        // Track存储条数
        protected Integer max;

        @Bean
        @ConditionalOnMissingBean(value = TrackChatService.class)
        public TrackChatService trackChatService() throws Exception {
            RedisTrackChatService trackChatService = new RedisTrackChatService();
            BeanUtils.copyProperties(this, trackChatService);
            log.info("RedisTrackChatService inited, expire={},max={}", trackChatService.getExpire(), trackChatService.getMax());
            return trackChatService;
        }
    }
}
