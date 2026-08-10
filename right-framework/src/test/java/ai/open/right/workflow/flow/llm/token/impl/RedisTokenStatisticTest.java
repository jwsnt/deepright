package ai.open.right.workflow.flow.llm.token.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.google.GoogleRequest;
import ai.open.right.workflow.flow.llm.token.TokenData;
import ai.open.right.workflow.notify.Notifier;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.data.redis.core.RedisCallback;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.ValueOperations;

import java.util.*;

public class RedisTokenStatisticTest {

    @Test
    public void test() throws Exception {
        RedisTemplate r = EasyMock.createMock(RedisTemplate.class);
        GoogleRequest c = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(c, r);
        RedisTokenStatistic redisTokenStatistic = new RedisTokenStatistic() {
            @Override
            protected void stat(String key4thinking, String key4input, String key4token, String key4cache, TokenData tokenData) {
                Assert.assertEquals("rightRedisTokenStatisticUNKNOWN_i", key4thinking);
                Assert.assertEquals("rightRedisTokenStatisticUNKNOWN_p", key4input);
                Assert.assertEquals("rightRedisTokenStatisticUNKNOWN_t", key4token);
                Assert.assertEquals("rightRedisTokenStatisticUNKNOWN_c", key4cache);
                Assert.assertEquals(Integer.valueOf(2), Integer.valueOf(key4token));
                Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(key4cache));
            }
        };
        TokenData tokenData = TokenData.builder()
                .cache(1)
                .total(2)
                .build();
        redisTokenStatistic.setRedis4array(r);
        redisTokenStatistic.stat(c, tokenData);
        EasyMock.verify(c, r);
    }

    @Test
    public void testStat() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisCallback.class))).andReturn(Arrays.asList(1)).anyTimes();
        EasyMock.replay(template);
        RedisTokenStatistic redisTokenStatistic = new RedisTokenStatistic();
        redisTokenStatistic.setRedis4array(template);
        redisTokenStatistic.setExpire(10);
        TokenData tokenData = TokenData.builder()
                .cache(1)
                .total(2)
                .build();
        redisTokenStatistic.stat("C", "D", "A", "B", tokenData);
        EasyMock.verify(template);
    }

    @Test
    public void testStatWithOutToken1() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisCallback.class))).andReturn(Arrays.asList(1)).anyTimes();
        EasyMock.replay(template);
        RedisTokenStatistic redisTokenStatistic = new RedisTokenStatistic();
        redisTokenStatistic.setRedis4array(template);
        redisTokenStatistic.setExpire(10);
        TokenData tokenData = TokenData.builder()
                .cache(null)
                .total(2)
                .build();
        redisTokenStatistic.stat("C", "D", "A", "B", tokenData);
        EasyMock.verify(template);
    }

    @Test
    public void testStatWithOutToken2() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisCallback.class))).andReturn(Arrays.asList(1)).anyTimes();
        EasyMock.replay(template);
        RedisTokenStatistic redisTokenStatistic = new RedisTokenStatistic();
        redisTokenStatistic.setRedis4array(template);
        redisTokenStatistic.setExpire(10);
        TokenData tokenData = TokenData.builder()
                .cache(0)
                .total(2)
                .build();
        redisTokenStatistic.stat("C", "D", "A", "B", tokenData);
        EasyMock.verify(template);
    }

    @Test
    public void testStatWithOutCache1() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisCallback.class))).andReturn(Arrays.asList(1)).anyTimes();
        EasyMock.replay(template);
        RedisTokenStatistic redisTokenStatistic = new RedisTokenStatistic();
        redisTokenStatistic.setRedis4array(template);
        redisTokenStatistic.setExpire(10);
        TokenData tokenData = TokenData.builder()
                .cache(1)
                .total(null)
                .build();
        redisTokenStatistic.stat("C", "D", "A", "B", tokenData);
        EasyMock.verify(template);
    }

    @Test
    public void testStatWithOutCache2() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisCallback.class))).andReturn(Arrays.asList(1)).anyTimes();
        EasyMock.replay(template);
        RedisTokenStatistic redisTokenStatistic = new RedisTokenStatistic();
        redisTokenStatistic.setRedis4array(template);
        redisTokenStatistic.setExpire(10);
        TokenData tokenData = TokenData.builder()
                .cache(1)
                .total(0)
                .build();
        redisTokenStatistic.stat("C", "D", "A", "B", tokenData);
        EasyMock.verify(template);
    }

    @Test
    public void testWithException() throws Exception {
        GoogleRequest c = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(c);
        RedisTokenStatistic redisTokenStatistic = new RedisTokenStatistic() {
            @Override
            protected void stat(String key4thinking, String key4input, String key4token, String key4cache, TokenData tokenData) {
                throw new RuntimeException("ERROR");
            }
        };
        TokenData tokenData = TokenData.builder()
                .cache(1)
                .total(2)
                .build();
        redisTokenStatistic.stat(c, tokenData);
        EasyMock.verify(c);
    }

    @Test
    public void testInit() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        RedisTokenStatistic.InitConfig service = new RedisTokenStatistic.InitConfig();
        service.setRedis4array(template);
        service.setExpire(100);
        RedisTokenStatistic empty = (RedisTokenStatistic) service.tokenStatistic();
        Assert.assertEquals(template, empty.getRedis4array());
        Assert.assertEquals(Integer.valueOf(100), empty.getExpire());
    }

    @Test
    public void testReadAl() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(template.opsForValue()).andReturn(valueOperations).anyTimes();
        EasyMock.expect(valueOperations.multiGet(Arrays.asList("rightRedisTokenStatisticUNKNOWN_t", "rightRedisTokenStatisticUNKNOWN_c"))).andReturn(Arrays.asList("1".getBytes(), "2".getBytes())).anyTimes();
        EasyMock.replay(valueOperations, template);
        RedisTokenStatistic redisTokenStatistic = new RedisTokenStatistic();
        redisTokenStatistic.setRedis4array(template);
        redisTokenStatistic.setExpire(10);
        TokenData tokenData = redisTokenStatistic.read(ObjectBuilder.buildWorkflowTask());
        Assert.assertEquals(Integer.valueOf(1), tokenData.getTotal());
        Assert.assertEquals(Integer.valueOf(2), tokenData.getCache());
        EasyMock.verify(valueOperations, template);
    }

    @Test
    public void testReadAlWithEmpty() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(template.opsForValue()).andReturn(valueOperations).anyTimes();
        EasyMock.expect(valueOperations.multiGet(Arrays.asList("rightRedisTokenStatisticUNKNOWN_t", "rightRedisTokenStatisticUNKNOWN_c"))).andReturn(Arrays.asList()).anyTimes();
        EasyMock.replay(valueOperations, template);
        RedisTokenStatistic redisTokenStatistic = new RedisTokenStatistic();
        redisTokenStatistic.setRedis4array(template);
        redisTokenStatistic.setExpire(10);
        TokenData tokenData = redisTokenStatistic.read(ObjectBuilder.buildWorkflowTask());
        Assert.assertEquals(RedisTokenStatistic.EMPTY, tokenData);
        EasyMock.verify(valueOperations, template);
    }

    @Test
    public void testStatNullRedis() throws Exception {
        RedisTokenStatistic service = new RedisTokenStatistic();
        service.setRedis4array(null);
        service.stat(new ProviderRequest(), TokenData.builder().build()); // Should log error and return
    }

    @Test
    public void testReadAlNullRedis() throws Exception {
        RedisTokenStatistic service = new RedisTokenStatistic();
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ValueOperations ops = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(template.opsForValue()).andReturn(ops).anyTimes();
        EasyMock.expect(ops.multiGet(EasyMock.anyObject())).andReturn(null).anyTimes();
        EasyMock.replay(template, ops);
        service.setRedis4array(template);
        Assert.assertEquals(RedisTokenStatistic.EMPTY, service.read(ObjectBuilder.buildWorkflowTask()));
    }

    /**
     * read(Dimension, String) 在 redis4array 为 null 时抛出 IllegalArgumentException
     */
    @Test
    public void testReadAlDimensionStringNullRedis() throws Exception {
        RedisTokenStatistic service = new RedisTokenStatistic();
        service.setRedis4array(null);
        try {
            service.read(ObjectBuilder.buildWorkflowTask(), "model");
            Assert.fail("expected IllegalArgumentException when redis4array is null");
        } catch (IllegalArgumentException e) {
            Assert.assertTrue("message should mention Redis4array", e.getMessage() != null && e.getMessage().contains("Redis4array"));
        }
    }

    /**
     * read(Dimension, List) 正常返回：多 model 对应多组 token/cache
     */
    @Test
    public void testReadAlDimensionList() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(template.opsForValue()).andReturn(valueOperations).anyTimes();
        List<String> expectedKeys = Arrays.asList(
                "rightRedisTokenStatisticUNKNOWNm1_t", "rightRedisTokenStatisticUNKNOWNm1_c",
                "rightRedisTokenStatisticUNKNOWNm2_t", "rightRedisTokenStatisticUNKNOWNm2_c");
        EasyMock.expect(valueOperations.multiGet(expectedKeys)).andReturn(Arrays.asList(
                "10".getBytes(), "20".getBytes(),
                "30".getBytes(), "40".getBytes())).once();
        EasyMock.replay(valueOperations, template);
        RedisTokenStatistic redisTokenStatistic = new RedisTokenStatistic();
        redisTokenStatistic.setRedis4array(template);
        List<TokenData> result = redisTokenStatistic.readAll(ObjectBuilder.buildWorkflowTask(), Arrays.asList("m1", "m2"));
        EasyMock.verify(valueOperations, template);
        Assert.assertNotNull(result);
        Assert.assertEquals(2, result.size());
        Assert.assertEquals(Integer.valueOf(10), result.get(0).getTotal());
        Assert.assertEquals(Integer.valueOf(20), result.get(0).getCache());
        Assert.assertEquals(Integer.valueOf(30), result.get(1).getTotal());
        Assert.assertEquals(Integer.valueOf(40), result.get(1).getCache());
    }

    /**
     * read(Dimension, List) 当 model 为空列表时返回空列表
     */
    @Test
    public void testReadAlDimensionListEmptyModels() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(template.opsForValue()).andReturn(valueOperations).anyTimes();
        EasyMock.expect(valueOperations.multiGet(Collections.emptyList())).andReturn(Collections.emptyList()).once();
        EasyMock.replay(valueOperations, template);
        RedisTokenStatistic redisTokenStatistic = new RedisTokenStatistic();
        redisTokenStatistic.setRedis4array(template);
        List<TokenData> result = redisTokenStatistic.readAll(ObjectBuilder.buildWorkflowTask(), Collections.emptyList());
        EasyMock.verify(valueOperations, template);
        Assert.assertNotNull(result);
        Assert.assertTrue(result.isEmpty());
    }

    /**
     * read(Dimension, List) 当 multiGet 返回 null 或空时返回空列表
     */
    @Test
    public void testReadAlDimensionListMultiGetEmpty() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(template.opsForValue()).andReturn(valueOperations).anyTimes();
        List<String> expectedKeys = Arrays.asList("rightRedisTokenStatisticUNKNOWNm1_t", "rightRedisTokenStatisticUNKNOWNm1_c");
        EasyMock.expect(valueOperations.multiGet(expectedKeys)).andReturn(null).once();
        EasyMock.replay(valueOperations, template);
        RedisTokenStatistic redisTokenStatistic = new RedisTokenStatistic();
        redisTokenStatistic.setRedis4array(template);
        List<TokenData> result = redisTokenStatistic.readAll(ObjectBuilder.buildWorkflowTask(), Collections.singletonList("m1"));
        EasyMock.verify(valueOperations, template);
        Assert.assertNotNull(result);
        Assert.assertTrue(result.isEmpty());
    }

    /**
     * read(Dimension, List) 中某 key 对应值为 null 时解析为 0
     */
    @Test
    public void testReadAlDimensionListNullValueTreatedAsZero() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(template.opsForValue()).andReturn(valueOperations).anyTimes();
        List<String> expectedKeys = Arrays.asList("rightRedisTokenStatisticUNKNOWNm1_t", "rightRedisTokenStatisticUNKNOWNm1_c");
        EasyMock.expect(valueOperations.multiGet(expectedKeys)).andReturn(Arrays.asList("5".getBytes(), null)).once();
        EasyMock.replay(valueOperations, template);
        RedisTokenStatistic redisTokenStatistic = new RedisTokenStatistic();
        redisTokenStatistic.setRedis4array(template);
        List<TokenData> result = redisTokenStatistic.readAll(ObjectBuilder.buildWorkflowTask(), Collections.singletonList("m1"));
        EasyMock.verify(valueOperations, template);
        Assert.assertEquals(1, result.size());
        Assert.assertEquals(Integer.valueOf(5), result.get(0).getTotal());
        Assert.assertEquals(Integer.valueOf(0), result.get(0).getCache());
    }

    /**
     * read(Dimension, List) 在 redis4array 为 null 时抛出 IllegalArgumentException
     */
    @Test
    public void testReadAlDimensionListNullRedis() throws Exception {
        RedisTokenStatistic service = new RedisTokenStatistic();
        service.setRedis4array(null);
        try {
            service.readAll(ObjectBuilder.buildWorkflowTask(), Arrays.asList("m1", "m2"));
            Assert.fail("expected IllegalArgumentException when redis4array is null");
        } catch (IllegalArgumentException e) {
            Assert.assertTrue("message should mention Redis4array", e.getMessage() != null && e.getMessage().contains("Redis4array"));
        }
    }

    /**
     * stat(key4token, key4cache, tokenData) 在 redis4array 为 null 时抛出 IllegalArgumentException
     */
    @Test
    public void testStatKeysNullRedis() throws Exception {
        RedisTokenStatistic service = new RedisTokenStatistic();
        service.setRedis4array(null);
        TokenData tokenData = TokenData.builder().total(1).cache(2).build();
        try {
            service.stat("C", "D", "A", "B", tokenData);
            Assert.fail("expected IllegalArgumentException when redis4array is null");
        } catch (IllegalArgumentException e) {
            Assert.assertTrue("message should mention Redis4array", e.getMessage() != null && e.getMessage().contains("Redis4array"));
        }
    }

    /**
     * getKey(biz, chat, device, List&lt;String&gt; model) 多 model 返回多个 key，key 格式为 domain+simpleName+device+model，biz/chat 未参与
     */
    @Test
    public void testGetKeyListMultipleModels() throws Exception {
        RedisTokenStatistic service = new RedisTokenStatistic();
        List<String> keys = service.getKey("biz", "chat", "device", Arrays.asList("m1", "m2"));
        Assert.assertEquals(2, keys.size());
        Assert.assertEquals("rightRedisTokenStatisticdevicem1", keys.get(0));
        Assert.assertEquals("rightRedisTokenStatisticdevicem2", keys.get(1));
    }

    /**
     * getKey(biz, chat, device, List) 空 model 列表返回空列表
     */
    @Test
    public void testGetKeyListEmptyModels() throws Exception {
        RedisTokenStatistic service = new RedisTokenStatistic();
        List<String> keys = service.getKey("biz", "chat", "device", Collections.emptyList());
        Assert.assertNotNull(keys);
        Assert.assertTrue(keys.isEmpty());
    }

    /**
     * getKey(biz, chat, device, List) 单 model 返回单个 key
     */
    @Test
    public void testGetKeyListSingleModel() throws Exception {
        RedisTokenStatistic service = new RedisTokenStatistic();
        List<String> keys = service.getKey("b", "c", "d", Collections.singletonList("model"));
        Assert.assertEquals(1, keys.size());
        Assert.assertEquals("rightRedisTokenStatisticdmodel", keys.get(0));
    }

    /**
     * addModel(providerRequest, model) 非空 model 会加入 models 并返回原 model
     */
    @Test
    public void testAddModel_nonEmpty_addsAndReturnsModel() throws Exception {
        RedisTokenStatistic service = new RedisTokenStatistic();
        service.init();
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.replay(request);
        String result = service.addModel(request, "m1");
        EasyMock.verify(request);
        Assert.assertEquals("m1", result);
        Set<String> models = service.models();
        Assert.assertTrue(models.contains("m1"));
        Assert.assertEquals(1, models.size());
    }

    /**
     * addModel(providerRequest, "") 空字符串不加入 models，返回 ""
     */
    @Test
    public void testAddModel_emptyString_doesNotAddReturnsEmpty() throws Exception {
        RedisTokenStatistic service = new RedisTokenStatistic();
        service.init();
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.replay(request);
        String result = service.addModel(request, "");
        EasyMock.verify(request);
        Assert.assertEquals("", result);
        Assert.assertTrue(service.models().isEmpty());
    }

    /**
     * addModel(providerRequest, null) 不加入 models，返回 null
     */
    @Test
    public void testAddModel_null_doesNotAddReturnsNull() throws Exception {
        RedisTokenStatistic service = new RedisTokenStatistic();
        service.init();
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.replay(request);
        String result = service.addModel(request, null);
        EasyMock.verify(request);
        Assert.assertNull(result);
        Assert.assertTrue(service.models().isEmpty());
    }

    /**
     * addModel 多次不同 model，models 包含全部且去重；同一 model 只保留一份
     */
    @Test
    public void testAddModel_multipleModels_andDuplicate() throws Exception {
        RedisTokenStatistic service = new RedisTokenStatistic();
        service.init();
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.replay(request);
        Assert.assertEquals("m1", service.addModel(request, "m1"));
        Assert.assertEquals("m2", service.addModel(request, "m2"));
        Assert.assertEquals("m1", service.addModel(request, "m1"));
        Set<String> models = service.models();
        Assert.assertEquals(2, models.size());
        Assert.assertTrue(models.contains("m1"));
        Assert.assertTrue(models.contains("m2"));
        EasyMock.verify(request);
    }

    /**
     * readAll(Dimension) 使用内部 models 集合，有 model 时委托 readAll(dimension, models) 并返回对应 TokenData 列表
     */
    @Test
    public void testReadAllDimension_withModels_returnsTokenDataList() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(template.opsForValue()).andReturn(valueOperations).anyTimes();
        List<String> expectedKeys = Arrays.asList(
                "rightRedisTokenStatisticUNKNOWNm1_t", "rightRedisTokenStatisticUNKNOWNm1_c",
                "rightRedisTokenStatisticUNKNOWNm2_t", "rightRedisTokenStatisticUNKNOWNm2_c");
        EasyMock.expect(valueOperations.multiGet(expectedKeys)).andReturn(Arrays.asList(
                "10".getBytes(), "20".getBytes(),
                "30".getBytes(), "40".getBytes())).once();
        EasyMock.replay(valueOperations, template);
        RedisTokenStatistic service = new RedisTokenStatistic();
        service.init();
        service.setModels(new LinkedHashSet<>(Arrays.asList("m1", "m2")));
        service.setRedis4array(template);
        List<TokenData> result = service.readAll(ObjectBuilder.buildWorkflowTask());
        EasyMock.verify(valueOperations, template);
        Assert.assertNotNull(result);
        Assert.assertEquals(2, result.size());
        Assert.assertEquals(Integer.valueOf(10), result.get(0).getTotal());
        Assert.assertEquals(Integer.valueOf(20), result.get(0).getCache());
        Assert.assertEquals(Integer.valueOf(30), result.get(1).getTotal());
        Assert.assertEquals(Integer.valueOf(40), result.get(1).getCache());
    }

    /**
     * readAll(Dimension) 当 models 为空时委托 readAll(dimension, emptyList)，返回空列表
     */
    @Test
    public void testReadAllDimension_emptyModels_returnsEmptyList() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ValueOperations valueOperations = EasyMock.createMock(ValueOperations.class);
        EasyMock.expect(template.opsForValue()).andReturn(valueOperations).anyTimes();
        EasyMock.expect(valueOperations.multiGet(Collections.emptyList())).andReturn(Collections.emptyList()).once();
        EasyMock.replay(valueOperations, template);
        RedisTokenStatistic service = new RedisTokenStatistic();
        service.init();
        service.setRedis4array(template);
        List<TokenData> result = service.readAll(ObjectBuilder.buildWorkflowTask());
        EasyMock.verify(valueOperations, template);
        Assert.assertNotNull(result);
        Assert.assertTrue(result.isEmpty());
    }

    /**
     * readAll(Dimension) 当 redis4array 为 null 时抛出 IllegalArgumentException
     */
    @Test
    public void testReadAllDimension_nullRedis_throws() throws Exception {
        RedisTokenStatistic service = new RedisTokenStatistic();
        service.init();
        service.setModels(new HashSet<>(Collections.singletonList("m1")));
        service.setRedis4array(null);
        try {
            service.readAll(ObjectBuilder.buildWorkflowTask());
            Assert.fail("expected IllegalArgumentException when redis4array is null");
        } catch (IllegalArgumentException e) {
            Assert.assertTrue("message should mention Redis4array", e.getMessage() != null && e.getMessage().contains("Redis4array"));
        }
    }
}
