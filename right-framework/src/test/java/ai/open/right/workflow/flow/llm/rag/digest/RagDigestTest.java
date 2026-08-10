package ai.open.right.workflow.flow.llm.rag.digest;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.SegmentUsage;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.flow.llm.rag.remote.RagRemoteConfig;
import ai.open.right.workflow.flow.llm.store.digest.Digest;
import ai.open.right.workflow.flow.llm.store.digest.DigestStore;
import ai.open.right.workflow.flow.llm.token.TokenData;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import com.fasterxml.jackson.core.JsonParseException;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.*;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

public class RagDigestTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = RagDigest.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = RagDigest.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testWithoutRagDigest() throws Exception {
        RagConfig ragConfig = new RagConfig();
        RagDigestConfig ragDigestConfig = new RagDigestConfig();
        ragDigestConfig.setDynamic("HELLO");
        ragConfig.setRagDigestConfig(ragDigestConfig);
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .build();
        RagDigest digest = new RagDigest();
        digest.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect());
        RagDigest.DigestFuture digestFuture = digest.new DigestFuture(ragConfig, ragData) {
            protected String upsertDigest(RagDigestConfig ragDigestConfig, Digest digest) throws Exception {
                return "";
            }
        };
        digestFuture.call();
    }

    @Test
    public void testWithoutDigestBody() throws Exception {
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
                workflowTask.setQuery("HELLO");
                Segment segment1 = Segment.build(workflowTask, Segment.SegmentConfig.builder().build());
                segment1.setUsage(new SegmentUsage(TokenData.builder().cache(1).total(1).build()));
                notifierWriteBack.writeBack(segment1);
            }
        };
        RagDigestConfig ragDigestConfig = new RagDigestConfig();
        ragDigestConfig.setDynamic("WORKFLOW");
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagDigestConfig(ragDigestConfig);
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .build();
        RagDigest digest = new RagDigest();
        digest.setNotifierService(notifierManager);
        digest.setTimeout4Llm(10000);
        RagDigest.DigestFuture digestFuture = digest.new DigestFuture(ragConfig, ragData) {
            protected String upsertDigest(RagDigestConfig ragDigestConfig, Digest digest) throws Exception {
                return "";
            }
        };
        digestFuture.call();
    }

    @Test
    public void testWithDigestBodyAndMaps() throws Exception {
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
                workflowTask.setQuery("{\"Key1\":\"Val1\"}");
                Segment segment1 = Segment.build(workflowTask, Segment.SegmentConfig.builder().build());
                segment1.setUsage(new SegmentUsage(TokenData.builder().cache(1).total(1).build()));
                notifierWriteBack.writeBack(segment1);
            }
        };
        RagDigestConfig ragDigestConfig = new RagDigestConfig();
        ragDigestConfig.setDynamic("WORKFLOW");
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagDigestConfig(ragDigestConfig);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .query(llmQuery)
                .build();
        RagDigest ragDigest = new RagDigest();
        ragDigest.setTimeout4Llm(10000);
        ragDigest.setNotifierService(notifierManager);
        RagDigest.DigestFuture digestFuture = ragDigest.new DigestFuture(ragConfig, ragData) {
            protected String upsertDigest(RagDigestConfig ragDigestConfig, Digest digest) throws Exception {
                return "{\"Key1\":\"Val1\"}";
            }
        };
        digestFuture.call();
    }

    @Test
    public void testWithDigestBodyAndInvalidMaps() throws Exception {
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
                // Invalid Json
                workflowTask.setQuery("{\"Key1\":\"Val1\"");
                Segment segment1 = Segment.build(workflowTask, Segment.SegmentConfig.builder().build());
                segment1.setUsage(new SegmentUsage(TokenData.builder().cache(1).total(1).build()));
                notifierWriteBack.writeBack(segment1);
            }
        };
        RagDigestConfig ragDigestConfig = new RagDigestConfig();
        ragDigestConfig.setDynamic("WORKFLOW");
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagDigestConfig(ragDigestConfig);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .query(llmQuery)
                .build();
        RagDigest ragDigest = new RagDigest();
        ragDigest.setTimeout4Llm(10000);
        ragDigest.setNotifierService(notifierManager);
        RagDigest.DigestFuture digestFuture = ragDigest.new DigestFuture(ragConfig, ragData) {
            protected String upsertDigest(RagDigestConfig ragDigestConfig, Digest digest) throws Exception {
                // Invalid Json
                return "{\"Key1\":\"Val1\"";
            }
        };
        digestFuture.call();
    }

    @Test
    public void testUpsertRagResponseWithNull() throws Exception {
        RagDigestConfig ragDigestConfig = new RagDigestConfig();
        ragDigestConfig.setDynamic("WORKFLOW");
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagDigestConfig(ragDigestConfig);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .query(llmQuery)
                .build();
        Digest digest = new Digest(new HashMap<String, Object>(), new ArrayList<String>());
        DigestStore digestStore = EasyMock.createMock(DigestStore.class);
        EasyMock.expect(digestStore.upsert(llmQuery, "UNKNOWN", digest)).andReturn(digest);
        EasyMock.replay(digestStore);
        RagDigest ragDigest = new RagDigest();
        ragDigest.setDigestStore(digestStore);
        RagDigest.DigestFuture digestFuture = ragDigest.new DigestFuture(ragConfig, ragData);
        Assert.assertEquals(null, digestFuture.upsertDigest(ragDigestConfig, digest));
        EasyMock.verify(digestStore);
    }

    @Test
    public void testUpsertRagResponseWithDigest() throws Exception {
        RagDigestConfig ragDigestConfig = new RagDigestConfig();
        ragDigestConfig.setDynamic("WORKFLOW");
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagDigestConfig(ragDigestConfig);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .query(llmQuery)
                .build();
        Digest digest = new Digest(new HashMap<String, Object>(), Arrays.asList("KEY1"));
        digest.getDigest().put("KEY1", "VAL1");
        DigestStore digestStore = EasyMock.createMock(DigestStore.class);
        EasyMock.expect(digestStore.upsert(llmQuery, "UNKNOWN", digest)).andReturn(digest);
        EasyMock.replay(digestStore);
        RagDigest ragDigest = new RagDigest();
        ragDigest.setDigestStore(digestStore);
        RagDigest.DigestFuture digestFuture = ragDigest.new DigestFuture(ragConfig, ragData);
        Assert.assertEquals("<?xml version=\"1.0\" encoding=\"UTF-8\"?><Digest xmlns=\"\"><KEY1>VAL1</KEY1></Digest>", digestFuture.upsertDigest(ragDigestConfig, digest));
        EasyMock.verify(digestStore);
    }

    @Test
    public void testRagWithNotAllowed() throws Exception {
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
                workflowTask.setQuery("");
                notifierWriteBack.writeBack(Segment.build(workflowTask, Segment.SegmentConfig.builder().build()));
            }
        };
        RagDigestConfig ragDigestConfig = new RagDigestConfig();
        ragDigestConfig.setKeys(Arrays.asList("KEY1"));
        ragDigestConfig.setDynamic("WORKFLOW");
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagDigestConfig(ragDigestConfig);
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .build();
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        RagDigest digest = new RagDigest();
        digest.setNotifierService(notifierManager);
        digest.setExecutorService(executorService);
        digest.rag(ragConfig, ragData);
        executorService.shutdown();
    }

    @Test
    public void testRag() throws Exception {
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
                workflowTask.setQuery("");
                notifierWriteBack.writeBack(Segment.build(workflowTask, Segment.SegmentConfig.builder().build()));
            }
        };
        RagDigestConfig ragDigestConfig = new RagDigestConfig();
        ragDigestConfig.setKeys(Arrays.asList("KEY1"));
        ragDigestConfig.setDynamic("WORKFLOW");
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagRemoteConfig(new RagRemoteConfig());
        ragConfig.setRagDigestConfig(ragDigestConfig);
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .build();
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        RagDigest digest = new RagDigest();
        digest.setNotifierService(notifierManager);
        digest.setExecutorService(executorService);
        digest.rag(ragConfig, ragData);
        executorService.shutdown();
    }

    @Test
    public void testGetRagRequestWithEmpty() throws Exception {
        RagDigestConfig ragDigestConfig = new RagDigestConfig();
        ragDigestConfig.setDynamic("WORKFLOW");
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagDigestConfig(ragDigestConfig);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .query(llmQuery)
                .build();
        Digest digest = new Digest(new HashMap<String, Object>(), new ArrayList<String>());
        digest.getDigest().put("KEY1", "VAL1");
        RagDigest ragDigest = new RagDigest();
        RagDigest.DigestFuture digestFuture = ragDigest.new DigestFuture(ragConfig, ragData);
        Map<String, Object> digestMaps = digestFuture.parseDigest("{}");
        Assert.assertEquals(0, digestMaps.size());
    }

    @Test
    public void testGetRagRequestWithJson() throws Exception {
        RagDigestConfig ragDigestConfig = new RagDigestConfig();
        ragDigestConfig.setDynamic("WORKFLOW");
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagDigestConfig(ragDigestConfig);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .query(llmQuery)
                .build();
        Digest digest = new Digest(new HashMap<String, Object>(), new ArrayList<String>());
        digest.getDigest().put("KEY1", "VAL1");
        RagDigest ragDigest = new RagDigest();
        RagDigest.DigestFuture digestFuture = ragDigest.new DigestFuture(ragConfig, ragData);
        Map<String, Object> digestMaps = digestFuture.parseDigest("{\"KEY1\":\"VAL1\"}");
        Assert.assertEquals(1, digestMaps.size());
    }

    @Test
    public void testGetRagRequestWithProperties() throws Exception {
        RagDigestConfig ragDigestConfig = new RagDigestConfig();
        ragDigestConfig.setDynamic("WORKFLOW");
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagDigestConfig(ragDigestConfig);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .query(llmQuery)
                .build();
        Digest digest = new Digest(new HashMap<String, Object>(), new ArrayList<String>());
        digest.getDigest().put("KEY1", "VAL1");
        RagDigest ragDigest = new RagDigest();
        RagDigest.DigestFuture digestFuture = ragDigest.new DigestFuture(ragConfig, ragData);
        Map<String, Object> digestMaps = digestFuture.parseDigest("KEY1=VAL1\nKEY2=VAL2");
        Assert.assertEquals(2, digestMaps.size());
    }

    @Test
    public void testGetRagRequestWithLikeJson() throws Exception {
        RagDigestConfig ragDigestConfig = new RagDigestConfig();
        ragDigestConfig.setDynamic("WORKFLOW");
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagDigestConfig(ragDigestConfig);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .query(llmQuery)
                .build();
        Digest digest = new Digest(new HashMap<String, Object>(), new ArrayList<String>());
        digest.getDigest().put("KEY1", "VAL1");
        RagDigest ragDigest = new RagDigest();
        RagDigest.DigestFuture digestFuture = ragDigest.new DigestFuture(ragConfig, ragData);
        Map<String, Object> digestMaps = digestFuture.parseDigest("KEY{1=VAL1\nKEY2=VAL}2");
        Assert.assertEquals(2, digestMaps.size());
    }

    @Test
    public void testGetRagRequestWithRightJson() throws Exception {
        RagDigestConfig ragDigestConfig = new RagDigestConfig();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagDigestConfig(ragDigestConfig);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .query(llmQuery)
                .build();
        RagDigest ragDigest = new RagDigest();
        RagDigest.DigestFuture digestFuture = ragDigest.new DigestFuture(ragConfig, ragData);
        Assert.assertEquals("VAL", digestFuture.parseDigest("{\"KEY\":\"VAL\"}").get("KEY"));
    }

    @Test(expected = JsonParseException.class)
    public void testGetRagRequestWithInvalidJson() throws Exception {
        RagDigestConfig ragDigestConfig = new RagDigestConfig();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagDigestConfig(ragDigestConfig);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .query(llmQuery)
                .build();
        RagDigest ragDigest = new RagDigest();
        RagDigest.DigestFuture digestFuture = ragDigest.new DigestFuture(ragConfig, ragData);
        Assert.assertEquals(null, digestFuture.parseDigest("{\"KEY\"{:}\"VAL\"}").get("KEY"));
    }

    @Test
    public void testWithConditionFailed() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("false");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        RagDigest digest = new RagDigest();
        digest.setNotifierService(notifierManager);
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        RagDigestConfig ragDigestConfig = new RagDigestConfig();
        ragConfig.setRagDigestConfig(ragDigestConfig);
        Assert.assertEquals(RagFuture.NOTHING, digest.rag(ragConfig, ragData));
    }

    @Test
    public void testUpsertRagResponseWithJson() throws Exception {
        RagDigestConfig ragDigestConfig = new RagDigestConfig();
        ragDigestConfig.setDynamic("WORKFLOW");
        ragDigestConfig.setMode(RagDigestConfig.MODE_JSON);
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagDigestConfig(ragDigestConfig);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .query(llmQuery)
                .build();
        Digest digest = new Digest(new HashMap<String, Object>(), Arrays.asList("KEY1"));
        digest.getDigest().put("KEY1", "VAL1");
        DigestStore digestStore = EasyMock.createMock(DigestStore.class);
        EasyMock.expect(digestStore.upsert(llmQuery, "UNKNOWN", digest)).andReturn(digest);
        EasyMock.replay(digestStore);
        RagDigest ragDigest = new RagDigest();
        ragDigest.setDigestStore(digestStore);
        RagDigest.DigestFuture digestFuture = ragDigest.new DigestFuture(ragConfig, ragData);
        Assert.assertEquals("{\"KEY1\":\"VAL1\"}", digestFuture.upsertDigest(ragDigestConfig, digest));
        EasyMock.verify(digestStore);
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        ExecutorService executorService = EasyMock.createMock(ExecutorService.class);
        DigestStore digestStore = EasyMock.createMock(DigestStore.class);
        EasyMock.replay(executorService, digestStore);
        RagDigest.InitConfig service = new RagDigest.InitConfig();
        service.setNotifierService(notifierManager);
        service.setExecutorService(executorService);
        service.setDigestStore(digestStore);
        service.setTimeout4Condition(10086);
        service.setTimeout4Llm(1000);
        RagDigest empty = service.ragDigest();
        Assert.assertEquals(Integer.valueOf(10086), empty.getTimeout4Condition());
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(executorService, empty.getExecutorService());
        Assert.assertEquals(Integer.valueOf(1000), empty.getTimeout4Llm());
        Assert.assertEquals(digestStore, empty.getDigestStore());
        EasyMock.verify(executorService, digestStore);
    }
    @Test
    public void testAllowedNoDigest() throws Exception {
        RagDigest service = new RagDigest();
        RagConfig config = new RagConfig();
        Assert.assertFalse(service.allowed(config, null));
    }
}
