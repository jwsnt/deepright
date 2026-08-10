package ai.open.right.workflow.flow.pubsub.impl;

import org.easymock.EasyMock;
import org.junit.Test;
import org.springframework.data.redis.connection.RedisConnection;
import org.springframework.data.redis.connection.RedisKeyCommands;
import org.springframework.data.redis.connection.RedisListCommands;

import java.nio.charset.StandardCharsets;

public class RedisSubCallbackTest {

    @Test
    public void test() {
        byte[] kByte = "KEY".getBytes(StandardCharsets.UTF_8);
        RedisConnection connect = EasyMock.createMock(RedisConnection.class);
        RedisListCommands redisListCommands = EasyMock.createMock(RedisListCommands.class);
        RedisKeyCommands redisKeyCommands = EasyMock.createMock(RedisKeyCommands.class);
        EasyMock.expect(connect.listCommands()).andReturn(redisListCommands).anyTimes();
        EasyMock.expect(connect.keyCommands()).andReturn(redisKeyCommands).anyTimes();
        EasyMock.expect(redisListCommands.bLPop(10, kByte)).andReturn(null).anyTimes();
        EasyMock.expect(redisKeyCommands.expire(kByte, 10L)).andReturn(true).anyTimes();
        EasyMock.replay(redisKeyCommands, redisListCommands, connect);
        PubSubServiceImpl.RedisSubCallback redisCallback = new PubSubServiceImpl.RedisSubCallback("KEY", 10);
        redisCallback.doInRedis(connect);
        EasyMock.verify(redisKeyCommands, redisListCommands, connect);
    }
}
