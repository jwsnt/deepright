package ai.open.right.workflow.flow.track.impl;

import org.easymock.EasyMock;
import org.junit.Test;
import org.springframework.data.redis.connection.RedisConnection;
import org.springframework.data.redis.connection.RedisKeyCommands;
import org.springframework.data.redis.connection.RedisZSetCommands;

import java.nio.charset.StandardCharsets;

public class RedisStoreCallbackTest {


    @Test
    public void testStore() {
        byte[] kBytes = "HELLO".getBytes(StandardCharsets.UTF_8);
        byte[] vBytes = "WORLD".getBytes(StandardCharsets.UTF_8);
        Integer expire = 100;
        Integer num = 10;
        Long now = 10086L;
        RedisConnection connection = EasyMock.createMock(RedisConnection.class);
        EasyMock.expect(connection.zAdd(kBytes, -10086L, vBytes)).andReturn(true).anyTimes();
        RedisZSetCommands zSetCommands = EasyMock.createMock(RedisZSetCommands.class);
        EasyMock.expect(connection.zSetCommands()).andReturn(zSetCommands).anyTimes();
        EasyMock.expect(zSetCommands.zRemRange(kBytes, num, -1)).andReturn(10L).anyTimes();
        RedisKeyCommands keyCommands = EasyMock.createMock(RedisKeyCommands.class);
        EasyMock.expect(connection.keyCommands()).andReturn(keyCommands).anyTimes();
        EasyMock.expect(keyCommands.expire(kBytes, expire)).andReturn(true).anyTimes();
        EasyMock.replay(connection, zSetCommands, keyCommands);
        RedisTrackChatService.RedisStoreCallback store = new RedisTrackChatService.RedisStoreCallback(kBytes, vBytes, expire, num, now);
        store.doInRedis(connection);
        EasyMock.verify(connection, zSetCommands, keyCommands);
    }
}
