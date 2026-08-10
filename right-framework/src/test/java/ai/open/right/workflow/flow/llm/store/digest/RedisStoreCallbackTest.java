package ai.open.right.workflow.flow.llm.store.digest;

import ai.open.right.workflow.flow.llm.store.history.impl.RedisHistoryStore;
import org.easymock.EasyMock;
import org.junit.Test;
import org.springframework.data.redis.connection.RedisConnection;
import org.springframework.data.redis.connection.RedisKeyCommands;
import org.springframework.data.redis.connection.RedisZSetCommands;

import java.nio.charset.StandardCharsets;
import java.util.Arrays;

public class RedisStoreCallbackTest {

    @Test
    public void test() {
        byte[] kByte = "KEY".getBytes(StandardCharsets.UTF_8);
        byte[] vByte = "HELLO".getBytes(StandardCharsets.UTF_8);
        RedisConnection connect = EasyMock.createMock(RedisConnection.class);
        RedisZSetCommands redisZSetCommands = EasyMock.createMock(RedisZSetCommands.class);
        RedisKeyCommands redisKeyCommands = EasyMock.createMock(RedisKeyCommands.class);
        EasyMock.expect(connect.zAdd(kByte, 10086D, vByte)).andReturn(true).anyTimes();
        EasyMock.expect(connect.zSetCommands()).andReturn(redisZSetCommands).anyTimes();
        EasyMock.expect(connect.keyCommands()).andReturn(redisKeyCommands).anyTimes();
        EasyMock.expect(redisZSetCommands.zRemRange(kByte, 10, -1)).andReturn(0L).anyTimes();
        EasyMock.expect(redisKeyCommands.expire(kByte, 10L)).andReturn(true).anyTimes();
        EasyMock.replay(redisKeyCommands, redisZSetCommands, connect);
        RedisHistoryStore.RedisStoreCallback redisCallback = new RedisHistoryStore.RedisStoreCallback(Arrays.asList(kByte), vByte, 10, 10, -10086L);
        redisCallback.doInRedis(connect);
        EasyMock.verify(redisKeyCommands, redisZSetCommands, connect);
    }
}
