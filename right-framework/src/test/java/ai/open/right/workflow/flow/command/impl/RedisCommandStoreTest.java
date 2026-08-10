package ai.open.right.workflow.flow.command.impl;

import ai.open.right.utils.GzipUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.command.QuickCommand;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.data.redis.core.RedisCallback;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.ZSetOperations;

import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Set;

public class RedisCommandStoreTest {

    @Test
    public void testRestoreWithNull() {
        RedisCommandStore store = new RedisCommandStore();
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        Set<Object> members = new HashSet<Object>();
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        EasyMock.expect(zset.range(EasyMock.anyString(), EasyMock.anyLong(), EasyMock.anyLong())).andReturn(members).anyTimes();
        EasyMock.replay(template, zset);
        store.setRedis4array(template);
        store.setExpire(100);
        Assert.assertNull(store.restore("Biz", "Chat", "Device"));
        EasyMock.verify(template, zset);
    }

    @Test
    public void testRestoreWithObject() throws Exception {
        RedisCommandStore store = new RedisCommandStore();
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        Set<Object> members = new HashSet<Object>();
        QuickCommand quickCommand1 = new QuickCommand();
        quickCommand1.setCommand("Command");
        quickCommand1.setContent("Content");
        quickCommand1.setPriority(100L);
        members.add(GzipUtils.compress(JsonUtils.write(quickCommand1)));
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        EasyMock.expect(zset.range(EasyMock.anyString(), EasyMock.anyLong(), EasyMock.anyLong())).andReturn(members).anyTimes();
        EasyMock.replay(template, zset);
        store.setRedis4array(template);
        store.setExpire(10);
        List<QuickCommand> cmd = store.restore("Biz", "Chat", "Device");
        Assert.assertEquals(cmd.size(), 1);
        QuickCommand quickCommand2 = cmd.get(0);
        Assert.assertEquals(quickCommand2.getPriority(), quickCommand1.getPriority());
        Assert.assertEquals(quickCommand2.getContent(), quickCommand1.getContent());
        Assert.assertEquals(quickCommand2.getCommand(), quickCommand1.getCommand());
        EasyMock.verify(template, zset);
    }

    @Test
    public void testRestoreWithInValidObject() throws Exception {
        RedisCommandStore store = new RedisCommandStore();
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        Set<Object> members = new HashSet<Object>();
        QuickCommand quickCommand1 = new QuickCommand();
        quickCommand1.setCommand("Command");
        quickCommand1.setContent("Content");
        quickCommand1.setPriority(100L);
        members.add(JsonUtils.write(quickCommand1).getBytes());
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        EasyMock.expect(zset.range(EasyMock.anyString(), EasyMock.anyLong(), EasyMock.anyLong())).andReturn(members).anyTimes();
        EasyMock.replay(template, zset);
        store.setRedis4array(template);
        store.setExpire(10);
        Assert.assertNull(store.restore("Biz", "Chat", "Device"));
        EasyMock.verify(template, zset);
    }

    @Test
    public void testRestoreWithNullTemplate() throws Exception {
        RedisCommandStore store = new RedisCommandStore();
        store.restore("Biz", "Chat", "Device");
    }

    @Test
    public void testStoreWithEmpty() {
        RedisCommandStore store = new RedisCommandStore();
        store.store(null, 10000, "Biz", "Chat", "Device");
    }

    @Test
    public void testStore() {
        RedisCommandStore store = new RedisCommandStore();
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisCallback.class))).andReturn(null).anyTimes();
        EasyMock.replay(template);
        store.setRedis4array(template);
        QuickCommand quickCommand = new QuickCommand();
        quickCommand.setCommand("Command");
        quickCommand.setContent("Content");
        quickCommand.setPriority(10L);
        List<QuickCommand> command = new ArrayList<QuickCommand>();
        command.add(quickCommand);
        store.store(command, 1000, "Biz", "Chat", "Device");
        EasyMock.verify(template);
    }

    @Test
    public void testStoreWithDefaultExpired() {
        RedisCommandStore store = new RedisCommandStore();
        store.setExpire(1000);
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisCallback.class))).andReturn(null).anyTimes();
        EasyMock.replay(template);
        store.setRedis4array(template);
        QuickCommand quickCommand = new QuickCommand();
        quickCommand.setCommand("Command");
        quickCommand.setContent("Content");
        quickCommand.setPriority(10L);
        List<QuickCommand> command = new ArrayList<QuickCommand>();
        command.add(quickCommand);
        store.store(command, "Biz", "Chat", "Device");
        EasyMock.verify(template);
    }

    @Test
    public void testStoreWithNull() {
        RedisCommandStore store = new RedisCommandStore();
        QuickCommand c1 = new QuickCommand();
        c1.setCommand("Command");
        c1.setContent("Content");
        c1.setPriority(100L);
        List<QuickCommand> command = new ArrayList<QuickCommand>();
        command.add(c1);
        store.store(command, 1000, "Biz", "Chat", "Device");
    }

    @Test
    public void testClearWithNull() {
        RedisCommandStore store = new RedisCommandStore();
        store.clear("Biz", "Chat", "Device");
    }

    @Test
    public void testClear() {
        RedisCommandStore store = new RedisCommandStore();
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.delete(store.getKey("Biz", "Chat", "Device"))).andReturn(true).anyTimes();
        EasyMock.replay(template);
        store.setRedis4array(template);
        store.clear("Biz", "Chat", "Device");
        EasyMock.verify(template);
    }

    @Test
    public void testInit() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.replay(template);
        RedisCommandStore.InitConfig redisCommandStore = new RedisCommandStore.InitConfig();
        redisCommandStore.setRedis4array(template);
        redisCommandStore.setExpire(1000);
        RedisCommandStore empty = (RedisCommandStore) redisCommandStore.redisCommandStore();
        Assert.assertEquals(template, empty.getRedis4array());
        Assert.assertEquals(Integer.valueOf(1000), empty.getExpire());
    }
}
