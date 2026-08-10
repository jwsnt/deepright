package ai.open.right.workflow.flow.llm.store.history.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.utils.GzipUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestModel;
import ai.open.right.workflow.flow.llm.store.Dimension;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.data.redis.connection.RedisConnection;
import org.springframework.data.redis.core.RedisCallback;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.ZSetOperations;

import java.util.*;
import java.util.concurrent.atomic.AtomicReference;

import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

public class RedisHistoryStoreTest {

    @Test
    public void testGetVal_roundTripPreservesModelAndApi() throws Exception {
        RedisHistoryStore store = new RedisHistoryStore();
        HistoryPair p = new HistoryPair();
        p.setQuery("q");
        p.setAnswer("a");
        p.setModel("m-z");
        p.setApi("api-z");
        Dimension dim = EasyMock.createMock(Dimension.class);
        EasyMock.replay(dim);
        HistoryPair back = JsonUtils.read(GzipUtils.decompress(store.getVal(dim, p)), HistoryPair.class);
        Assert.assertEquals("m-z", back.getModel());
        Assert.assertEquals("api-z", back.getApi());
    }

    @Test
    public void testStore_queryAnswerReasoning_setsDefaultModelApi() throws Exception {
        Dimension dim = EasyMock.createMock(Dimension.class);
        EasyMock.replay(dim);
        AtomicReference<HistoryPair> ref = new AtomicReference<>();
        RedisHistoryStore store = new RedisHistoryStore() {
            @Override
            public void store(Dimension dimension, List<String> repositories, HistoryPair pair, Integer expire, Integer nums) throws Exception {
                ref.set(pair);
            }
        };
        store.store(dim, Collections.singletonList("r"), "q", "a", "rsn", 100, 10, 1L);
        HistoryPair captured = ref.get();
        Assert.assertNotNull(captured);
        Assert.assertEquals(ProviderRequestModel.DEF, captured.getModel());
        Assert.assertEquals(ProviderRequest.REQUEST_DEF, captured.getApi());
    }

    @Test
    public void testStore_queryAnswer_setsDefaultModelApi() throws Exception {
        Dimension dim = EasyMock.createMock(Dimension.class);
        EasyMock.replay(dim);
        AtomicReference<HistoryPair> ref = new AtomicReference<>();
        RedisHistoryStore store = new RedisHistoryStore() {
            @Override
            public void store(Dimension dimension, List<String> repositories, HistoryPair pair, Integer expire, Integer nums) throws Exception {
                ref.set(pair);
            }
        };
        store.store(dim, Collections.singletonList("r"), "q", "a", 100, 10, 2L);
        HistoryPair captured = ref.get();
        Assert.assertNotNull(captured);
        Assert.assertEquals(ProviderRequestModel.DEF, captured.getModel());
        Assert.assertEquals(ProviderRequest.REQUEST_DEF, captured.getApi());
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = RedisHistoryStore.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = RedisHistoryStore.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testRestoreAndNow() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        Set<Object> members = new HashSet<Object>();
        HistoryPair pair = new HistoryPair();
        pair.setAnswer("Answer");
        pair.setQuery("Query");
        members.add(GzipUtils.compress(JsonUtils.write(pair)));
        EasyMock.expect(zset.rangeByScore(EasyMock.anyString(), EasyMock.anyInt(), EasyMock.anyDouble(), EasyMock.anyInt(), EasyMock.anyInt())).andReturn(members).anyTimes();
        EasyMock.replay(zset, template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setExpire(10);
        store.setMaxsize(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        List<History> histories = store.restore(message, "WORKFLOW", 20, 10086L);
        EasyMock.verify(zset, template);
        Assert.assertEquals(2, histories.size());
        Assert.assertEquals("Query", histories.get(0).getContent());
        Assert.assertEquals("Answer", histories.get(1).getContent());
        Assert.assertEquals(History.ROLE_USER, histories.get(0).getRole());
        Assert.assertEquals(History.ROLE_ASSISTANT, histories.get(1).getRole());
        Assert.assertEquals(History.TYPE_QUERY, histories.get(0).getType());
        Assert.assertEquals(History.TYPE_ANSWER, histories.get(1).getType());
    }

    @Test
    public void testRestoreDesc() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        Set<Object> members = new HashSet<Object>();
        HistoryPair pair = new HistoryPair();
        pair.setAnswer("Answer");
        pair.setQuery("Query");
        members.add(GzipUtils.compress(JsonUtils.write(pair)));
        EasyMock.expect(zset.range(EasyMock.anyString(), EasyMock.anyInt(), EasyMock.anyInt())).andReturn(members).anyTimes();
        EasyMock.replay(zset, template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setExpire(10);
        store.setMaxsize(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        List<History> histories = store.restore(message, "WORKFLOW", 20, false);
        EasyMock.verify(zset, template);
        Assert.assertEquals(2, histories.size());
        Assert.assertEquals("Query", histories.get(0).getContent());
        Assert.assertEquals("Answer", histories.get(1).getContent());
        Assert.assertEquals(History.ROLE_USER, histories.get(0).getRole());
        Assert.assertEquals(History.ROLE_ASSISTANT, histories.get(1).getRole());
        Assert.assertEquals(History.TYPE_QUERY, histories.get(0).getType());
        Assert.assertEquals(History.TYPE_ANSWER, histories.get(1).getType());
    }

    @Test
    public void testRestoreNotDesc() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        Set<Object> members = new HashSet<Object>();
        HistoryPair pair = new HistoryPair();
        pair.setAnswer("Answer");
        pair.setQuery("Query");
        members.add(GzipUtils.compress(JsonUtils.write(pair)));
        EasyMock.expect(zset.reverseRange(EasyMock.anyString(), EasyMock.anyInt(), EasyMock.anyInt())).andReturn(members).anyTimes();
        EasyMock.replay(zset, template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setExpire(10);
        store.setMaxsize(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        List<History> histories = store.restore(message, "WORKFLOW", 20, true);
        EasyMock.verify(zset, template);
        Assert.assertEquals(2, histories.size());
        Assert.assertEquals("Query", histories.get(0).getContent());
        Assert.assertEquals("Answer", histories.get(1).getContent());
        Assert.assertEquals(History.ROLE_USER, histories.get(0).getRole());
        Assert.assertEquals(History.ROLE_ASSISTANT, histories.get(1).getRole());
        Assert.assertEquals(History.TYPE_QUERY, histories.get(0).getType());
        Assert.assertEquals(History.TYPE_ANSWER, histories.get(1).getType());
    }

    @Test
    public void testRestoreAndNowAndDesc() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        Set<Object> members = new HashSet<Object>();
        HistoryPair pair = new HistoryPair();
        pair.setAnswer("Answer");
        pair.setQuery("Query");
        members.add(GzipUtils.compress(JsonUtils.write(pair)));
        EasyMock.expect(zset.reverseRangeByScore(EasyMock.anyString(), EasyMock.anyDouble(), EasyMock.anyInt(), EasyMock.anyInt(), EasyMock.anyInt())).andReturn(members).anyTimes();
        EasyMock.replay(zset, template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setExpire(10);
        store.setMaxsize(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        List<History> histories = store.restore(message, "WORKFLOW", 20, true, 10086L);
        EasyMock.verify(zset, template);
        Assert.assertEquals(2, histories.size());
        Assert.assertEquals("Query", histories.get(0).getContent());
        Assert.assertEquals("Answer", histories.get(1).getContent());
        Assert.assertEquals(History.ROLE_USER, histories.get(0).getRole());
        Assert.assertEquals(History.ROLE_ASSISTANT, histories.get(1).getRole());
        Assert.assertEquals(History.TYPE_QUERY, histories.get(0).getType());
        Assert.assertEquals(History.TYPE_ANSWER, histories.get(1).getType());
    }

    @Test
    public void testRestoreAndSort() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        Set<Object> members = new HashSet<Object>();
        HistoryPair pair = new HistoryPair();
        pair.setAnswer("Answer");
        pair.setQuery("Query");
        members.add(GzipUtils.compress(JsonUtils.write(pair)));
        EasyMock.expect(zset.range(EasyMock.anyString(), EasyMock.anyInt(), EasyMock.anyInt())).andReturn(members).anyTimes();
        EasyMock.replay(zset, template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setExpire(10);
        store.setMaxsize(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        List<History> histories = store.restore(message, "WORKFLOW", 20);
        EasyMock.verify(zset, template);
        Assert.assertEquals(2, histories.size());
        Assert.assertEquals("Query", histories.get(0).getContent());
        Assert.assertEquals("Answer", histories.get(1).getContent());
        Assert.assertEquals(History.ROLE_USER, histories.get(0).getRole());
        Assert.assertEquals(History.ROLE_ASSISTANT, histories.get(1).getRole());
        Assert.assertEquals(History.TYPE_QUERY, histories.get(0).getType());
        Assert.assertEquals(History.TYPE_ANSWER, histories.get(1).getType());
    }

    @Test
    public void testRestoreAndFilter() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        Set<Object> members = new HashSet<Object>();
        HistoryPair pair = new HistoryPair();
        pair.setAnswer("Answer");
        pair.setQuery("Query");
        members.add(GzipUtils.compress(JsonUtils.write(pair)));
        EasyMock.expect(zset.range(EasyMock.anyString(), EasyMock.anyInt(), EasyMock.anyInt())).andReturn(members).anyTimes();
        EasyMock.replay(zset, template);
        RedisHistoryStore store = new RedisHistoryStore() {
            public HistoryPair restore(Dimension dimension, HistoryPair historyPair) {
                return null;
            }
        };
        store.setRedis4array(template);
        store.setExpire(10);
        store.setMaxsize(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        List<History> histories = store.restore(message, "WORKFLOW", 20);
        Assert.assertTrue(histories.isEmpty());
        EasyMock.verify(zset, template);
    }

    @Test
    public void testRestoreWithEmpty() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        Set<Object> members = new HashSet<Object>();
        EasyMock.expect(zset.range(EasyMock.anyString(), EasyMock.anyInt(), EasyMock.anyInt())).andReturn(members).anyTimes();
        EasyMock.replay(zset, template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setMaxsize(10);
        store.setExpire(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        List<History> histories = store.restore(message, "WORKFLOW", 20);
        EasyMock.verify(zset, template);
        Assert.assertEquals(0, histories.size());
    }

    @Test
    public void testRestoreWithInValid() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        Set<Object> members = new HashSet<Object>();
        members.add(new byte[]{1, 2, 3});
        EasyMock.expect(zset.range(EasyMock.anyString(), EasyMock.anyInt(), EasyMock.anyInt())).andReturn(members).anyTimes();
        EasyMock.replay(zset, template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setMaxsize(10);
        store.setExpire(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        List<History> histories = store.restore(message, "WORKFLOW", 20);
        EasyMock.verify(zset, template);
        Assert.assertEquals(0, histories.size());
    }

    @Test
    public void testRestoreWithNull() throws Exception {
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        Set<Object> members = new HashSet<Object>();
        EasyMock.expect(zset.range(EasyMock.anyString(), EasyMock.anyInt(), EasyMock.anyInt())).andReturn(members).anyTimes();
        EasyMock.replay(zset);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setMaxsize(10);
        store.setExpire(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        List<History> histories = store.restore(message, "WORKFLOW", 20);
        EasyMock.verify(zset);
        Assert.assertEquals(0, histories.size());
    }

    @Test
    public void testStore() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisCallback.class))).andReturn(null).anyTimes();
        RedisHistoryStore store = new RedisHistoryStore();
        store.setMaxsize(10);
        store.setExpire(10);
        store.setRedis4array(template);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        EasyMock.replay(template);
        store.store(message, Arrays.asList("WORKFLOW"), "QUERY", "ANSWER", 1000, 10, 100L);
        EasyMock.verify(template);
    }

    @Test
    public void testStoreAndRewriterNull() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisCallback.class))).andReturn(null).anyTimes();
        RedisHistoryStore store = new RedisHistoryStore() {
            public HistoryPair store(Dimension dimension, HistoryPair historyPair) throws Exception {
                return null;
            }
        };
        store.setMaxsize(10);
        store.setExpire(10);
        store.setRedis4array(template);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        EasyMock.replay(template);
        store.store(message, Arrays.asList("WORKFLOW"), "QUERY", "ANSWER", 1000, 10, 100L);
        EasyMock.verify(template);
    }

    @Test
    public void testStoreWithPair() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisCallback.class))).andReturn(null).anyTimes();
        RedisHistoryStore store = new RedisHistoryStore();
        store.setMaxsize(10);
        store.setExpire(10);
        store.setRedis4array(template);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        EasyMock.replay(template);
        List<HistoryPair> pairs = new ArrayList<>();
        HistoryPair pair = new HistoryPair();
        pair.setQuery("QUERY");
        pair.setAnswer("ANSWER");
        pair.setCreated(100L);
        pairs.add(pair);
        store.store(message, Arrays.asList("WORKFLOW"), pairs, 1000, 10);
        EasyMock.verify(template);
    }

    @Test
    public void testStoreWithPairAndRewriterNull() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisCallback.class))).andReturn(null).anyTimes();
        RedisHistoryStore store = new RedisHistoryStore() {
            public HistoryPair store(Dimension dimension, HistoryPair historyPair) throws Exception {
                return null;
            }
        };
        store.setMaxsize(10);
        store.setExpire(10);
        store.setRedis4array(template);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        EasyMock.replay(template);
        List<HistoryPair> pairs = new ArrayList<>();
        HistoryPair pair = new HistoryPair();
        pair.setQuery("QUERY");
        pair.setAnswer("ANSWER");
        pair.setCreated(100L);
        pairs.add(pair);
        store.store(message, Arrays.asList("WORKFLOW"), pairs, 1000, 10);
        EasyMock.verify(template);
    }

    @Test
    public void testStoreWithNullPointException() throws Exception {
        RedisHistoryStore store = new RedisHistoryStore();
        store.setMaxsize(10);
        store.setExpire(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        List<HistoryPair> pairs = new ArrayList<>();
        HistoryPair pair = new HistoryPair();
        pair.setQuery("QUERY");
        pair.setAnswer("ANSWER");
        pair.setCreated(100L);
        pairs.add(pair);
        store.store(message, Arrays.asList("WORKFLOW"), pairs, 1000, 10);
    }

    @Test
    public void testStoreWithPairAndNullPointException1() throws Exception {
        RedisHistoryStore store = new RedisHistoryStore();
        store.setMaxsize(10);
        store.setExpire(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        store.store(message, Arrays.asList("WORKFLOW"), "QUERY", "ANSWER", 1000, 10, 100L);
    }

    @Test
    public void testStoreWithPairAndNullPointException2() throws Exception {
        RedisHistoryStore store = new RedisHistoryStore();
        store.setMaxsize(10);
        store.setExpire(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        List<HistoryPair> pairs = new ArrayList<>();
        HistoryPair pair = new HistoryPair();
        pair.setQuery("QUERY");
        pair.setAnswer("ANSWER");
        pair.setCreated(100L);
        pairs.add(pair);
        store.store(message, Arrays.asList("WORKFLOW"), pairs, 1000, 10);
    }

    @Test
    public void testStoreWithEmptyAnswer() throws Exception {
        RedisHistoryStore store = new RedisHistoryStore();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        store.store(message, Arrays.asList("WORKFLOW"), "QUERY", "", 1000, 10, 100L);
    }

    @Test
    public void testStoreWithPairAndEmptyAnswer() throws Exception {
        RedisHistoryStore store = new RedisHistoryStore();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        List<HistoryPair> pairs = new ArrayList<>();
        HistoryPair pair = new HistoryPair();
        pair.setQuery("QUERY");
        pair.setAnswer("");
        pair.setCreated(100L);
        pairs.add(pair);
        store.store(message, Arrays.asList("WORKFLOW"), pairs, 1000, 10);
    }

    @Test
    public void testStoreWithEmptyQuery() throws Exception {
        RedisHistoryStore store = new RedisHistoryStore();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        store.store(message, Arrays.asList("WORKFLOW"), "", "ANSWER", 1000, 10, 100L);
    }

    @Test
    public void testStoreWithPairAndEmptyQuery() throws Exception {
        RedisHistoryStore store = new RedisHistoryStore();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        List<HistoryPair> pairs = new ArrayList<>();
        HistoryPair pair = new HistoryPair();
        pair.setQuery("");
        pair.setAnswer("ANSWER");
        pair.setCreated(100L);
        pairs.add(pair);
        store.store(message, Arrays.asList("WORKFLOW"), pairs, 1000, 10);
    }

    @Test
    public void testClear() throws Exception {
        RedisConnection connection = EasyMock.createMock(RedisConnection.class);
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisCallback.class))).andReturn(null).anyTimes();
        EasyMock.replay(connection, template);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.clear(message, Arrays.asList("WORKFLOW"), 10086L);
        EasyMock.verify(connection, template);
    }

    @Test
    public void testClearWithException() throws Exception {
        RedisConnection connection = EasyMock.createMock(RedisConnection.class);
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisCallback.class))).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(connection, template);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.clear(message, Arrays.asList("WORKFLOW"), 10086L);
        EasyMock.verify(connection, template);
    }

    @Test
    public void testInit() throws Exception {
        RedisTemplate<String, Object> redisTemplate = EasyMock.createMock(RedisTemplate.class);
        EasyMock.replay(redisTemplate);
        RedisHistoryStore.InitConfig redisHistoryStore = new RedisHistoryStore.InitConfig();
        redisHistoryStore.setRedis4array(redisTemplate);
        redisHistoryStore.setMaxsize(100);
        redisHistoryStore.setExpire(1000);
        RedisHistoryStore empty = (RedisHistoryStore) redisHistoryStore.historyStore();
        Assert.assertEquals(redisTemplate, empty.getRedis4array());
        Assert.assertEquals(Integer.valueOf(1000), empty.getExpire());
        Assert.assertEquals(Integer.valueOf(100), empty.getMaxsize());
        EasyMock.verify(redisTemplate);
    }

    @Test
    public void testStoreWithPairAndNow() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisCallback.class))).andReturn(null).anyTimes();
        RedisHistoryStore store = new RedisHistoryStore() {
            public void store(Dimension dimension, List<String> repositories, HistoryPair pair, Integer expire, Integer nums) throws Exception {
                Assert.assertEquals("ANSWER", pair.getAnswer());
                Assert.assertEquals("QUERY", pair.getQuery());
                Assert.assertEquals(Long.valueOf(100L), Long.valueOf(pair.getCreated()));
            }
        };
        store.setMaxsize(10);
        store.setExpire(10);
        store.setRedis4array(template);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        EasyMock.replay(template);
        List<HistoryPair> pairs = new ArrayList<>();
        HistoryPair pair = new HistoryPair();
        pair.setReasoning("REASONING");
        pair.setQuery("QUERY");
        pair.setAnswer("ANSWER");
        pair.setCreated(100L);
        pairs.add(pair);
        store.store(message, Arrays.asList("WORKFLOW"), "QUERY", "ANSWER", "REASONING", 100, 1000, 100L);
        EasyMock.verify(template);
    }

    @Test
    public void testRestoreLimit() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        EasyMock.expect(zset.range(EasyMock.anyString(), EasyMock.anyLong(), EasyMock.anyLong())).andReturn(Collections.emptySet()).anyTimes();
        EasyMock.replay(zset, template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setMaxsize(5);
        store.restore(ObjectBuilder.buildWorkflowTask(), "SCENE", 10); // nums 10 > maxsize 5
        EasyMock.verify(zset, template);
    }

    @Test
    public void testStoreEmptyPairs() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.replay(template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        List<HistoryPair> pairs = new ArrayList<>();
        pairs.add(new HistoryPair()); // Both query and answer empty
        store.store(ObjectBuilder.buildWorkflowTask(), Arrays.asList("REPO"), pairs, 100, 10);
        EasyMock.verify(template); // executePipelined should not be called
    }

    @Test
    public void testGetKey() throws Exception {
        RedisHistoryStore store = new RedisHistoryStore();
        Dimension dim = EasyMock.createMock(Dimension.class);
        EasyMock.expect(dim.getBiz()).andReturn("BIZ").anyTimes();
        EasyMock.expect(dim.getChat()).andReturn("CHAT").anyTimes();
        EasyMock.expect(dim.getDevice()).andReturn("DEV").anyTimes();
        EasyMock.replay(dim);
        String key = store.getKey(dim, "BIZ@SCENE");
        Assert.assertTrue(key.contains("SCENE"));
        Assert.assertTrue(key.contains("CHAT"));
        Assert.assertTrue(key.contains("DEV"));
    }

    @Test
    public void testRedisStoreCallback() {
        List<byte[]> keys = Arrays.asList("K1".getBytes());
        byte[] history = "H1".getBytes();
        RedisHistoryStore.RedisStoreCallback callback = new RedisHistoryStore.RedisStoreCallback(keys, history, 100, 10, 1000L);
        Assert.assertNotNull(callback);
    }

    @Test
    public void testRedisClearCallback() {
        List<byte[]> keys = Arrays.asList("K1".getBytes());
        RedisHistoryStore.RedisClearCallback callback = new RedisHistoryStore.RedisClearCallback(keys, false, 1000L);
        Assert.assertNotNull(callback);
    }

    // ---------- 边界测试 ----------

    /** restore：nums=0 时 limit=0，返回空列表且为 EMPTY 常量。 */
    @Test
    public void restore_numsZero_returnsEmpty() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        EasyMock.expect(zset.range(EasyMock.anyString(), EasyMock.eq(0L), EasyMock.eq(0L))).andReturn(Collections.emptySet()).once();
        EasyMock.replay(zset, template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setMaxsize(10);
        store.setExpire(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        List<History> result = store.restore(message, "WORKFLOW", 0);
        Assert.assertNotNull(result);
        Assert.assertTrue(result.isEmpty());
        EasyMock.verify(zset, template);
    }

    /** restore：Redis 抛异常时返回 EMPTY，不向上抛。 */
    @Test
    public void restore_throws_returnsEmpty() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andThrow(new RuntimeException("redis down")).once();
        EasyMock.replay(zset, template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setMaxsize(10);
        store.setExpire(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        List<History> result = store.restore(message, "WORKFLOW", 5);
        Assert.assertNotNull(result);
        Assert.assertTrue(result.isEmpty());
        EasyMock.verify(zset, template);
    }

    /** restore：pairData 为空时返回空列表。 */
    @Test
    public void restore_emptySet_returnsEmptyConstant() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        EasyMock.expect(zset.range(EasyMock.anyString(), EasyMock.anyInt(), EasyMock.anyInt())).andReturn(Collections.emptySet()).once();
        EasyMock.replay(zset, template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setMaxsize(10);
        store.setExpire(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        List<History> result = store.restore(message, "WORKFLOW", 10);
        Assert.assertNotNull(result);
        Assert.assertTrue(result.isEmpty());
        EasyMock.verify(zset, template);
    }

    /** store：pairs 为空列表时不调用 executePipelined。 */
    @Test
    public void store_emptyPairsList_doesNotCallExecutePipelined() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.replay(template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setMaxsize(10);
        store.setExpire(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        store.store(message, Arrays.asList("WORKFLOW"), Collections.emptyList(), 1000, 10);
        EasyMock.verify(template);
    }

    /** store：repositories 为空列表时不抛异常。 */
    @Test
    public void store_emptyRepositories_noException() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisCallback.class))).andReturn(Collections.emptyList()).once();
        EasyMock.replay(template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setMaxsize(10);
        store.setExpire(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        List<HistoryPair> pairs = new ArrayList<>();
        HistoryPair pair = new HistoryPair();
        pair.setQuery("Q");
        pair.setAnswer("A");
        pair.setCreated(100L);
        pairs.add(pair);
        store.store(message, Collections.emptyList(), pairs, 1000, 10);
        EasyMock.verify(template);
    }

    /** clear：repositories 为空列表时不抛异常。 */
    @Test
    public void clear_emptyRepositories_noException() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisCallback.class))).andReturn(Collections.emptyList()).once();
        EasyMock.replay(template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        store.clear(message, Collections.emptyList(), true, 1000L);
        EasyMock.verify(template);
    }

    /** clear：now=null 时走 del 分支，不抛异常。 */
    @Test
    public void clear_nowNull_noException() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisCallback.class))).andReturn(Collections.emptyList()).once();
        EasyMock.replay(template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        store.clear(message, Arrays.asList("WORKFLOW"), true, null);
        EasyMock.verify(template);
    }

    /** restore：nums 为 null 时可能 NPE 或由调用方保证；此处测 nums 为正，maxsize 为 0 的边界。 */
    @Test
    public void restore_maxsizeZero_limitZero() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        EasyMock.expect(zset.range(EasyMock.anyString(), EasyMock.eq(0L), EasyMock.eq(0L))).andReturn(Collections.emptySet()).once();
        EasyMock.replay(zset, template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setMaxsize(0);
        store.setExpire(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        List<History> result = store.restore(message, "WORKFLOW", 10);
        Assert.assertNotNull(result);
        Assert.assertTrue(result.isEmpty());
        EasyMock.verify(zset, template);
    }

    // ---------- restore 逻辑验证：desc/asc 分支及 now 参数 ----------

    /** restore(desc=true, now!=null)：验证调用 reverseRangeByScore 且参数为 (now+1, 0]，排除 now 边界。
     * now 参数为负数（score），如当前时间戳 100000 → now = -100000 */
    @Test
    public void restore_descWithNow_usesReverseRangeByScoreWithNowPlusOne() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        Set<Object> members = new HashSet<>();
        HistoryPair pair = new HistoryPair();
        pair.setAnswer("A");
        pair.setQuery("Q");
        members.add(GzipUtils.compress(JsonUtils.write(pair)));
        Long now = -100000L; // now 为负数（score），对应时间戳 100000
        // 验证：reverseRangeByScore(key, now+1, 0, 0, limit) - 开区间 (now, 0]
        EasyMock.expect(zset.reverseRangeByScore(
                EasyMock.anyString(),
                EasyMock.eq((double) (now + 1)), // now+1 = -99999，实现开区间 (now, 0]
                EasyMock.eq(0.0),
                EasyMock.eq(0L),
                EasyMock.anyInt()
        )).andReturn(members).once();
        EasyMock.replay(zset, template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setExpire(10);
        store.setMaxsize(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        store.restore(message, "WORKFLOW", 20, true, now); // now 为负数（score）
        EasyMock.verify(zset, template);
    }

    /** restore(desc=false, now!=null)：验证调用 rangeByScore 且参数为 (now+1, 0]，排除 now 边界。
     * now 参数为负数（score），如当前时间戳 100000 → now = -100000 */
    @Test
    public void restore_ascWithNow_usesRangeByScoreWithNowPlusOne() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        Set<Object> members = new HashSet<>();
        HistoryPair pair = new HistoryPair();
        pair.setAnswer("A");
        pair.setQuery("Q");
        members.add(GzipUtils.compress(JsonUtils.write(pair)));
        Long now = -100000L; // now 为负数（score），对应时间戳 100000
        // 验证：rangeByScore(key, now+1, 0, 0, limit) - 开区间 (now, 0]
        EasyMock.expect(zset.rangeByScore(
                EasyMock.anyString(),
                EasyMock.eq((double) (now + 1)), // now+1 = -99999，实现开区间 (now, 0]
                EasyMock.eq(0.0),
                EasyMock.eq(0L),
                EasyMock.anyInt()
        )).andReturn(members).once();
        EasyMock.replay(zset, template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setExpire(10);
        store.setMaxsize(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        store.restore(message, "WORKFLOW", 20, false, now); // now 为负数（score）
        EasyMock.verify(zset, template);
    }

    /**
     * 当前时刻 T（毫秒）、now=-T、end=-(T-90s)、desc=false：按 score 区间取过去约 90 秒到「当前」一侧的数据。
     * 实现为 rangeByScore(key, now+1, end)，即 [-(T-1), -(T-90000)]，左开 (now, end] 语义。
     * 不 mock RedisTemplate（JDK 25 下 ByteBuddy/Mockito 对 final 类受限），用匿名类委托 opsForZSet。
     */
    @Test
    @SuppressWarnings("unchecked")
    public void restore_ascWithNowAndEnd_past90SecondsWindow_usesRangeByScore() throws Exception {
        ZSetOperations<String, Object> zset = mock(ZSetOperations.class);
        RedisTemplate<String, Object> template = new RedisTemplate<String, Object>() {
            @Override
            public ZSetOperations<String, Object> opsForZSet() {
                return zset;
            }
        };
        Set<Object> members = new HashSet<>();
        HistoryPair pair = new HistoryPair();
        pair.setAnswer("A");
        pair.setQuery("Q");
        members.add(GzipUtils.compress(JsonUtils.write(pair)));
        long T = 1_700_000_000_000L;
        long ninetySecondsMs = 90_000L;
        Long now = -T;
        Long end = -(T - ninetySecondsMs);
        long limit = Math.min(20, 10);
        when(zset.rangeByScore(
                anyString(),
                eq((double) (now + 1)),
                eq(end.doubleValue()),
                eq(0L),
                eq(limit)
        )).thenReturn(members);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setExpire(10);
        store.setMaxsize(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        store.restore(message, "WORKFLOW", 20, false, now, end);
        verify(zset).rangeByScore(
                anyString(),
                eq((double) (now + 1)),
                eq(end.doubleValue()),
                eq(0L),
                eq(limit)
        );
    }

    /** restore(desc=true, now=null)：验证调用 reverseRange，不传 now 参数。 */
    @Test
    public void restore_descWithoutNow_usesReverseRange() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        Set<Object> members = new HashSet<>();
        HistoryPair pair = new HistoryPair();
        pair.setAnswer("A");
        pair.setQuery("Q");
        members.add(GzipUtils.compress(JsonUtils.write(pair)));
        EasyMock.expect(zset.reverseRange(
                EasyMock.anyString(),
                EasyMock.eq(0L),
                EasyMock.anyInt()
        )).andReturn(members).once();
        EasyMock.replay(zset, template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setExpire(10);
        store.setMaxsize(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        store.restore(message, "WORKFLOW", 20, true, null);
        EasyMock.verify(zset, template);
    }

    /** restore(desc=false, now=null)：验证调用 range，不传 now 参数。 */
    @Test
    public void restore_ascWithoutNow_usesRange() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        Set<Object> members = new HashSet<>();
        HistoryPair pair = new HistoryPair();
        pair.setAnswer("A");
        pair.setQuery("Q");
        members.add(GzipUtils.compress(JsonUtils.write(pair)));
        EasyMock.expect(zset.range(
                EasyMock.anyString(),
                EasyMock.eq(0L),
                EasyMock.anyInt()
        )).andReturn(members).once();
        EasyMock.replay(zset, template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setExpire(10);
        store.setMaxsize(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        store.restore(message, "WORKFLOW", 20, false, null);
        EasyMock.verify(zset, template);
    }

    /** restore：验证 limit = min(nums, maxsize) 的逻辑。range(key, start, end) 参数为 long。 */
    @Test
    public void restore_limitIsMinOfNumsAndMaxsize() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        Set<Object> members = new HashSet<>();
        EasyMock.expect(zset.range(
                EasyMock.anyString(),
                EasyMock.eq(0L),
                EasyMock.eq(5L) // nums=10, maxsize=5, limit=min(10,5)=5；range 第三参为 long
        )).andReturn(members).once();
        EasyMock.replay(zset, template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setExpire(10);
        store.setMaxsize(5); // maxsize=5
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        store.restore(message, "WORKFLOW", 10, false, null); // nums=10
        EasyMock.verify(zset, template);
    }

    /** restore：验证 now 为边界值（如 -1, 0）时的处理。now 为负数（score）。 */
    @Test
    public void restore_nowBoundaryValue_handlesCorrectly() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ZSetOperations zset = EasyMock.createMock(ZSetOperations.class);
        EasyMock.expect(template.opsForZSet()).andReturn(zset).anyTimes();
        Set<Object> members = new HashSet<>();
        HistoryPair pair = new HistoryPair();
        pair.setAnswer("A");
        pair.setQuery("Q");
        members.add(GzipUtils.compress(JsonUtils.write(pair)));
        Long now = -1L; // now 为负数（score），now+1 = 0
        // 验证：区间 (now, 0] = (-1, 0]，now+1 = 0
        EasyMock.expect(zset.reverseRangeByScore(
                EasyMock.anyString(),
                EasyMock.eq(0.0), // now+1 = -1+1 = 0
                EasyMock.eq(0.0),
                EasyMock.eq(0L),
                EasyMock.anyInt()
        )).andReturn(members).once();
        EasyMock.replay(zset, template);
        RedisHistoryStore store = new RedisHistoryStore();
        store.setRedis4array(template);
        store.setExpire(10);
        store.setMaxsize(10);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        store.restore(message, "WORKFLOW", 20, true, now); // now 为负数（score）
        EasyMock.verify(zset, template);
    }
}
