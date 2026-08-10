package ai.open.right.workflow.flow.llm.token.impl;

import ai.open.right.workflow.flow.llm.token.TokenData;
import org.easymock.EasyMock;
import org.junit.Test;
import org.springframework.data.redis.connection.RedisConnection;
import org.springframework.data.redis.connection.RedisKeyCommands;
import org.springframework.data.redis.connection.RedisStringCommands;

import java.nio.charset.StandardCharsets;

public class RedisTokenStatisticCallbackTest {

    @Test
    public void test() {
        byte[] k3 = "i".getBytes(StandardCharsets.UTF_8);
        byte[] k4 = "p".getBytes(StandardCharsets.UTF_8);
        byte[] k1 = "t".getBytes(StandardCharsets.UTF_8);
        byte[] k2 = "c".getBytes(StandardCharsets.UTF_8);
        RedisConnection connect = EasyMock.createMock(RedisConnection.class);
        RedisStringCommands redisStringCommands = EasyMock.createMock(RedisStringCommands.class);
        RedisKeyCommands redisKeyCommands = EasyMock.createMock(RedisKeyCommands.class);
        EasyMock.expect(connect.stringCommands()).andReturn(redisStringCommands).anyTimes();
        EasyMock.expect(connect.keyCommands()).andReturn(redisKeyCommands).anyTimes();
        EasyMock.expect(redisStringCommands.incrBy(k3, 0L)).andReturn(1L).anyTimes();
        EasyMock.expect(redisStringCommands.incrBy(k4, 0L)).andReturn(0L).anyTimes();
        EasyMock.expect(redisStringCommands.incrBy(k1, 10)).andReturn(1L).anyTimes();
        EasyMock.expect(redisStringCommands.incrBy(k2, 20)).andReturn(2L).anyTimes();
        EasyMock.expect(redisKeyCommands.expire(k3, 30L)).andReturn(true).anyTimes();
        EasyMock.expect(redisKeyCommands.expire(k4, 30L)).andReturn(true).anyTimes();
        EasyMock.expect(redisKeyCommands.expire(k1, 30L)).andReturn(true).anyTimes();
        EasyMock.expect(redisKeyCommands.expire(k2, 30L)).andReturn(true).anyTimes();
        EasyMock.replay(redisKeyCommands, redisStringCommands, connect);
        TokenData tokenData = TokenData.builder()
                .cache(20)
                .total(10)
                .build();
        RedisTokenStatisticCallback redisCallback = new RedisTokenStatisticCallback(k3, k4, k1, k2, tokenData, 30);
        redisCallback.doInRedis(connect);
        EasyMock.verify(redisKeyCommands, redisStringCommands, connect);
    }
}
