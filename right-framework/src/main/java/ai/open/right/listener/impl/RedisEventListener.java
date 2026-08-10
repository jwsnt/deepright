package ai.open.right.listener.impl;

import ai.open.right.WorkflowException;
import ai.open.right.config.RedisConfig;
import ai.open.right.listener.Event;
import ai.open.right.listener.EventImpl;
import ai.open.right.listener.EventListener;
import ai.open.right.listener.EventReplay;
import ai.open.right.utils.GzipUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.store.Dimension;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.dao.DataAccessException;
import org.springframework.data.redis.connection.RedisConnection;
import org.springframework.data.redis.core.RedisCallback;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.ZSetOperations;
import org.springframework.stereotype.Component;
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
@Component(RedisEventListener.NAME)
@ConditionalOnProperty(name = "event.listener.redis.enable", havingValue = "true", matchIfMissing = false)
// 内部事件监听的Redis实现
public class RedisEventListener implements EventListener, EventReplay {

    public static final List<Event> EMPTY = Collections.unmodifiableList(new ArrayList<Event>());

    public static final String NAME = "redis_event_listener";

    @Autowired(required = false)
    protected RedisTemplate<String, Object> redis4array;

    @Value("${event.listener.redis.maxsize:50}")
    // Redis内部事件监听的最大条数
    protected Integer maxsize;

    // SECONDS
    @Value("${event.listener.redis.expire:3600}")
    // Redis内部事件监听持久化时间
    protected Integer expire;

    @Override
    public List<Event> replay(Dimension dimension) throws Exception {
        try {
            Assert.notNull(this.redis4array, "Redis4array can not be empty");
            String key = this.getKey(dimension);
            ZSetOperations<String, Object> zSetOps = this.redis4array.opsForZSet();
            Set<Object> members = zSetOps.range(key, 0, this.maxsize);
            if (log.isDebugEnabled()) {
                log.debug("Event replay={}-{}", key, CollectionUtils.isEmpty(members) ? 0 : members.size());
            }
            if (CollectionUtils.isEmpty(members)) {
                return RedisEventListener.EMPTY;
            }
            List<Event> events = new ArrayList<Event>();
            for (Object each : members) {
                try {
                    events.add(JsonUtils.read(GzipUtils.decompress((byte[]) each), EventImpl.class));
                } catch (Exception e) {
                    WorkflowException.dolog(e);
                }
            }
            return events;
        } catch (Exception e) {
            WorkflowException.dolog(e);
            return RedisEventListener.EMPTY;
        }
    }

    @Override
    public void listen(Event event) throws Exception {
        try {
            Assert.notNull(this.redis4array, "Redis4array can not be empty");
            Assert.notNull(event.getData(), "Event body can not be empty");
            String key = this.getKey(event);
            List<Object> result = this.redis4array.executePipelined((new RedisEventCallback(new EventImpl(event), expire, this.maxsize, key.getBytes(StandardCharsets.UTF_8), event.getNow())));
            if (log.isDebugEnabled()) {
                log.debug("Event listen={}-{}", key, result);
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
    }

    protected String getKey(Dimension dimension) {
        // Biz + Chat + Device
        return RedisConfig.DOMAIN + RedisEventListener.class.getSimpleName() + dimension.getBiz() + dimension.getChat() + dimension.getDevice();
    }

    public static class RedisEventCallback implements RedisCallback<Void> {

        protected final byte[] eventData;

        protected final Integer expire;

        protected final Integer num;

        protected final byte[] key;

        protected final Long now;

        public RedisEventCallback(EventImpl event, Integer expire, Integer num, byte[] key, Long now) throws Exception {
            this.eventData = GzipUtils.compress(JsonUtils.write(event));
            this.expire = expire;
            this.num = num;
            this.key = key;
            this.now = now;
        }

        @Override
        public Void doInRedis(RedisConnection connection) throws DataAccessException {
            connection.zAdd(this.key, -this.now, this.eventData);
            connection.zSetCommands().zRemRange(this.key, this.num, -1);
            connection.keyCommands().expire(this.key, this.expire);
            return null;
        }
    }
}
