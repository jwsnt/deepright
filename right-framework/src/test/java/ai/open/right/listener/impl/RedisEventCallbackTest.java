package ai.open.right.listener.impl;

import ai.open.right.listener.EventImpl;
import ai.open.right.utils.GzipUtils;
import ai.open.right.utils.JsonUtils;
import org.easymock.EasyMock;
import org.junit.Test;
import org.springframework.data.redis.connection.RedisConnection;
import org.springframework.data.redis.connection.RedisKeyCommands;
import org.springframework.data.redis.connection.RedisZSetCommands;

import java.nio.charset.StandardCharsets;

public class RedisEventCallbackTest {
    @Test
    public void testWithData() throws Exception {
        EventImpl eventData = new EventImpl();
        eventData.setData("HELLO");
        eventData.setNow(10086L);
        byte[] kByte = "KEY".getBytes(StandardCharsets.UTF_8);
        byte[] vByte = GzipUtils.compress(JsonUtils.write(eventData));
        RedisConnection connect = EasyMock.createMock(RedisConnection.class);
        RedisZSetCommands redisZSetCommands = EasyMock.createMock(RedisZSetCommands.class);
        RedisKeyCommands redisKeyCommands = EasyMock.createMock(RedisKeyCommands.class);
        EasyMock.expect(connect.zAdd(kByte, -100D, vByte)).andReturn(true).anyTimes();
        EasyMock.expect(connect.zSetCommands()).andReturn(redisZSetCommands).anyTimes();
        EasyMock.expect(connect.keyCommands()).andReturn(redisKeyCommands).anyTimes();
        EasyMock.expect(redisZSetCommands.zRemRange(kByte, 10L, -1)).andReturn(0L).anyTimes();
        EasyMock.expect(redisKeyCommands.expire(kByte, 10L)).andReturn(true).anyTimes();
        EasyMock.replay(redisKeyCommands, redisZSetCommands, connect);
        RedisEventListener.RedisEventCallback callback = new RedisEventListener.RedisEventCallback(eventData, 10, 10, kByte, 100L) {
        };
        callback.doInRedis(connect);
        EasyMock.verify(redisKeyCommands, redisZSetCommands, connect);
    }
}
