package ai.open.right.listener.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.listener.Event;
import ai.open.right.listener.EventImpl;
import ai.open.right.utils.GzipUtils;
import ai.open.right.utils.JsonUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.junit.jupiter.api.Assertions;
import org.springframework.data.redis.core.RedisCallback;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.ZSetOperations;

import java.util.*;

public class RedisEventListenerTest {

    @Test
    public void testKey() {
        RedisEventListener redisListener = new RedisEventListener();
        Assert.assertEquals("rightRedisEventListenerBIZCHATDEVICE", redisListener.getKey(ObjectBuilder.buildDimension()));
    }

    @Test
    public void testListen() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisCallback.class))).andReturn(Arrays.asList(1)).anyTimes();
        RedisEventListener listener = new RedisEventListener();
        listener.setMaxsize(10);
        listener.setExpire(10);
        listener.setRedis4array(template);
        EasyMock.replay(template);
        listener.listen(ObjectBuilder.buildEvent());
        EasyMock.verify(template);
    }

    @Test
    public void testListenWithFailed() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisCallback.class))).andThrow(new RuntimeException()).anyTimes();
        RedisEventListener listener = new RedisEventListener();
        listener.setMaxsize(10);
        listener.setExpire(10);
        listener.setRedis4array(template);
        EasyMock.replay(template);
        listener.listen(ObjectBuilder.buildEvent());
        EasyMock.verify(template);
    }

    @Test
    public void testObserve() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        EventImpl eventData = new EventImpl();
        eventData.setData("HELLO");
        EasyMock.expect(zset.range(EasyMock.anyString(), EasyMock.anyInt(), EasyMock.anyInt())).andReturn(new HashSet(Arrays.asList(GzipUtils.compress(JsonUtils.write(eventData))))).anyTimes();
        RedisEventListener listener = new RedisEventListener();
        listener.setMaxsize(10);
        listener.setExpire(10);
        listener.setRedis4array(template);
        EasyMock.replay(zset, template);
        List<Event> result = listener.replay(ObjectBuilder.buildDimension());
        Assert.assertEquals(1, result.size());
        Assert.assertEquals(EventImpl.class.cast(result.getFirst()).getData(), "HELLO");
        EasyMock.verify(zset, template);
    }

    @Test
    public void testObserveWithJson() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        EventImpl eventData = new EventImpl();
        eventData.setData(Collections.singletonMap("HELLO", "WORLD"));
        EasyMock.expect(zset.range(EasyMock.anyString(), EasyMock.anyInt(), EasyMock.anyInt())).andReturn(new HashSet(Arrays.asList(GzipUtils.compress(JsonUtils.write(eventData))))).anyTimes();
        RedisEventListener listener = new RedisEventListener();
        listener.setMaxsize(10);
        listener.setExpire(10);
        listener.setRedis4array(template);
        EasyMock.replay(zset, template);
        List<Event> result = listener.replay(ObjectBuilder.buildDimension());
        Assert.assertEquals(1, result.size());
        Assert.assertEquals(Map.class.cast(EventImpl.class.cast(result.getFirst()).getData()).get("HELLO"), "WORLD");
        EasyMock.verify(zset, template);
    }

    @Test
    public void testObserveWithEmpty() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        EventImpl eventData = new EventImpl();
        eventData.setData("HELLO");
        EasyMock.expect(zset.range(EasyMock.anyString(), EasyMock.anyInt(), EasyMock.anyInt())).andReturn(null).anyTimes();
        RedisEventListener listener = new RedisEventListener();
        listener.setMaxsize(10);
        listener.setExpire(10);
        listener.setRedis4array(template);
        EasyMock.replay(zset, template);
        List<Event> result = listener.replay(ObjectBuilder.buildDimension());
        Assert.assertEquals(0, result.size());
        EasyMock.verify(zset, template);
    }

    @Test
    public void testObserveWithInvalidJson() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        EasyMock.expect(zset.range(EasyMock.anyString(), EasyMock.anyInt(), EasyMock.anyInt())).andReturn(new HashSet(Arrays.asList(GzipUtils.compress(JsonUtils.write("1000"))))).anyTimes();
        RedisEventListener listener = new RedisEventListener();
        listener.setMaxsize(10);
        listener.setExpire(10);
        listener.setRedis4array(template);
        EasyMock.replay(zset, template);
        List<Event> result = listener.replay(ObjectBuilder.buildDimension());
        Assert.assertEquals(0, result.size());
    }

    @Test
    public void testObserveWithException() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        EasyMock.expect(zset.range(EasyMock.anyString(), EasyMock.anyInt(), EasyMock.anyInt())).andThrow(new RuntimeException()).anyTimes();
        RedisEventListener listener = new RedisEventListener();
        listener.setMaxsize(10);
        listener.setExpire(10);
        listener.setRedis4array(template);
        EasyMock.replay(zset, template);
        List<Event> result = listener.replay(ObjectBuilder.buildDimension());
        Assert.assertEquals(0, result.size());
    }

    @Test
    public void testSetGet() {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.replay(template);
        RedisEventListener eventListener = new RedisEventListener();
        eventListener.setRedis4array(template);
        eventListener.setExpire(1000);
        eventListener.setMaxsize(1024);
        Assert.assertEquals(template, eventListener.getRedis4array());
        Assert.assertEquals(Integer.valueOf(1000), eventListener.getExpire());
        Assert.assertEquals(Integer.valueOf(1024), eventListener.getMaxsize());
        EasyMock.verify(template);
    }

    @org.junit.jupiter.api.Test
    public void testReplayNullRedis() throws Exception {
        RedisEventListener listener = new RedisEventListener();
        listener.setRedis4array(null);
        List<Event> result = listener.replay(ObjectBuilder.buildDimension());
        Assertions.assertEquals(RedisEventListener.EMPTY, result);
    }

    @org.junit.jupiter.api.Test
    public void testListenNullRedis() throws Exception {
        RedisEventListener listener = new RedisEventListener();
        listener.setRedis4array(null);
        // 内部捕获 Assert.notNull 抛出的异常
        listener.listen(ObjectBuilder.buildEvent());
    }

    @org.junit.jupiter.api.Test
    public void testListenNullData() throws Exception {
        RedisEventListener listener = new RedisEventListener();
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        listener.setRedis4array(template);
        EventImpl event = new EventImpl();
        event.setData(null);
        // 内部捕获 Assert.notNull 抛出的异常
        listener.listen(event);
    }

    @org.junit.jupiter.api.Test
    public void testRedisEventCallbackConstructor() throws Exception {
        EventImpl event = new EventImpl();
        event.setData("test");
        RedisEventListener.RedisEventCallback callback = new RedisEventListener.RedisEventCallback(
                event, 3600, 50, "key".getBytes(), System.currentTimeMillis());
        Assertions.assertNotNull(callback);
    }

    @org.junit.jupiter.api.Test
    public void testReplayEmptyMembers() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        EasyMock.expect(zset.range(EasyMock.anyString(), EasyMock.anyLong(), EasyMock.anyLong())).andReturn(Collections.emptySet()).anyTimes();
        RedisEventListener listener = new RedisEventListener();
        listener.setRedis4array(template);
        listener.setMaxsize(50);
        EasyMock.replay(zset, template);
        List<Event> result = listener.replay(ObjectBuilder.buildDimension());
        Assertions.assertEquals(RedisEventListener.EMPTY, result);
        EasyMock.verify(zset, template);
    }

    @org.junit.jupiter.api.Test
    public void testGetKeyNullDimension() {
        RedisEventListener listener = new RedisEventListener();
        org.junit.jupiter.api.Assertions.assertThrows(NullPointerException.class, () -> {
            listener.getKey(null);
        });
    }

    @org.junit.jupiter.api.Test
    public void testReplayWithCorruptedData() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        // 模拟损坏的数据（非 Gzip 格式）
        EasyMock.expect(zset.range(EasyMock.anyString(), EasyMock.anyLong(), EasyMock.anyLong())).andReturn(new HashSet(Arrays.asList("corrupted".getBytes()))).anyTimes();
        RedisEventListener listener = new RedisEventListener();
        listener.setRedis4array(template);
        listener.setMaxsize(50);
        EasyMock.replay(zset, template);
        List<ai.open.right.listener.Event> result = listener.replay(ObjectBuilder.buildDimension());
        // 损坏的数据应该被跳过，返回空列表或部分列表
        org.junit.jupiter.api.Assertions.assertTrue(result.isEmpty());
        EasyMock.verify(zset, template);
    }

    @org.junit.jupiter.api.Test
    public void testRedisEventCallbackDoInRedis() throws Exception {
        EventImpl event = new EventImpl();
        event.setData("test");
        RedisEventListener.RedisEventCallback callback = new RedisEventListener.RedisEventCallback(
                event, 3600, 50, "key".getBytes(), 1000L);
        org.springframework.data.redis.connection.RedisConnection connection = EasyMock.createMock(org.springframework.data.redis.connection.RedisConnection.class);
        org.springframework.data.redis.connection.RedisZSetCommands zSetCommands = EasyMock.createMock(org.springframework.data.redis.connection.RedisZSetCommands.class);
        org.springframework.data.redis.connection.RedisKeyCommands keyCommands = EasyMock.createMock(org.springframework.data.redis.connection.RedisKeyCommands.class);

        EasyMock.expect(connection.zAdd(EasyMock.anyObject(byte[].class), EasyMock.anyDouble(), EasyMock.anyObject(byte[].class))).andReturn(true);
        EasyMock.expect(connection.zSetCommands()).andReturn(zSetCommands).anyTimes();
        EasyMock.expect(zSetCommands.zRemRange(EasyMock.anyObject(byte[].class), EasyMock.anyLong(), EasyMock.anyLong())).andReturn(1L);
        EasyMock.expect(connection.keyCommands()).andReturn(keyCommands).anyTimes();
        EasyMock.expect(keyCommands.expire(EasyMock.anyObject(byte[].class), EasyMock.anyLong())).andReturn(true);

        EasyMock.replay(connection, zSetCommands, keyCommands);
        callback.doInRedis(connection);
        EasyMock.verify(connection, zSetCommands, keyCommands);
    }
}

