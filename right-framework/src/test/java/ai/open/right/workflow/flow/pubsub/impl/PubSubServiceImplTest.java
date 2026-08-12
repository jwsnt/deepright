package ai.open.right.workflow.flow.pubsub.impl;

import ai.open.right.workflow.flow.pubsub.PubSubService;
import ai.open.right.ObjectBuilder;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.SegmentDelegate;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.pubsub.PubSubConfig;
import ai.open.right.workflow.flow.pubsub.PubSubFormatter;
import ai.open.right.workflow.flow.pubsub.PubSubQuery;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import com.fasterxml.jackson.core.JsonParseException;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.data.redis.core.RedisTemplate;

import java.util.*;

public class PubSubServiceImplTest {

    @Test
    public void testKey() throws Exception {
        PubSubServiceImpl pubSubService = new PubSubServiceImpl();
        Assert.assertEquals(Integer.valueOf(36), Integer.valueOf(pubSubService.key().length()));
    }

    @Test
    public void testPubSubQuery() throws Exception {
        PubSubQuery query = new PubSubQuery();
        query.setAnswer("Answer");
        query.setQuery("Query");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(JsonUtils.write(query));
        PubSubServiceImpl pubSubService = new PubSubServiceImpl();
        PubSubQuery result = pubSubService.pubSubQuery(workflowTask);
        Assert.assertEquals(query.getQuery(), result.getQuery());
        Assert.assertEquals(query.getAnswer(), result.getAnswer());
    }

    @Test(expected = JsonParseException.class)
    public void testPubSubQueryWithInvalidJson() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("\\{");
        PubSubServiceImpl pubSubService = new PubSubServiceImpl();
        pubSubService.pubSubQuery(workflowTask);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testPubSubQueryWithEmpty() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("");
        PubSubServiceImpl pubSubService = new PubSubServiceImpl();
        pubSubService.pubSubQuery(workflowTask);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testPubSubQueryWithEmptyQuery() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("{}");
        PubSubServiceImpl pubSubService = new PubSubServiceImpl();
        pubSubService.pubSubQuery(workflowTask);
    }

    @Test
    public void testFormatWithOutFormat1() throws Exception {
        PubSubServiceImpl pubSubService = new PubSubServiceImpl();
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQuery();
        Segment segment = new SegmentDelegate(workflowTask);
        Assert.assertEquals(segment, pubSubService.format(new PubSubConfig(), new PubSubQuery(), workflowTask, segment));
    }

    @Test
    public void testFormatWithOutFormat2() throws Exception {
        PubSubServiceImpl pubSubService = new PubSubServiceImpl();
        pubSubService.setFormatters(Collections.singletonMap("HELLO", new PubSubFormatter() {
            @Override
            public Segment format(PubSubConfig pubSubConfig, PubSubQuery pubSubQuery, WorkflowTask workTask, Segment segment) throws Exception {
                return segment;
            }
        }));
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQuery();
        Segment segment = new SegmentDelegate(workflowTask);
        PubSubConfig pubSubConfig = new PubSubConfig();
        Assert.assertEquals(segment, pubSubService.format(pubSubConfig, new PubSubQuery(), workflowTask, segment));
    }

    @Test(expected = IllegalArgumentException.class)
    public void testFormatWithOutFormat3() throws Exception {
        PubSubServiceImpl pubSubService = new PubSubServiceImpl();
        pubSubService.setFormatters(Collections.singletonMap("HELLO", new PubSubFormatter() {
            @Override
            public Segment format(PubSubConfig pubSubConfig, PubSubQuery pubSubQuery, WorkflowTask workTask, Segment segment) throws Exception {
                return segment;
            }
        }));
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQuery();
        Segment segment = new SegmentDelegate(workflowTask);
        PubSubConfig pubSubConfig = new PubSubConfig();
        pubSubConfig.setFormatter("FORMAT");
        Assert.assertEquals(segment, pubSubService.format(pubSubConfig, new PubSubQuery(), workflowTask, segment).getContent());
    }

    @Test
    public void testFormat() throws Exception {
        PubSubServiceImpl pubSubService = new PubSubServiceImpl();
        pubSubService.setFormatters(Collections.singletonMap("HELLO", new PubSubFormatter() {
            @Override
            public Segment format(PubSubConfig pubSubConfig, PubSubQuery pubSubQuery, WorkflowTask workTask, Segment segment) throws Exception {
                segment.setContent("WORLD");
                return segment;
            }
        }));
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQuery();
        Segment segment = new SegmentDelegate(workflowTask);
        PubSubConfig pubSubConfig = new PubSubConfig();
        pubSubConfig.setFormatter("HELLO");
        Assert.assertEquals("WORLD", pubSubService.format(pubSubConfig, new PubSubQuery(), workflowTask, segment).getContent());
    }

    @Test
    public void testSegment() throws Exception {
        PubSubServiceImpl pubSubService = new PubSubServiceImpl();
        PubSubConfig pubSubConfig = new PubSubConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        Segment segment = pubSubService.segment(pubSubConfig, workflowTask, "HELLO", "WORLD");
        Assert.assertEquals(Notifier.SOURCE, segment.getNotifier());
        Assert.assertEquals("HELLO", segment.getContent());
        Assert.assertEquals(Integer.valueOf(200), segment.getCode());
        Assert.assertEquals("WORLD", segment.getMetadata().get(PubSubServiceImpl.KEY));
    }

    @Test
    public void testSegmentWithNullContent() throws Exception {
        PubSubServiceImpl pubSubService = new PubSubServiceImpl();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(null);

        Segment segment = pubSubService.segment(new PubSubConfig(), workflowTask, null, "WORLD");

        Assert.assertNull(segment.getContent());
    }

    @Test
    public void testSegmentWithNotifier() throws Exception {
        PubSubServiceImpl pubSubService = new PubSubServiceImpl();
        PubSubConfig pubSubConfig = new PubSubConfig();
        pubSubConfig.setNotifier("NO");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        Segment segment = pubSubService.segment(pubSubConfig, workflowTask, "HELLO", "WORLD");
        Assert.assertEquals("NO", segment.getNotifier());
        Assert.assertEquals("HELLO", segment.getContent());
        Assert.assertEquals(Integer.valueOf(200), segment.getCode());
        Assert.assertEquals("WORLD", segment.getMetadata().get(PubSubServiceImpl.KEY));
    }

    @Test
    public void testPub() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(PubSubServiceImpl.RedisPubCallback.class))).andReturn(null).anyTimes();
        EasyMock.replay(template);
        PubSubServiceImpl pubSubService = new PubSubServiceImpl();
        pubSubService.setRedis4event(template);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        PubSubConfig pubSubConfig = new PubSubConfig();
        pubSubConfig.setContainHistories(true);
        pubSubService.pub(pubSubConfig, workflowTask);
        EasyMock.verify(template);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testPubWithEmpty() throws Exception {
        PubSubServiceImpl pubSubService = new PubSubServiceImpl();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("");
        pubSubService.pub(new PubSubConfig(), workflowTask);
    }

    @Test
    public void testSub() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        List<Object> result = new ArrayList<Object>();
        List<Object> data = new ArrayList<Object>();
        result.add(data);
        data.add("HELLO".getBytes());
        data.add("WORLD".getBytes());
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(PubSubServiceImpl.RedisSubCallback.class))).andReturn(result).anyTimes();
        EasyMock.replay(template);
        PubSubServiceImpl pubSubService = new PubSubServiceImpl() {
            protected PubSubQuery pubSubQuery(WorkflowTask workTask) {
                PubSubQuery pubSubQuery = new PubSubQuery();
                pubSubQuery.setQuery("Q");
                pubSubQuery.setAnswer("A");
                return pubSubQuery;
            }
        };
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        PubSubConfig pubSubConfig = new PubSubConfig();
        pubSubConfig.setContainHistories(true);
        pubSubService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        pubSubService.setRedis4event(template);
        pubSubService.setTimeout(2000);
        pubSubService.setInterval(100);
        pubSubService.setCircle(100);
        Assert.assertEquals("WORLD", pubSubService.sub(pubSubConfig, workflowTask));
        EasyMock.verify(template);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testSubWithNull() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(PubSubServiceImpl.RedisSubCallback.class))).andReturn(null).anyTimes();
        EasyMock.replay(template);
        PubSubServiceImpl pubSubService = new PubSubServiceImpl() {
            protected PubSubQuery pubSubQuery(WorkflowTask workTask) {
                PubSubQuery pubSubQuery = new PubSubQuery();
                pubSubQuery.setQuery("Q");
                return pubSubQuery;
            }
        };
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        PubSubConfig pubSubConfig = new PubSubConfig();
        pubSubConfig.setContainHistories(true);
        pubSubService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        pubSubService.setRedis4event(template);
        pubSubService.setTimeout(2000);
        pubSubService.setInterval(100);
        pubSubService.setCircle(100);
        try {
            pubSubService.sub(pubSubConfig, workflowTask);
        } finally {
            EasyMock.verify(template);
        }
    }

    @Test(expected = IllegalArgumentException.class)
    public void testSubInValidData() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        List<Object> result = new ArrayList<Object>();
        List<Object> data = new ArrayList<Object>();
        result.add(data);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(PubSubServiceImpl.RedisSubCallback.class))).andReturn(null).anyTimes();
        EasyMock.replay(template);
        PubSubServiceImpl pubSubService = new PubSubServiceImpl() {
            protected PubSubQuery pubSubQuery(WorkflowTask workTask) {
                PubSubQuery pubSubQuery = new PubSubQuery();
                pubSubQuery.setQuery("Q");
                return pubSubQuery;
            }
        };
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        PubSubConfig pubSubConfig = new PubSubConfig();
        pubSubConfig.setContainHistories(true);
        pubSubService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        pubSubService.setRedis4event(template);
        pubSubService.setTimeout(2000);
        pubSubService.setInterval(100);
        pubSubService.setCircle(100);
        try {
            pubSubService.sub(pubSubConfig, workflowTask);
        } finally {
            EasyMock.verify(template);
        }
    }

    @Test
    public void testSubExceptionWithStaticAnswer() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        List<Object> result = new ArrayList<Object>();
        List<Object> data = new ArrayList<Object>();
        result.add(data);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(PubSubServiceImpl.RedisSubCallback.class))).andReturn(null).anyTimes();
        EasyMock.replay(template);
        PubSubServiceImpl pubSubService = new PubSubServiceImpl() {
            protected PubSubQuery pubSubQuery(WorkflowTask workTask) {
                PubSubQuery pubSubQuery = new PubSubQuery();
                pubSubQuery.setQuery("Q");
                pubSubQuery.setAnswer("A");
                return pubSubQuery;
            }
        };
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        PubSubConfig pubSubConfig = new PubSubConfig();
        pubSubConfig.setContainHistories(true);
        pubSubService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        pubSubService.setRedis4event(template);
        try {
            Assert.assertEquals("A", pubSubService.sub(pubSubConfig, workflowTask));
        } finally {
            EasyMock.verify(template);
        }
    }

    @Test
    public void testSubExceptionWithStaticReply() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        List<Object> result = new ArrayList<Object>();
        List<Object> data = new ArrayList<Object>();
        result.add(data);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(PubSubServiceImpl.RedisSubCallback.class))).andReturn(null).anyTimes();
        EasyMock.replay(template);
        PubSubServiceImpl pubSubService = new PubSubServiceImpl() {
            protected PubSubQuery pubSubQuery(WorkflowTask workTask) {
                PubSubQuery pubSubQuery = new PubSubQuery();
                pubSubQuery.setQuery("Q");
                return pubSubQuery;
            }
        };
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        PubSubConfig pubSubConfig = new PubSubConfig();
        pubSubConfig.setContainHistories(true);
        pubSubService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        pubSubService.setRedis4event(template);
        pubSubConfig.setReply("A2");
        Assert.assertEquals("A2", pubSubService.sub(pubSubConfig, workflowTask));
        EasyMock.verify(template);
    }

    @Test
    public void testInit() throws Exception {
        RedisTemplate<String, Object> redisTemplate = EasyMock.createMock(RedisTemplate.class);
        EasyMock.replay(redisTemplate);
        Map<String, PubSubFormatter> formatterMap = new HashMap<>();
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        PubSubServiceImpl.InitConfig service = new PubSubServiceImpl.InitConfig();
        service.setNotifierService(notifierManager);
        service.setTimeout(100);
        service.setExpire(200);
        service.setCircle(300);
        service.setFormatters(formatterMap);
        service.setRedis4event(redisTemplate);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.replay(historyStore);
        PubSubServiceImpl empty = (PubSubServiceImpl) service.pubSubService();
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(redisTemplate, empty.getRedis4event());
        Assert.assertEquals(formatterMap, empty.getFormatters());
        Assert.assertEquals(Integer.valueOf(100), empty.getTimeout());
        Assert.assertEquals(Integer.valueOf(200), empty.getExpire());
        Assert.assertEquals(Integer.valueOf(300), empty.getCircle());
        Assert.assertEquals(formatterMap, empty.getFormatters());
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(redisTemplate, empty.getRedis4event());
        EasyMock.verify(redisTemplate);
    }

    @Test
    public void setSubWithTimeout() throws Exception {
        PubSubServiceImpl pubSubService = new PubSubServiceImpl() {
            @Override
            public String sub(Integer timeout, String k) throws Exception {
                Assert.assertEquals(Integer.valueOf(10086), timeout);
                Assert.assertEquals("HELLO", k);
                return "WORLD";
            }
        };
        pubSubService.setTimeout(10086);
        Assert.assertEquals("WORLD", pubSubService.sub("HELLO"));
    }

    @Test
    public void setPubWithTimeout() throws Exception {
        PubSubServiceImpl pubSubService = new PubSubServiceImpl() {
            @Override
            public void pub(Integer expire, String k, String v) throws Exception {
                Assert.assertEquals(Integer.valueOf(10086), expire);
                Assert.assertEquals("HELLO", k);
                Assert.assertEquals("WORLD", v);
            }
        };
        pubSubService.setExpire(10086);
        pubSubService.pub("HELLO", "WORLD");
    }

    @Test
    public void testSubWithNullConfig() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        List<Object> result = Arrays.asList(Arrays.asList("K".getBytes(), "V".getBytes()));
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(org.springframework.data.redis.core.RedisCallback.class))).andReturn(result);
        EasyMock.replay(template);
        PubSubServiceImpl service = new PubSubServiceImpl();
        service.setRedis4event(template);
        service.setTimeout(100);
        service.setInterval(100);
        service.setCircle(100);
        Assert.assertEquals("V", ((PubSubService) service).sub((PubSubConfig) null, "KEY"));
    }

    @Test
    public void testPubWithNullConfig() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(org.springframework.data.redis.core.RedisCallback.class))).andReturn(null);
        EasyMock.replay(template);
        PubSubServiceImpl service = new PubSubServiceImpl();
        service.setRedis4event(template);
        service.setExpire(100);
        ((PubSubService) service).pub((PubSubConfig) null, "K", "V");
        EasyMock.verify(template);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testFormatInvalidFormatter() throws Exception {
        PubSubServiceImpl service = new PubSubServiceImpl();
        Map<String, PubSubFormatter> formatterMap = new HashMap<>();
        formatterMap.put("ABC", new PubSubFormatter() {
            @Override
            public Segment format(PubSubConfig pubSubConfig, PubSubQuery pubSubQuery, WorkflowTask workTask, Segment segment) throws Exception {
                return null;
            }
        });
        service.setFormatters(formatterMap);
        PubSubConfig config = new PubSubConfig();
        config.setFormatter("INVALID");
        service.format(config, new PubSubQuery(), ObjectBuilder.buildWorkflowTask(), null);
    }
}
