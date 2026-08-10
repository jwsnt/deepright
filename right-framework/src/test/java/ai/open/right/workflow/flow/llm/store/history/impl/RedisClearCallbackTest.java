package ai.open.right.workflow.flow.llm.store.history.impl;

import org.easymock.EasyMock;
import org.junit.Test;
import org.springframework.data.redis.connection.RedisConnection;
import org.springframework.data.redis.connection.RedisKeyCommands;
import org.springframework.data.redis.connection.RedisZSetCommands;

import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.Collections;

public class RedisClearCallbackTest {

    /**
     * desc=false：删除比 now 更晚(新)的消息，区间 [-∞, now]。
     * now=-10 为时间戳取负（score），与 Redis 存法 score=-created 一致。
     */
    @Test
    public void doInRedis_descFalse_removesNewerMessages() {
        byte[] kByte = "KEY".getBytes(StandardCharsets.UTF_8);
        RedisConnection connect = EasyMock.createMock(RedisConnection.class);
        RedisZSetCommands redisZSetCommands = EasyMock.createMock(RedisZSetCommands.class);
        RedisKeyCommands redisKeyCommands = EasyMock.createMock(RedisKeyCommands.class);
        EasyMock.expect(connect.zSetCommands()).andReturn(redisZSetCommands).anyTimes();
        EasyMock.expect(connect.keyCommands()).andReturn(redisKeyCommands).anyTimes();
        EasyMock.expect(redisZSetCommands.zRemRangeByScore(kByte, Double.NEGATIVE_INFINITY, -10D)).andReturn(0L).anyTimes();
        EasyMock.replay(redisZSetCommands, redisKeyCommands, connect);
        RedisHistoryStore.RedisClearCallback callback = new RedisHistoryStore.RedisClearCallback(Arrays.asList(kByte), false, -10L);
        callback.doInRedis(connect);
        EasyMock.verify(redisZSetCommands, redisKeyCommands, connect);
    }

    /**
     * desc=true：删除比 now 更早的消息，区间 [now, 0]。
     */
    @Test
    public void doInRedis_descTrue_removesOlderMessages() {
        byte[] kByte = "KEY".getBytes(StandardCharsets.UTF_8);
        RedisConnection connect = EasyMock.createMock(RedisConnection.class);
        RedisZSetCommands redisZSetCommands = EasyMock.createMock(RedisZSetCommands.class);
        RedisKeyCommands redisKeyCommands = EasyMock.createMock(RedisKeyCommands.class);
        EasyMock.expect(connect.zSetCommands()).andReturn(redisZSetCommands).anyTimes();
        EasyMock.expect(connect.keyCommands()).andReturn(redisKeyCommands).anyTimes();
        EasyMock.expect(redisZSetCommands.zRemRangeByScore(kByte, -100L, 0L)).andReturn(0L).anyTimes();
        EasyMock.replay(redisZSetCommands, redisKeyCommands, connect);
        RedisHistoryStore.RedisClearCallback callback = new RedisHistoryStore.RedisClearCallback(Arrays.asList(kByte), true, -100L);
        callback.doInRedis(connect);
        EasyMock.verify(redisZSetCommands, redisKeyCommands, connect);
    }

    /**
     * now=null：没有基准时间，清空全部，走 keyCommands().del(each)。
     */
    @Test
    public void doInRedis_nowNull_deletesKey() {
        byte[] kByte = "KEY".getBytes(StandardCharsets.UTF_8);
        RedisConnection connect = EasyMock.createMock(RedisConnection.class);
        RedisZSetCommands redisZSetCommands = EasyMock.createMock(RedisZSetCommands.class);
        RedisKeyCommands redisKeyCommands = EasyMock.createMock(RedisKeyCommands.class);
        EasyMock.expect(connect.zSetCommands()).andReturn(redisZSetCommands).anyTimes();
        EasyMock.expect(connect.keyCommands()).andReturn(redisKeyCommands).anyTimes();
        EasyMock.expect(redisKeyCommands.del(kByte)).andReturn(1L).once();
        EasyMock.replay(redisZSetCommands, redisKeyCommands, connect);
        RedisHistoryStore.RedisClearCallback callback = new RedisHistoryStore.RedisClearCallback(Collections.singletonList(kByte), false, null);
        callback.doInRedis(connect);
        EasyMock.verify(redisZSetCommands, redisKeyCommands, connect);
    }
}
