package ai.open.right.workflow.flow.llm.store.digest.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.utils.GzipUtils;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.store.digest.Digest;
import ai.open.right.workflow.flow.llm.store.digest.impl.RedisDigestStore;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.ValueOperations;

import java.io.IOException;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.concurrent.TimeUnit;

public class RedisDigestStoreTest {

    @Test
    public void testGetKey() {
        RedisDigestStore redisDigestStore = new RedisDigestStore();
        Assert.assertEquals("rightRedisDigestStoreUNKNOWNUNKNOWNSCENEUNKNOWN", redisDigestStore.getKey(ObjectBuilder.buildLLMQuery(), "SCENE"));
    }

    @Test
    public void testUpdateDigest() throws Exception {
        RedisTemplate redisTemplate = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(redisTemplate.opsForValue()).andReturn(valueOperations).anyTimes();
        valueOperations.set("KEY1", GzipUtils.compress("VAL1"), 10, TimeUnit.SECONDS);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(redisTemplate, valueOperations);
        RedisDigestStore redisDigestStore = new RedisDigestStore();
        redisDigestStore.setRedis(redisTemplate);
        redisDigestStore.setExpire(10);
        redisDigestStore.updateDigest("KEY1", "VAL1");
        EasyMock.verify(redisTemplate, valueOperations);
    }

    @Test
    public void testUpsertWithRestoreNull() {
        RedisTemplate redisTemplate = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(redisTemplate.opsForValue()).andReturn(valueOperations).anyTimes();
        EasyMock.expect(valueOperations.get("rightRedisDigestStoreUNKNOWNUNKNOWNSCENEUNKNOWN")).andReturn(null).anyTimes();
        EasyMock.replay(redisTemplate, valueOperations);
        RedisDigestStore redisDigestStore = new RedisDigestStore() {
            protected void updateDigest(String key, String value) throws Exception {

            }
        };
        redisDigestStore.setRedis(redisTemplate);
        redisDigestStore.setExpire(10);
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        Digest digest = new Digest(new HashMap<String, Object>(), new ArrayList<String>());
        digest.getDigest().put("KEY1", "VAL1");
        redisDigestStore.upsert(query, "SCENE", digest);
        EasyMock.verify(redisTemplate, valueOperations);
    }

    @Test
    public void testUpsertWithRestoreEmpty() throws IOException {
        RedisTemplate redisTemplate = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(redisTemplate.opsForValue()).andReturn(valueOperations).anyTimes();
        EasyMock.expect(valueOperations.get("rightRedisDigestStoreUNKNOWNUNKNOWNSCENEUNKNOWN")).andReturn(GzipUtils.compress("{\"KEY1\":\"VAL1\"}")).anyTimes();
        EasyMock.replay(redisTemplate, valueOperations);
        RedisDigestStore redisDigestStore = new RedisDigestStore() {
            protected void updateDigest(String key, String value) throws Exception {

            }
        };
        redisDigestStore.setRedis(redisTemplate);
        redisDigestStore.setExpire(10);
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        Digest digest = new Digest(new HashMap<String, Object>(), new ArrayList<String>());
        digest.getDigest().put("KEY1", "VAL1");
        redisDigestStore.upsert(query, "SCENE", digest);
        EasyMock.verify(redisTemplate, valueOperations);
    }

    @Test
    public void testUpsertWithException() throws IOException {
        RedisTemplate redisTemplate = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(redisTemplate.opsForValue()).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(redisTemplate);
        RedisDigestStore redisDigestStore = new RedisDigestStore();
        redisDigestStore.setRedis(redisTemplate);
        redisDigestStore.setExpire(10);
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        Digest digest = new Digest(new HashMap<String, Object>(), new ArrayList<String>());
        digest.getDigest().put("KEY1", "VAL1");
        Assert.assertEquals(digest, redisDigestStore.upsert(query, "SCENE", digest));
        EasyMock.verify(redisTemplate);
    }

    @Test
    public void testInit() throws Exception {
        RedisTemplate<String, Object> redisTemplate = EasyMock.createMock(RedisTemplate.class);
        EasyMock.replay(redisTemplate);
        RedisDigestStore.InitConfig redisDigestStore = new RedisDigestStore.InitConfig();
        redisDigestStore.setRedis(redisTemplate);
        redisDigestStore.setExpire(1000);
        RedisDigestStore empty = (RedisDigestStore) redisDigestStore.digestStore();
        Assert.assertEquals(redisTemplate, empty.getRedis());
        Assert.assertEquals(Integer.valueOf(1000), empty.getExpire());
        EasyMock.verify(redisTemplate);
    }
}
