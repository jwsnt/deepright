package ai.open.right.workflow.flow.track.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.utils.GzipUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.track.TrackChat;
import ai.open.right.workflow.flow.track.TrackChatBody;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.ZSetOperations;

import java.util.*;

public class RedisTrackChatServiceTest {

    @Test
    public void testKey() {
        RedisTrackChatService redisTrackChatService = new RedisTrackChatService();
        Assert.assertEquals("rightRedisTrackChatServiceUNKNOWNUNKNOWN", redisTrackChatService.getKey(ObjectBuilder.buildLLMQuery()));
    }

    @Test
    public void testStore() throws Exception {
        TrackChatBody trackChatBody = new TrackChatBody(ObjectBuilder.buildSegment());
        TrackChat trackChat = TrackChat.builder()
                .dimension(ObjectBuilder.buildLLMQuery())
                .trackChatBody(trackChatBody)
                .build();
        RedisTrackChatService redisTrackChatService = new RedisTrackChatService();
        RedisTemplate<String, Object> redis = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(redis.executePipelined(EasyMock.anyObject(RedisTrackChatService.RedisStoreCallback.class))).andReturn(new ArrayList<>()).anyTimes();
        redisTrackChatService.setRedis4chat(redis);
        redisTrackChatService.setExpire(100);
        redisTrackChatService.setMax(10);
        EasyMock.replay(redis);
        redisTrackChatService.store(trackChat);
        EasyMock.verify(redis);
    }

    @Test
    public void testStoreWithException() throws Exception {
        TrackChatBody trackChatBody = new TrackChatBody(ObjectBuilder.buildSegment());
        TrackChat trackChat = TrackChat.builder()
                .dimension(ObjectBuilder.buildLLMQuery())
                .trackChatBody(trackChatBody)
                .build();
        RedisTrackChatService redisTrackChatService = new RedisTrackChatService();
        redisTrackChatService.setExpire(100);
        redisTrackChatService.setMax(10);
        redisTrackChatService.store(trackChat);
    }

    @Test
    public void testRestoreWithException() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQuery();
        RedisTrackChatService redisTrackChatService = new RedisTrackChatService();
        redisTrackChatService.setExpire(100);
        redisTrackChatService.setMax(10);
        List<TrackChatBody> body = redisTrackChatService.restore(workflowTask);
        Assert.assertTrue(body.isEmpty());
    }

    @Test
    public void testRestoreWithEmpty() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQuery();
        RedisTrackChatService redisTrackChatService = new RedisTrackChatService();
        RedisTemplate<String, Object> redis = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zSetOperations = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(redis.opsForZSet()).andReturn(zSetOperations).anyTimes();
        Set<Object> chats = new HashSet<>();
        EasyMock.expect(zSetOperations.range("rightRedisTrackChatServiceUNKNOWNUNKNOWN", 0L, 10L)).andReturn(chats).anyTimes();
        redisTrackChatService.setRedis4chat(redis);
        redisTrackChatService.setExpire(100);
        redisTrackChatService.setMax(10);
        EasyMock.replay(redis, zSetOperations);
        List<TrackChatBody> body = redisTrackChatService.restore(workflowTask);
        Assert.assertTrue(body.isEmpty());
        EasyMock.verify(redis, zSetOperations);
    }

    @Test
    public void testRestoreWithInValidData() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQuery();
        RedisTrackChatService redisTrackChatService = new RedisTrackChatService();
        RedisTemplate<String, Object> redis = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zSetOperations = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(redis.opsForZSet()).andReturn(zSetOperations).anyTimes();
        Set<Object> chats = new HashSet<>();
        chats.add(new Date());
        EasyMock.expect(zSetOperations.range("rightRedisTrackChatServiceUNKNOWNUNKNOWN", 0, 10)).andReturn(chats).anyTimes();
        redisTrackChatService.setRedis4chat(redis);
        redisTrackChatService.setExpire(100);
        redisTrackChatService.setMax(10);
        EasyMock.replay(redis, zSetOperations);
        List<TrackChatBody> body = redisTrackChatService.restore(workflowTask);
        Assert.assertTrue(body.isEmpty());
        EasyMock.verify(redis, zSetOperations);
    }

    @Test
    public void testRestoreWithDataAndSort() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQuery();
        RedisTrackChatService redisTrackChatService = new RedisTrackChatService();
        RedisTemplate<String, Object> redis = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zSetOperations = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(redis.opsForZSet()).andReturn(zSetOperations).anyTimes();
        Set<Object> chats = new LinkedHashSet<>();
        TrackChatBody trackChatBody1 = new TrackChatBody(ObjectBuilder.buildSegment());
        trackChatBody1.setContent("A");
        TrackChatBody trackChatBody2 = new TrackChatBody(ObjectBuilder.buildSegment());
        trackChatBody2.setContent("B");
        chats.add(GzipUtils.compress(JsonUtils.write(trackChatBody1)));
        chats.add(GzipUtils.compress(JsonUtils.write(trackChatBody2)));
        EasyMock.expect(zSetOperations.range("rightRedisTrackChatServiceUNKNOWNUNKNOWN", 0, 10)).andReturn(chats).anyTimes();
        redisTrackChatService.setRedis4chat(redis);
        redisTrackChatService.setExpire(100);
        redisTrackChatService.setMax(10);
        EasyMock.replay(redis, zSetOperations);
        List<TrackChatBody> body = redisTrackChatService.restore(workflowTask);
        Assert.assertEquals(Integer.valueOf(2), Integer.valueOf(body.size()));
        Assert.assertEquals("B", body.get(0).getContent());
        Assert.assertEquals("A", body.get(1).getContent());
        EasyMock.verify(redis, zSetOperations);
    }

    @Test
    public void testInit() throws Exception {
        RedisTemplate<String, Object> redis4chat = EasyMock.createMock(RedisTemplate.class);
        EasyMock.replay(redis4chat);
        RedisTrackChatService.InitConfig redisTrackChatService = new RedisTrackChatService.InitConfig();
        redisTrackChatService.setRedis4chat(redis4chat);
        redisTrackChatService.setMax(100);
        redisTrackChatService.setExpire(1000);
        RedisTrackChatService empty = (RedisTrackChatService) redisTrackChatService.trackChatService();
        Assert.assertEquals(redis4chat, empty.getRedis4chat());
        Assert.assertEquals(Integer.valueOf(1000), empty.getExpire());
        Assert.assertEquals(Integer.valueOf(100), empty.getMax());
        EasyMock.verify(redis4chat);
    }
    @Test
    public void testRestoreNullRedis() throws Exception {
        RedisTrackChatService service = new RedisTrackChatService();
        service.setRedis4chat(null);
        Assert.assertTrue(service.restore(ObjectBuilder.buildWorkflowTask()).isEmpty());
    }

    @Test
    public void testStoreNullRedis() throws Exception {
        RedisTrackChatService service = new RedisTrackChatService();
        service.setRedis4chat(null);
        service.store(TrackChat.builder().build()); // Should log error and return
    }
}
