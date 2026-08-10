package ai.open.right.workflow.flow.command;

import ai.open.right.utils.GzipUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.command.impl.RedisCallbackImpl;
import org.easymock.EasyMock;
import org.junit.Test;
import org.springframework.data.redis.connection.RedisConnection;
import org.springframework.data.redis.connection.RedisKeyCommands;
import org.springframework.data.redis.connection.RedisZSetCommands;

import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;

public class RedisCallbackImplTest {

    @Test
    public void testWithData() throws Exception {
        QuickCommand quickComamnd = new QuickCommand();
        quickComamnd.setCommand("Command");
        quickComamnd.setContent("Content");
        quickComamnd.setPriority(100L);
        byte[] kByte = "KEY".getBytes(StandardCharsets.UTF_8);
        byte[] vByte = GzipUtils.compress(JsonUtils.write(quickComamnd).getBytes(StandardCharsets.UTF_8));
        RedisConnection connect = EasyMock.createMock(RedisConnection.class);
        RedisZSetCommands redisZSetCommands = EasyMock.createMock(RedisZSetCommands.class);
        RedisKeyCommands redisKeyCommands = EasyMock.createMock(RedisKeyCommands.class);
        EasyMock.expect(connect.zAdd(kByte, 100D, vByte)).andReturn(true).anyTimes();
        EasyMock.expect(connect.zSetCommands()).andReturn(redisZSetCommands).anyTimes();
        EasyMock.expect(connect.keyCommands()).andReturn(redisKeyCommands).anyTimes();
        EasyMock.expect(redisZSetCommands.zRemRange(kByte, 0, -1)).andReturn(0L).anyTimes();
        EasyMock.expect(redisKeyCommands.expire(kByte, 10L)).andReturn(true).anyTimes();
        EasyMock.replay(redisKeyCommands, redisZSetCommands, connect);
        List<QuickCommand> command = new ArrayList<QuickCommand>();
        command.add(quickComamnd);
        RedisCallbackImpl callback = new RedisCallbackImpl(command, 10, kByte);
        callback.doInRedis(connect);
        EasyMock.verify(redisKeyCommands, redisZSetCommands, connect);
    }

    @Test
    public void testWithNull() throws Exception {
        QuickCommand quickCommand = new QuickCommand();
        quickCommand.setCommand("Command");
        quickCommand.setContent("Content");
        quickCommand.setPriority(100L);
        byte[] kByte = "KEY".getBytes(StandardCharsets.UTF_8);
        byte[] vByte = "HELLO".getBytes(StandardCharsets.UTF_8);
        RedisConnection connect = EasyMock.createMock(RedisConnection.class);
        RedisZSetCommands redisZSetCommands = EasyMock.createMock(RedisZSetCommands.class);
        RedisKeyCommands redisKeyCommands = EasyMock.createMock(RedisKeyCommands.class);
        EasyMock.expect(connect.zAdd(kByte, 10086D, vByte)).andReturn(true).anyTimes();
        EasyMock.expect(connect.zSetCommands()).andReturn(redisZSetCommands).anyTimes();
        EasyMock.expect(connect.keyCommands()).andReturn(redisKeyCommands).anyTimes();
        EasyMock.expect(redisZSetCommands.zRemRange(kByte, 0, -1)).andReturn(0L).anyTimes();
        EasyMock.expect(redisKeyCommands.expire(kByte, 10L)).andReturn(true).anyTimes();
        EasyMock.replay(redisKeyCommands, redisZSetCommands, connect);
        List<QuickCommand> command = new ArrayList<QuickCommand>();
        command.add(null);
        RedisCallbackImpl callback = new RedisCallbackImpl(command, 10, kByte);
        callback.doInRedis(connect);
        EasyMock.verify(redisKeyCommands, redisZSetCommands, connect);
    }
}
