package ai.open.right.workflow.flow.block.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.WorkflowTask;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.ValueOperations;

import java.nio.charset.StandardCharsets;
import java.util.concurrent.TimeUnit;

/**
 * RedisBlockServiceImpl 单元测试
 */
public class RedisBlockServiceImplTest {

    @Test
    public void test() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        RedisTemplate<String, Object> redis = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(redis.opsForValue()).andReturn(valueOperations).anyTimes();
        valueOperations.set("rightRedisBlockServiceImplUNKNOWNUNKNOWNUNKNOWN", workflowTask.getCreated().toString().getBytes(StandardCharsets.UTF_8), 1000, TimeUnit.SECONDS);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(valueOperations.get("rightRedisBlockServiceImplUNKNOWNUNKNOWNUNKNOWN")).andReturn(workflowTask.getCreated().toString().getBytes(StandardCharsets.UTF_8)).anyTimes();
        EasyMock.replay(redis, valueOperations);
        RedisBlockServiceImpl redisBlockService = new RedisBlockServiceImpl();
        redisBlockService.setExpire(1000);
        redisBlockService.setRedis4array(redis);
        redisBlockService.block(workflowTask);
        EasyMock.verify(redis, valueOperations);
    }

    @Test
    public void testWithException1() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(Long.MAX_VALUE + "");
        RedisTemplate<String, Object> redis = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(redis.opsForValue()).andReturn(valueOperations).anyTimes();
        valueOperations.set("rightRedisBlockServiceImplUNKNOWNUNKNOWNUNKNOWN", workflowTask.getCreated().toString().getBytes(StandardCharsets.UTF_8), 1000, TimeUnit.SECONDS);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(valueOperations.get("rightRedisBlockServiceImplUNKNOWNUNKNOWNUNKNOWN")).andReturn(workflowTask.getQuery().getBytes(StandardCharsets.UTF_8)).anyTimes();
        EasyMock.replay(redis, valueOperations);
        RedisBlockServiceImpl redisBlockService = new RedisBlockServiceImpl();
        redisBlockService.setExpire(1000);
        redisBlockService.setRedis4array(redis);
        Assertions.assertThrows(WorkflowException.class, () -> redisBlockService.block(workflowTask));
        EasyMock.verify(redis, valueOperations);
    }

    @Test
    public void testWithException2() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(Long.MAX_VALUE);
        RedisTemplate<String, Object> redis = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(redis.opsForValue()).andReturn(valueOperations).anyTimes();
        valueOperations.set("rightRedisBlockServiceImplUNKNOWNUNKNOWNUNKNOWN", workflowTask.getCreated().toString().getBytes(StandardCharsets.UTF_8), 1000, TimeUnit.SECONDS);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(valueOperations.get("rightRedisBlockServiceImplUNKNOWNUNKNOWNUNKNOWN")).andReturn(String.valueOf(workflowTask.getCreated() + 1).getBytes(StandardCharsets.UTF_8)).anyTimes();
        EasyMock.replay(redis, valueOperations);
        RedisBlockServiceImpl redisBlockService = new RedisBlockServiceImpl();
        redisBlockService.setExpire(1000);
        redisBlockService.setRedis4array(redis);
        Assertions.assertThrows(WorkflowException.class, () -> redisBlockService.block(workflowTask));
        EasyMock.verify(redis, valueOperations);
    }

    @Test
    public void testWithException3() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(Long.MAX_VALUE);
        RedisTemplate<String, Object> redis = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(redis.opsForValue()).andReturn(valueOperations).anyTimes();
        valueOperations.set("rightRedisBlockServiceImplUNKNOWNUNKNOWNUNKNOWN", workflowTask.getCreated().toString().getBytes(StandardCharsets.UTF_8), 1000, TimeUnit.SECONDS);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(valueOperations.get("rightRedisBlockServiceImplUNKNOWNUNKNOWNUNKNOWN")).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(redis, valueOperations);
        RedisBlockServiceImpl redisBlockService = new RedisBlockServiceImpl();
        redisBlockService.setExpire(1000);
        redisBlockService.setRedis4array(redis);
        redisBlockService.block(workflowTask);
        EasyMock.verify(redis, valueOperations);
    }

    @Test
    public void testSubmit() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(Long.MAX_VALUE + "");
        RedisTemplate<String, Object> redis = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(redis.opsForValue()).andReturn(valueOperations).anyTimes();
        valueOperations.set("rightRedisBlockServiceImplUNKNOWNUNKNOWNUNKNOWN", workflowTask.getQuery().getBytes(StandardCharsets.UTF_8), 1000, TimeUnit.SECONDS);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(valueOperations.get("rightRedisBlockServiceImplUNKNOWNUNKNOWNUNKNOWN")).andReturn(workflowTask.getQuery().getBytes(StandardCharsets.UTF_8)).anyTimes();
        EasyMock.replay(redis, valueOperations);
        RedisBlockServiceImpl redisBlockService = new RedisBlockServiceImpl();
        redisBlockService.setExpire(1000);
        redisBlockService.setRedis4array(redis);
        redisBlockService.submit(workflowTask);
        EasyMock.verify(redis, valueOperations);
    }

    @Test
    public void testSubmitWithDefault() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("");
        RedisTemplate<String, Object> redis = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(redis.opsForValue()).andReturn(valueOperations).anyTimes();
        valueOperations.set("rightRedisBlockServiceImplUNKNOWNUNKNOWNUNKNOWN", workflowTask.getCreated().toString().getBytes(StandardCharsets.UTF_8), 1000, TimeUnit.SECONDS);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(valueOperations.get("rightRedisBlockServiceImplUNKNOWNUNKNOWNUNKNOWN")).andReturn(workflowTask.getQuery().getBytes(StandardCharsets.UTF_8)).anyTimes();
        EasyMock.replay(redis, valueOperations);
        RedisBlockServiceImpl redisBlockService = new RedisBlockServiceImpl();
        redisBlockService.setExpire(1000);
        redisBlockService.setRedis4array(redis);
        redisBlockService.submit(workflowTask);
        EasyMock.verify(redis, valueOperations);
    }

    @Test
    public void testSubmitWithException() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(Long.MAX_VALUE + "");
        RedisTemplate<String, Object> redis = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(redis.opsForValue()).andReturn(valueOperations).anyTimes();
        valueOperations.set("rightRedisBlockServiceImplUNKNOWNUNKNOWNUNKNOWN", workflowTask.getQuery().getBytes(StandardCharsets.UTF_8), 1000, TimeUnit.SECONDS);
        EasyMock.expectLastCall().andThrow(new RuntimeException()).anyTimes();
        EasyMock.expect(valueOperations.get("rightRedisBlockServiceImplUNKNOWNUNKNOWNUNKNOWN")).andReturn(workflowTask.getQuery().getBytes(StandardCharsets.UTF_8)).anyTimes();
        EasyMock.replay(redis, valueOperations);
        RedisBlockServiceImpl redisBlockService = new RedisBlockServiceImpl();
        redisBlockService.setExpire(1000);
        redisBlockService.setRedis4array(redis);
        redisBlockService.submit(workflowTask);
        EasyMock.verify(redis, valueOperations);
    }

    @Test
    public void testInit() throws Exception {
        RedisTemplate<String, Object> redis = EasyMock.createMock(RedisTemplate.class);
        RedisBlockServiceImpl.InitConfig initConfig = new RedisBlockServiceImpl.InitConfig();
        initConfig.setRedis4array(redis);
        initConfig.setExpire(1000);
        RedisBlockServiceImpl empty = (RedisBlockServiceImpl) initConfig.blockService();
        Assertions.assertEquals(Integer.valueOf(1000), empty.getExpire());
        Assertions.assertEquals(redis, empty.getRedis4array());
    }

    @Test
    public void testBlockWithNullRedis() {
        RedisBlockServiceImpl redisBlockService = new RedisBlockServiceImpl();
        // 由于 block 内部捕获了 Exception 并仅记录日志，因此当 redis4array 为空时不会抛出异常
        Assertions.assertDoesNotThrow(() -> redisBlockService.block(ObjectBuilder.buildWorkflowTask()));
    }

    @Test
    public void testBlockWithNullVal() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        RedisTemplate<String, Object> redis = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(redis.opsForValue()).andReturn(valueOperations).anyTimes();
        EasyMock.expect(valueOperations.get(EasyMock.anyString())).andReturn(null).anyTimes();
        EasyMock.replay(redis, valueOperations);
        RedisBlockServiceImpl redisBlockService = new RedisBlockServiceImpl();
        redisBlockService.setRedis4array(redis);
        redisBlockService.block(workflowTask);
        EasyMock.verify(redis, valueOperations);
    }

    @Test
    public void testSubmitWithNullRedis() {
        RedisBlockServiceImpl redisBlockService = new RedisBlockServiceImpl();
        // 由于 submit 内部捕获了 Exception 并仅记录日志，因此当 redis4array 为空时不会抛出异常
        Assertions.assertDoesNotThrow(() -> redisBlockService.submit(ObjectBuilder.buildWorkflowTask()));
    }

    @Test
    public void testGetterSetter() {
        RedisBlockServiceImpl redisBlockService = new RedisBlockServiceImpl();
        redisBlockService.setExpire(500);
        Assertions.assertEquals(500, redisBlockService.getExpire());
        RedisTemplate<String, Object> redis = EasyMock.createMock(RedisTemplate.class);
        redisBlockService.setRedis4array(redis);
        Assertions.assertEquals(redis, redisBlockService.getRedis4array());
    }

    @Test
    public void testSubmitWithExplicitTimestamp() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        long explicitTs = 999L;
        RedisTemplate<String, Object> redis = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(redis.opsForValue()).andReturn(valueOperations).anyTimes();
        valueOperations.set("rightRedisBlockServiceImplUNKNOWNUNKNOWNUNKNOWN", String.valueOf(explicitTs).getBytes(StandardCharsets.UTF_8), 1000, TimeUnit.SECONDS);
        EasyMock.expectLastCall().once();
        EasyMock.replay(redis, valueOperations);
        RedisBlockServiceImpl redisBlockService = new RedisBlockServiceImpl();
        redisBlockService.setExpire(1000);
        redisBlockService.setRedis4array(redis);
        redisBlockService.submit(workflowTask, explicitTs);
        EasyMock.verify(redis, valueOperations);
    }

    @Test
    public void testBlockThreeArgs() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        RedisTemplate<String, Object> redis = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        String key = "rightRedisBlockServiceImplbiz1chat1UNKNOWN";
        EasyMock.expect(redis.opsForValue()).andReturn(valueOperations).anyTimes();
        EasyMock.expect(valueOperations.get(key)).andReturn(null).once();
        EasyMock.replay(redis, valueOperations);
        RedisBlockServiceImpl redisBlockService = new RedisBlockServiceImpl();
        redisBlockService.setRedis4array(redis);
        redisBlockService.block("biz1", "chat1", workflowTask);
        EasyMock.verify(redis, valueOperations);
    }

    @Test
    public void testSubmitBizChatDeviceTask_delegatesWithNumericQueryAsLastTime() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("4242424242");
        SubmitCaptureRedisBlockService svc = new SubmitCaptureRedisBlockService();
        svc.submit("bizX", "chatY", "deviceZ", workflowTask);
        Assertions.assertEquals("bizX", svc.capturedBiz);
        Assertions.assertEquals("chatY", svc.capturedChat);
        Assertions.assertEquals("deviceZ", svc.capturedDevice);
        Assertions.assertSame(workflowTask, svc.capturedTask);
        Assertions.assertEquals(4242424242L, svc.capturedTimestamp.longValue());
    }

    @Test
    public void testSubmitBizChatDeviceTask_delegatesWithTimestampWhenQueryNotNumeric() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("not-a-number");
        SubmitCaptureRedisBlockService svc = new SubmitCaptureRedisBlockService();
        svc.submit("bizX", "chatY", "deviceZ", workflowTask);
        Assertions.assertEquals("bizX", svc.capturedBiz);
        Assertions.assertEquals("chatY", svc.capturedChat);
        Assertions.assertEquals("deviceZ", svc.capturedDevice);
        Assertions.assertSame(workflowTask, svc.capturedTask);
        Assertions.assertEquals(workflowTask.getCreated(), svc.capturedTimestamp);
    }

    /**
     * 覆盖五参数 submit，避免 EasyMock 对 RedisTemplate 在部分 JDK 下字节码注入失败，同时验证四参数 submit 对 getLastTime 的委托。
     */
    private static final class SubmitCaptureRedisBlockService extends RedisBlockServiceImpl {
        String capturedBiz;
        String capturedChat;
        String capturedDevice;
        WorkflowTask capturedTask;
        Long capturedTimestamp;

        @Override
        public void submit(String biz, String chat, String device, WorkflowTask workTask, Long timestamp) throws Exception {
            this.capturedBiz = biz;
            this.capturedChat = chat;
            this.capturedDevice = device;
            this.capturedTask = workTask;
            this.capturedTimestamp = timestamp;
        }
    }
}

