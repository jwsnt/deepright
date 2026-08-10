package ai.open.right.workflow.flow.summary.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.SegmentUsage;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestModel;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.flow.summary.SummaryConfig;
import ai.open.right.workflow.flow.file.DefStore;
import ai.open.right.workflow.flow.summary.SummaryPart;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.apache.commons.lang3.StringUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.List;
import java.util.concurrent.CopyOnWriteArrayList;
import java.util.concurrent.atomic.AtomicReference;

public class SummaryServiceImplTest {

    private static SummaryConfig defaultSummaryConfig() {
        return new SummaryConfig();
    }

    private static WorkflowTask defaultWorkflowTask() {
        return ObjectBuilder.buildWorkflowTask();
    }

    @Test
    public void testGetPair1() throws Exception {
        SummaryServiceImpl summaryServiceImpl = new SummaryServiceImpl();
        List<HistoryPair> pairs = summaryServiceImpl.buildPairs(defaultSummaryConfig(), defaultWorkflowTask(), "[{\"answer\":\"WORLD\",\"query\":\"HELLO\"}]");
        HistoryPair pair = pairs.get(0);
        Assert.assertEquals("HELLO", pair.getQuery());
        Assert.assertEquals("WORLD", pair.getAnswer());
    }

    @Test
    public void testGetPair1WithTwoKey() throws Exception {
        SummaryServiceImpl summaryServiceImpl = new SummaryServiceImpl();
        List<HistoryPair> pairs1 = summaryServiceImpl.buildPairs(defaultSummaryConfig(), defaultWorkflowTask(), "[{\"answer\":\"WORLD1\",\"query\":\"HELLO1\"}],{\"answer\":\"HELLO2\",\"query\":\"WORLD2\"}]]");
        HistoryPair pair = pairs1.get(0);
        Assert.assertEquals("HELLO1", pair.getQuery());
        Assert.assertEquals("WORLD1", pair.getAnswer());
        List<HistoryPair> pairs2 = summaryServiceImpl.buildPairs(defaultSummaryConfig(), defaultWorkflowTask(), "[{\"answer\":\"WORLD1\",\"query\":\"HELLO1\"},{\"answer\":\"WORLD2\",\"query\":\"HELLO2\"}]");
        pair = pairs2.get(pairs2.size() - 1);
        Assert.assertEquals("HELLO2", pair.getQuery());
        Assert.assertEquals("WORLD2", pair.getAnswer());
    }

    @Test
    public void testGetPair2() throws Exception {
        SummaryServiceImpl summaryServiceImpl = new SummaryServiceImpl();
        List<HistoryPair> pairs = summaryServiceImpl.buildPairs(defaultSummaryConfig(), defaultWorkflowTask(), "HELLO=WORLD");
        HistoryPair pair = pairs.get(0);
        Assert.assertEquals("HELLO", pair.getQuery());
        Assert.assertEquals("WORLD", pair.getAnswer());
    }

    /** buildPairs：JSON 与文本分支均为 HistoryPair 设置默认 model、api */
    @Test
    public void testBuildPairs_jsonAndTextBranch_setDefaultModelAndApi() throws Exception {
        SummaryServiceImpl impl = new SummaryServiceImpl();
        WorkflowTask task = defaultWorkflowTask();
        HistoryPair jsonPair = impl.buildPairs(defaultSummaryConfig(), task, "[{\"answer\":\"A\",\"query\":\"Q\"}]").get(0);
        Assert.assertEquals(ProviderRequestModel.DEF, jsonPair.getModel());
        Assert.assertEquals(ProviderRequest.REQUEST_DEF, jsonPair.getApi());
        HistoryPair textPair = impl.buildPairs(defaultSummaryConfig(), task, "K=V").get(0);
        Assert.assertEquals(ProviderRequestModel.DEF, textPair.getModel());
        Assert.assertEquals(ProviderRequest.REQUEST_DEF, textPair.getApi());
    }

    @Test
    public void testGetPair3() throws Exception {
        SummaryServiceImpl summaryServiceImpl = new SummaryServiceImpl();
        List<HistoryPair> pairs = summaryServiceImpl.buildPairs(defaultSummaryConfig(), defaultWorkflowTask(), "{\"HELLO=WORLD\"");
        HistoryPair pair = pairs.get(0);
        Assert.assertEquals("{\"HELLO", pair.getQuery());
        Assert.assertEquals("WORLD\"", pair.getAnswer());
    }

    @Test
    public void testGetPair4() throws Exception {
        SummaryServiceImpl summaryServiceImpl = new SummaryServiceImpl();
        List<HistoryPair> pairs = summaryServiceImpl.buildPairs(defaultSummaryConfig(), defaultWorkflowTask(), "{\"HELLO_WORLD\"");
        Assert.assertTrue(pairs.isEmpty());
    }

    @Test
    public void testAllowedTRUE() throws Exception {
        SummaryServiceImpl summaryServiceImpl = new SummaryServiceImpl();
        SummaryConfig summaryConfig = new SummaryConfig();
        summaryConfig.setCondition("NEXT");
        List<History> histories = new ArrayList<History>();
        summaryServiceImpl.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("TRUE"));
        Assert.assertTrue(summaryServiceImpl.allowed(summaryConfig, ObjectBuilder.buildWorkflowTask(), histories, ""));
    }

    @Test
    public void testAllowedFALSE() throws Exception {
        SummaryServiceImpl summaryServiceImpl = new SummaryServiceImpl();
        SummaryConfig summaryConfig = new SummaryConfig();
        summaryConfig.setCondition("NEXT");
        List<History> histories = new ArrayList<History>();
        summaryServiceImpl.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("FALSE"));
        Assert.assertFalse(summaryServiceImpl.allowed(summaryConfig, ObjectBuilder.buildWorkflowTask(), histories, ""));
    }

    @Test
    public void testAllowedWithOutCondition() throws Exception {
        SummaryServiceImpl summaryServiceImpl = new SummaryServiceImpl();
        SummaryConfig summaryConfig = new SummaryConfig();
        List<History> histories = new ArrayList<History>();
        Assert.assertTrue(summaryServiceImpl.allowed(summaryConfig, ObjectBuilder.buildWorkflowTask(), histories, ""));
    }

    @Test
    public void testSummarizeWithEmptyHistory() throws Exception {
        SummaryServiceImpl summaryServiceImpl = new SummaryServiceImpl();
        SummaryConfig summaryConfig = new SummaryConfig();
        summaryConfig.setCondition("NEXT");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        List<History> histories = new ArrayList<History>();
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(workflowTask, "UNKNOWN", Integer.MAX_VALUE, false, null)).andReturn(histories).anyTimes();
        EasyMock.replay(historyStore);
        summaryServiceImpl.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("FALSE"));
        summaryServiceImpl.setHistoryStore(historyStore);
        Assert.assertNull(summaryServiceImpl.summarize(summaryConfig, workflowTask));
        EasyMock.verify(historyStore);
    }

    @Test
    public void testSummarizeWithHistory() throws Exception {
        HistoryPair pair = new HistoryPair();
        pair.setAnswer("UNKNOWN");
        pair.setQuery("UNKNOWN");
        List<HistoryPair> pairs = List.of(pair);
        SummaryServiceImpl summaryServiceImpl = new SummaryServiceImpl() {

            @Override
            public Boolean allowed(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories, String append) throws Exception {
                return true;
            }

            @Override
            protected List<HistoryPair> buildPairs(SummaryConfig summaryConfig, WorkflowTask workTask, String content) throws Exception {
                return pairs;
            }
        };
        SummaryConfig summaryConfig = new SummaryConfig();
        summaryConfig.setExpired(1000);
        summaryConfig.setCondition("NEXT");
        summaryConfig.setDynamic("DYNAMIC");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(1000L);
        List<History> histories = new ArrayList<History>();
        History history = new History();
        history.setRole(History.ROLE_ASSISTANT);
        history.setType(History.TYPE_ANSWER);
        history.setCreated(10086L);
        histories.add(history);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(workflowTask, "UNKNOWN", Integer.MAX_VALUE, false, null)).andReturn(histories).anyTimes();
        historyStore.store(workflowTask, Arrays.asList("UNKNOWN"), pairs, 1000, Integer.MAX_VALUE);
        EasyMock.expectLastCall();
        historyStore.clear(workflowTask, Arrays.asList("UNKNOWN"), false, -10086L);
        EasyMock.expectLastCall();
        EasyMock.replay(historyStore);
        summaryServiceImpl.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("FALSE"));
        summaryServiceImpl.setHistoryStore(historyStore);
        summaryServiceImpl.setMaxSize(Integer.MAX_VALUE);
        summaryServiceImpl.setStore("");
        Assert.assertEquals("FALSE", summaryServiceImpl.summarize(summaryConfig, workflowTask).getContent());
        EasyMock.verify(historyStore);
    }

    @Test
    public void testSummarizeWithHistoryAndNotStore() throws Exception {
        HistoryPair pair = new HistoryPair();
        pair.setAnswer("UNKNOWN");
        pair.setQuery("UNKNOWN");
        List<HistoryPair> historyPairs = List.of(pair);
        SummaryServiceImpl summaryServiceImpl = new SummaryServiceImpl() {
            @Override
            public Boolean allowed(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories, String append) throws Exception {
                return true;
            }

            @Override
            protected List<HistoryPair> buildPairs(SummaryConfig summaryConfig, WorkflowTask workTask, String content) throws Exception {
                return historyPairs;
            }
        };
        SummaryConfig summaryConfig = new SummaryConfig();
        summaryConfig.setExpired(1000);
        summaryConfig.setCondition("NEXT");
        summaryConfig.setDynamic("DYNAMIC");
        summaryConfig.setStore(false);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(1000L);
        List<History> histories = new ArrayList<History>();
        History history = new History();
        history.setRole(History.ROLE_ASSISTANT);
        history.setType(History.TYPE_ANSWER);
        histories.add(history);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(workflowTask, "UNKNOWN", Integer.MAX_VALUE, false, null)).andReturn(histories).anyTimes();
        EasyMock.replay(historyStore);
        summaryServiceImpl.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("FALSE"));
        summaryServiceImpl.setHistoryStore(historyStore);
        summaryServiceImpl.setMaxSize(Integer.MAX_VALUE);
        summaryServiceImpl.setStore("");
        Assert.assertEquals("FALSE", summaryServiceImpl.summarize(summaryConfig, workflowTask).getContent());
        Assert.assertEquals(historyPairs, summaryServiceImpl.summarize(summaryConfig, workflowTask).getPairs());
        EasyMock.verify(historyStore);
    }

    @Test
    public void testSummarizeWithHistoryAndNotStoreDesc() throws Exception {
        SummaryServiceImpl summaryServiceImpl = new SummaryServiceImpl() {
            @Override
            public Boolean allowed(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories, String append) throws Exception {
                return true;
            }

            @Override
            protected List<HistoryPair> buildPairs(SummaryConfig summaryConfig, WorkflowTask workTask, String content) throws Exception {
                HistoryPair pair = new HistoryPair();
                pair.setAnswer("UNKNOWN");
                pair.setQuery("UNKNOWN");
                return List.of(pair);
            }
        };
        SummaryConfig summaryConfig = new SummaryConfig();
        summaryConfig.setExpired(1000);
        summaryConfig.setCondition("NEXT");
        summaryConfig.setDynamic("DYNAMIC");
        summaryConfig.setStore(false);
        summaryConfig.setDesc(true);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(1000L);
        List<History> histories = new ArrayList<History>();
        History history = new History();
        history.setRole(History.ROLE_ASSISTANT);
        history.setType(History.TYPE_ANSWER);
        histories.add(history);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(workflowTask, "UNKNOWN", Integer.MAX_VALUE, true, null)).andReturn(histories).anyTimes();
        EasyMock.replay(historyStore);
        summaryServiceImpl.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("FALSE"));
        summaryServiceImpl.setHistoryStore(historyStore);
        summaryServiceImpl.setMaxSize(Integer.MAX_VALUE);
        summaryServiceImpl.setStore("");
        Assert.assertEquals("FALSE", summaryServiceImpl.summarize(summaryConfig, workflowTask).getContent());
        EasyMock.verify(historyStore);
    }

    /** 覆盖 summarize：query 来自 buildQuery，buildMediaContext 使用 query；store 为空时 mediaContext 为 null。 */
    @Test
    public void testSummarizePassesMediaContextFromBuildMediaContext() throws Exception {
        HistoryPair pair = new HistoryPair();
        pair.setAnswer("UNKNOWN");
        pair.setQuery("UNKNOWN");
        List<HistoryPair> pairs = List.of(pair);
        final AtomicReference<List<MediaContext>> capturedMediaContext = new AtomicReference<>();
        NotifierServiceImpl capturingNotifier = new NotifierServiceImpl() {
            @Override
            public void notify(ai.open.right.workflow.flow.llm.Segment segment, ai.open.right.context.RedirectContext redirectContext, ai.open.right.workflow.notify.NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                capturedMediaContext.set(mediaContext);
                segment.setContent("FALSE");
                segment.setUsage(new SegmentUsage());
                notifierWriteBack.writeBack(segment);
            }
        };
        SummaryServiceImpl summaryServiceImpl = new SummaryServiceImpl() {
            @Override
            public Boolean allowed(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories, String append) throws Exception {
                return true;
            }

            @Override
            protected List<HistoryPair> buildPairs(SummaryConfig summaryConfig, WorkflowTask workTask, String content) throws Exception {
                return pairs;
            }
        };
        SummaryConfig summaryConfig = new SummaryConfig();
        summaryConfig.setExpired(1000);
        summaryConfig.setCondition("NEXT");
        summaryConfig.setDynamic("DYNAMIC");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(1000L);
        List<History> histories = new ArrayList<>();
        History history = new History();
        history.setRole(History.ROLE_ASSISTANT);
        history.setType(History.TYPE_ANSWER);
        history.setCreated(10086L);
        histories.add(history);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(workflowTask, "UNKNOWN", Integer.MAX_VALUE, false, null)).andReturn(histories).anyTimes();
        historyStore.store(workflowTask, Arrays.asList("UNKNOWN"), pairs, 1000, Integer.MAX_VALUE);
        EasyMock.expectLastCall();
        historyStore.clear(workflowTask, Arrays.asList("UNKNOWN"), false, -10086L);
        EasyMock.expectLastCall();
        EasyMock.replay(historyStore);
        summaryServiceImpl.setNotifierService(capturingNotifier);
        summaryServiceImpl.setHistoryStore(historyStore);
        summaryServiceImpl.setMaxSize(Integer.MAX_VALUE);
        summaryServiceImpl.setStore("");
        Assert.assertEquals("FALSE", summaryServiceImpl.summarize(summaryConfig, workflowTask).getContent());
        Assert.assertNull(capturedMediaContext.get());
        EasyMock.verify(historyStore);
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.replay(historyStore);
        SummaryServiceImpl.InitConfig service = new SummaryServiceImpl.InitConfig();
        service.setProvider4maxSize(1000);
        service.setProvider4maxRate(1.0);
        service.setNotifierService(notifierManager);
        service.setHistoryStore(historyStore);
        service.setTimeout4Llm(1000);
        SummaryServiceImpl empty = (SummaryServiceImpl) service.summaryService();
        Assert.assertEquals(historyStore, empty.getHistoryStore());
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(Integer.valueOf(1000), empty.getTimeout4Llm());
        EasyMock.verify(historyStore);
    }

    /** InitConfig.summaryService()：当 summary4maxSize 非空时，maxSize 取 summary4maxSize。 */
    @Test
    public void testInitConfigSummaryServiceMaxSizeFromSummary4maxSize() throws Exception {
        SummaryServiceImpl.InitConfig initConfig = new SummaryServiceImpl.InitConfig();
        initConfig.setProvider4maxSize(2048);
        initConfig.setSummary4maxSize(4096);
        SummaryServiceImpl service = (SummaryServiceImpl) initConfig.summaryService();
        Assert.assertEquals(Integer.valueOf(4096), service.getMaxSize());
    }

    /** InitConfig.summaryService()：当 summary4maxSize 为空时，maxSize 取 (int)(provider4maxSize * provider4maxRate)。未设 provider4maxRate 时需其他依赖避免 NPE，此处设 1 等价仅用 provider4maxSize。 */
    @Test
    public void testInitConfigSummaryServiceMaxSizeFromProvider4maxSize() throws Exception {
        SummaryServiceImpl.InitConfig initConfig = new SummaryServiceImpl.InitConfig();
        initConfig.setProvider4maxSize(8192);
        initConfig.setProvider4maxRate(01.D);
        initConfig.setSummary4maxSize(null);
        SummaryServiceImpl service = (SummaryServiceImpl) initConfig.summaryService();
        Assert.assertEquals(Integer.valueOf(8192), service.getMaxSize());
    }

    /** InitConfig：@Value("${provider.maxRate:0.5}") provider4maxRate 参与 maxSize 计算；summary4maxSize 为空时 maxSize=(int)(provider4maxSize*provider4maxRate)。 */
    @Test
    public void testInitConfigProvider4maxRateUsedInMaxSize() throws Exception {
        SummaryServiceImpl.InitConfig initConfig = new SummaryServiceImpl.InitConfig();
        initConfig.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        initConfig.setHistoryStore(EasyMock.createMock(HistoryStore.class));
        EasyMock.replay(initConfig.getHistoryStore());
        initConfig.setProvider4maxSize(10000);
        initConfig.setSummary4maxSize(null);
        initConfig.setProvider4maxRate(1.0);
        SummaryServiceImpl service = (SummaryServiceImpl) initConfig.summaryService();
        Assert.assertEquals(Integer.valueOf(10000), service.getMaxSize());
        initConfig.setProvider4maxRate(0.0);
        SummaryServiceImpl serviceZero = (SummaryServiceImpl) initConfig.summaryService();
        Assert.assertEquals(Integer.valueOf(0), serviceZero.getMaxSize());
        initConfig.setProvider4maxRate(0.5);
        SummaryServiceImpl serviceHalf = (SummaryServiceImpl) initConfig.summaryService();
        Assert.assertEquals(Integer.valueOf(5000), serviceHalf.getMaxSize());
    }

    /** InitConfig.provider4maxRate getter：覆盖 @Value("${provider.maxRate:0.5}") 注入的字段读。 */
    @Test
    public void testInitConfigProvider4maxRateGetter() {
        SummaryServiceImpl.InitConfig initConfig = new SummaryServiceImpl.InitConfig();
        Assert.assertNull(initConfig.getProvider4maxRate());
        initConfig.setProvider4maxRate(0.5);
        Assert.assertEquals(Double.valueOf(0.5), initConfig.getProvider4maxRate());
    }

    /** InitConfig.summaryService()：defStore、requery、store 通过 copyProperties 复制到 SummaryServiceImpl。 */
    @Test
    public void testInitConfigSummaryServiceCopiesDefStoreRequeryStore() throws Exception {
        DefStore defStore = new DefStore() {};
        SummaryServiceImpl.InitConfig initConfig = new SummaryServiceImpl.InitConfig();
        initConfig.setProvider4maxSize(1000);
        initConfig.setProvider4maxRate(1.0);
        initConfig.setDefStore(defStore);
        initConfig.setRequery("custom requery");
        initConfig.setStore("custom-store");
        SummaryServiceImpl service = (SummaryServiceImpl) initConfig.summaryService();
        Assert.assertEquals(defStore, service.getDefStore());
        Assert.assertEquals("custom requery", service.getRequery());
        Assert.assertEquals("custom-store", service.getStore());
    }

    @Test(expected = IllegalArgumentException.class)
    public void testSummarizeEmptyDynamic() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        SummaryConfig config = new SummaryConfig();
        config.setNow(10086L);
        SummaryServiceImpl service = new SummaryServiceImpl();
        service.setMaxSize(Integer.MAX_VALUE);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        List<History> histories = new ArrayList<>();
        History hist = new History();
        hist.setRole(History.ROLE_ASSISTANT);
        hist.setType(History.TYPE_ANSWER);
        histories.add(hist);
        EasyMock.expect(historyStore.restore(workflowTask, "UNKNOWN", config.getMaxsize(), false, -10086L)).andReturn(histories).anyTimes();
        EasyMock.replay(historyStore);
        service.setHistoryStore(historyStore);
        service.setStore("");
        config.setDynamic("");
        service.summarize(config, workflowTask);
        EasyMock.verify(historyStore);
    }

    @Test
    public void testLLMHistoriesPromptsNull() {
        SummaryServiceImpl.LLMHistoriesPrompts prompts = new SummaryServiceImpl.LLMHistoriesPrompts(Collections.emptyList());
        Assert.assertNull(prompts.getHistory());
    }

    /** 覆盖 summarize：restore 第五参为 -summaryConfig.getNow() 当 now 非空。 */
    @Test
    public void testSummarizeRestoreWithNow() throws Exception {
        SummaryServiceImpl summaryServiceImpl = new SummaryServiceImpl() {
            @Override
            public Boolean allowed(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories, String append) throws Exception {
                return true;
            }

            @Override
            protected List<HistoryPair> buildPairs(SummaryConfig summaryConfig, WorkflowTask workTask, String content) throws Exception {
                return List.of(new HistoryPair());
            }
        };
        summaryServiceImpl.setMaxSize(Integer.MAX_VALUE);
        SummaryConfig config = new SummaryConfig();
        config.setExpired(1000);
        config.setCondition("NEXT");
        config.setDynamic("DYNAMIC");
        config.setNow(10086L);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(1000L);
        List<History> histories = new ArrayList<>();
        History h = new History();
        h.setCreated(10086L);
        h.setRole(History.ROLE_ASSISTANT);
        h.setType(History.TYPE_ANSWER);
        histories.add(h);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(workflowTask, "UNKNOWN", Integer.MAX_VALUE, false, -10086L)).andReturn(histories).anyTimes();
        historyStore.store(EasyMock.anyObject(), EasyMock.anyObject(), EasyMock.<List<HistoryPair>>anyObject(), EasyMock.eq(1000), EasyMock.anyObject());
        EasyMock.expectLastCall();
        historyStore.clear(workflowTask, Arrays.asList("UNKNOWN"), false, -10086L);
        EasyMock.expectLastCall();
        EasyMock.replay(historyStore);
        summaryServiceImpl.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("OK"));
        summaryServiceImpl.setHistoryStore(historyStore);
        summaryServiceImpl.setStore("");
        Assert.assertEquals("OK", summaryServiceImpl.summarize(config, workflowTask).getContent());
        EasyMock.verify(historyStore);
    }

    /** 覆盖 summarize：query 由 buildQuery 生成并传入 buildMediaContext；store 非空且 storeStore 支持网络存储时返回 mediaContext。 */
    @Test
    public void testSummarizeBuildMediaContextWhenStoreSupportedAndQueryOverMaxSize() throws Exception {
        String networkUrl = "https://example.com/summary.json";
        ai.open.right.workflow.flow.file.FileStore fileStoreMock = EasyMock.createMock(ai.open.right.workflow.flow.file.FileStore.class);
        EasyMock.expect(fileStoreMock.supportNetwork()).andReturn(true).anyTimes();
        EasyMock.expect(fileStoreMock.store(EasyMock.anyObject(byte[].class), EasyMock.eq(".json"), EasyMock.anyObject(WorkflowTask.class))).andReturn(networkUrl).anyTimes();
        EasyMock.replay(fileStoreMock);

        ai.open.right.workflow.flow.file.DefStore defStoreStub = new ai.open.right.workflow.flow.file.DefStore() {
            @Override
            public Boolean supportFunction(String name) throws Exception {
                return true;
            }

            @Override
            public ai.open.right.workflow.flow.file.FileStore fetchStore(String name) throws Exception {
                return fileStoreMock;
            }

            @Override
            public String store(byte[] bytes, String suffix, WorkflowTask workTask) throws Exception {
                return fileStoreMock.store(bytes, suffix, workTask);
            }
        };

        final AtomicReference<List<MediaContext>> captured = new AtomicReference<>();
        NotifierServiceImpl notifier = new NotifierServiceImpl() {
            @Override
            public void notify(ai.open.right.workflow.flow.llm.Segment segment, ai.open.right.context.RedirectContext redirectContext, ai.open.right.workflow.notify.NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                captured.set(mediaContext);
                segment.setContent("FALSE");
                segment.setUsage(new SegmentUsage());
                notifierWriteBack.writeBack(segment);
            }
        };
        SummaryServiceImpl summaryServiceImpl = new SummaryServiceImpl() {
            @Override
            public Boolean allowed(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories, String append) throws Exception {
                return true;
            }

            @Override
            protected List<HistoryPair> buildPairs(SummaryConfig summaryConfig, WorkflowTask workTask, String content) throws Exception {
                return List.of(new HistoryPair());
            }
        };
        summaryServiceImpl.setMaxSize(1);
        summaryServiceImpl.setDefStore(defStoreStub);
        summaryServiceImpl.setStore("file.store.s3");
        summaryServiceImpl.setRequery("The content of the link **needs to be summarized.");
        SummaryConfig config = new SummaryConfig();
        config.setExpired(1000);
        config.setCondition("NEXT");
        config.setDynamic("DYNAMIC");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(1000L);
        List<History> histories = new ArrayList<>();
        History history = new History();
        history.setCreated(10086L);
        history.setRole(History.ROLE_ASSISTANT);
        history.setType(History.TYPE_ANSWER);
        history.setContent("long content to make query exceed maxSize=1");
        histories.add(history);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(workflowTask, "UNKNOWN", Integer.MAX_VALUE, false, null)).andReturn(histories).anyTimes();
        historyStore.store(EasyMock.anyObject(), EasyMock.anyObject(), EasyMock.<List<HistoryPair>>anyObject(), EasyMock.eq(1000), EasyMock.anyObject());
        EasyMock.expectLastCall();
        historyStore.clear(workflowTask, Arrays.asList("UNKNOWN"), false, -10086L);
        EasyMock.expectLastCall();
        EasyMock.replay(historyStore);
        summaryServiceImpl.setNotifierService(notifier);
        summaryServiceImpl.setHistoryStore(historyStore);
        summaryServiceImpl.setMimeType("text/plain");
        Assert.assertEquals("FALSE", summaryServiceImpl.summarize(config, workflowTask).getContent());
        Assert.assertNotNull(captured.get());
        Assert.assertEquals(1, captured.get().size());
        Assert.assertEquals("text/plain", captured.get().get(0).getType());
        Assert.assertEquals(networkUrl, captured.get().get(0).getData());
        EasyMock.verify(fileStoreMock, historyStore);
    }

    /** 覆盖 summarize：shouldStore 为 true 但 buildSummaryPart 的 pairs 为空时不调用 store/clear。 */
    @Test
    public void testSummarizeShouldStoreButPairsEmpty() throws Exception {
        SummaryServiceImpl summaryServiceImpl = new SummaryServiceImpl() {
            @Override
            public Boolean allowed(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories, String append) throws Exception {
                return true;
            }

            @Override
            protected List<HistoryPair> buildPairs(SummaryConfig summaryConfig, WorkflowTask workTask, String content) throws Exception {
                return Collections.emptyList();
            }
        };
        summaryServiceImpl.setMaxSize(Integer.MAX_VALUE);
        SummaryConfig config = new SummaryConfig();
        config.setExpired(1000);
        config.setCondition("NEXT");
        config.setDynamic("DYNAMIC");
        config.setStore(true);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(1000L);
        List<History> histories = new ArrayList<>();
        History hist = new History();
        hist.setRole(History.ROLE_ASSISTANT);
        hist.setType(History.TYPE_ANSWER);
        histories.add(hist);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(workflowTask, "UNKNOWN", Integer.MAX_VALUE, false, null)).andReturn(histories).anyTimes();
        EasyMock.replay(historyStore);
        summaryServiceImpl.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("FALSE"));
        summaryServiceImpl.setHistoryStore(historyStore);
        summaryServiceImpl.setStore("");
        SummaryPart part = summaryServiceImpl.summarize(config, workflowTask);
        Assert.assertNotNull(part);
        Assert.assertEquals("FALSE", part.getContent());
        Assert.assertNotNull(part.getPairs());
        Assert.assertTrue(part.getPairs().isEmpty());
        EasyMock.verify(historyStore);
    }

    /** 用于测试 protected buildMediaContext 的子类 */
    private static class SummaryServiceImplForBuildMediaContextTest extends SummaryServiceImpl {
        List<MediaContext> callBuildMediaContext(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories, String requery) throws Exception {
            return buildMediaContext(summaryConfig, workTask, histories, requery);
        }
    }

    /** 覆盖 buildMediaContext：maxSize > size 时返回 null（不落库、不打网络） */
    @Test
    public void testBuildMediaContextReturnsNullWhenMaxSizeGreaterThanSize() throws Exception {
        ai.open.right.workflow.flow.file.FileStore fileStoreMock = EasyMock.createMock(ai.open.right.workflow.flow.file.FileStore.class);
        EasyMock.expect(fileStoreMock.supportNetwork()).andReturn(true).anyTimes();
        EasyMock.replay(fileStoreMock);

        DefStore defStoreStub = new DefStore() {
            @Override
            public Boolean supportFunction(String name) throws Exception {
                return true;
            }

            @Override
            public ai.open.right.workflow.flow.file.FileStore fetchStore(String name) throws Exception {
                return fileStoreMock;
            }

            @Override
            public String store(byte[] bytes, String suffix, WorkflowTask workTask) throws Exception {
                return "https://example.com/x.json";
            }
        };

        SummaryServiceImplForBuildMediaContextTest service = new SummaryServiceImplForBuildMediaContextTest();
        service.setDefStore(defStoreStub);
        service.setStore("file.store.s3");
        service.setMaxSize(1000);
        SummaryConfig config = new SummaryConfig();
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        List<History> histories = new ArrayList<>();
        String shortRequery = "x";
        List<MediaContext> result = service.callBuildMediaContext(config, workTask, histories, shortRequery);
        Assert.assertNull("maxSize(1000) > size(1) should return null", result);
        EasyMock.verify(fileStoreMock);
    }

    /** 覆盖 buildMediaContext：fileStore.supportNetwork() 为 false 时返回 null */
    @Test
    public void testBuildMediaContextReturnsNullWhenStoreNotSupportNetwork() throws Exception {
        ai.open.right.workflow.flow.file.FileStore fileStoreMock = EasyMock.createMock(ai.open.right.workflow.flow.file.FileStore.class);
        EasyMock.expect(fileStoreMock.supportNetwork()).andReturn(false).anyTimes();
        EasyMock.replay(fileStoreMock);

        DefStore defStoreStub = new DefStore() {
            @Override
            public Boolean supportFunction(String name) throws Exception {
                return true;
            }

            @Override
            public ai.open.right.workflow.flow.file.FileStore fetchStore(String name) throws Exception {
                return fileStoreMock;
            }

            @Override
            public String store(byte[] bytes, String suffix, WorkflowTask workTask) throws Exception {
                return "https://example.com/x.json";
            }
        };

        SummaryServiceImplForBuildMediaContextTest service = new SummaryServiceImplForBuildMediaContextTest();
        service.setDefStore(defStoreStub);
        service.setStore("file.store.s3");
        service.setMaxSize(1);
        SummaryConfig config = new SummaryConfig();
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        List<History> histories = new ArrayList<>();
        String longRequery = StringUtils.repeat("a", 2000);
        List<MediaContext> result = service.callBuildMediaContext(config, workTask, histories, longRequery);
        Assert.assertNull("fileStore.supportNetwork() is false should return null", result);
        EasyMock.verify(fileStoreMock);
    }

    /** 覆盖 buildMediaContext：summaryConfig.getBase64() == true 时，data 为 requery 的 Base64 编码 */
    @Test
    public void testBuildMediaContextBase64True() throws Exception {
        ai.open.right.workflow.flow.file.FileStore fileStoreMock = EasyMock.createMock(ai.open.right.workflow.flow.file.FileStore.class);
        EasyMock.expect(fileStoreMock.supportNetwork()).andReturn(true).anyTimes();
        EasyMock.replay(fileStoreMock);

        DefStore defStoreStub = new DefStore() {
            @Override
            public Boolean supportFunction(String name) throws Exception {
                return true;
            }
            @Override
            public ai.open.right.workflow.flow.file.FileStore fetchStore(String name) throws Exception {
                return fileStoreMock;
            }
            @Override
            public String store(byte[] bytes, String suffix, WorkflowTask workTask) throws Exception {
                throw new IllegalStateException("Should not be called when base64 is true");
            }
        };

        SummaryServiceImplForBuildMediaContextTest service = new SummaryServiceImplForBuildMediaContextTest();
        service.setDefStore(defStoreStub);
        service.setStore("dummy");
        service.setMaxSize(5);
        service.setMimeType("text/plain");

        SummaryConfig config = new SummaryConfig();
        config.setBase64(true);
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        List<History> histories = new ArrayList<>();
        String requery = "hello world";
        
        List<MediaContext> result = service.callBuildMediaContext(config, workTask, histories, requery);
        Assert.assertNotNull(result);
        Assert.assertEquals(1, result.size());
        
        String expectedBase64 = java.util.Base64.getEncoder().encodeToString(requery.getBytes(java.nio.charset.StandardCharsets.UTF_8));
        Assert.assertEquals(expectedBase64, result.get(0).getData());
        Assert.assertEquals(MediaContext.PREFIX_INLINE + "text/plain", result.get(0).getType());
        
        EasyMock.verify(fileStoreMock);
    }

    /** 覆盖 buildMediaContext：summaryConfig.getBase64() == false 时，data 为 defStore.store 返回的值 */
    @Test
    public void testBuildMediaContextBase64False() throws Exception {
        ai.open.right.workflow.flow.file.FileStore fileStoreMock = EasyMock.createMock(ai.open.right.workflow.flow.file.FileStore.class);
        EasyMock.expect(fileStoreMock.supportNetwork()).andReturn(true).anyTimes();
        EasyMock.replay(fileStoreMock);

        DefStore defStoreStub = new DefStore() {
            @Override
            public Boolean supportFunction(String name) throws Exception {
                return true;
            }
            @Override
            public ai.open.right.workflow.flow.file.FileStore fetchStore(String name) throws Exception {
                return fileStoreMock;
            }
            @Override
            public String store(byte[] bytes, String suffix, WorkflowTask workTask) throws Exception {
                return "https://example.com/stored.json";
            }
        };

        SummaryServiceImplForBuildMediaContextTest service = new SummaryServiceImplForBuildMediaContextTest();
        service.setDefStore(defStoreStub);
        service.setStore("dummy");
        service.setMaxSize(5);
        service.setMimeType("text/plain");

        SummaryConfig config = new SummaryConfig();
        config.setBase64(false);
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        List<History> histories = new ArrayList<>();
        String requery = "hello world";
        
        List<MediaContext> result = service.callBuildMediaContext(config, workTask, histories, requery);
        Assert.assertNotNull(result);
        Assert.assertEquals(1, result.size());
        
        Assert.assertEquals("https://example.com/stored.json", result.get(0).getData());
        Assert.assertEquals("text/plain", result.get(0).getType());
        
        EasyMock.verify(fileStoreMock);
    }

    /** 用于测试 protected buildSummaryPart 的子类 */
    private static class SummaryServiceImplForBuildSummaryPartTest extends SummaryServiceImpl {
        SummaryPart callBuildSummaryPart(SummaryConfig summaryConfig, WorkflowTask workTask, String content) throws Exception {
            return buildSummaryPart(summaryConfig, workTask, content);
        }
    }

    @Test(expected = IllegalArgumentException.class)
    public void testBuildSummaryPartEmptyContent() throws Exception {
        SummaryServiceImplForBuildSummaryPartTest service = new SummaryServiceImplForBuildSummaryPartTest();
        SummaryConfig config = new SummaryConfig();
        service.callBuildSummaryPart(config, defaultWorkflowTask(), "");
    }

    @Test(expected = IllegalArgumentException.class)
    public void testBuildSummaryPartNullContent() throws Exception {
        SummaryServiceImplForBuildSummaryPartTest service = new SummaryServiceImplForBuildSummaryPartTest();
        SummaryConfig config = new SummaryConfig();
        service.callBuildSummaryPart(config, defaultWorkflowTask(), null);
    }

    /** getSplit 为 true 时，pairs 由 buildPairs(summaryConfig, workTask, content) 解析得到 */
    @Test
    public void testBuildSummaryPartSplitTrue() throws Exception {
        SummaryServiceImplForBuildSummaryPartTest service = new SummaryServiceImplForBuildSummaryPartTest();
        SummaryConfig config = new SummaryConfig();
        config.setSplit(true);
        SummaryPart part = service.callBuildSummaryPart(config, defaultWorkflowTask(), "Q=A");
        Assert.assertNotNull(part);
        Assert.assertEquals("Q=A", part.getContent());
        Assert.assertNotNull(part.getPairs());
        Assert.assertEquals(1, part.getPairs().size());
        Assert.assertEquals("Q", part.getPairs().get(0).getQuery());
        Assert.assertEquals("A", part.getPairs().get(0).getAnswer());
    }

    /** getSplit 为 false 时，pairs 为 null */
    @Test
    public void testBuildSummaryPartSplitFalse() throws Exception {
        SummaryServiceImplForBuildSummaryPartTest service = new SummaryServiceImplForBuildSummaryPartTest();
        SummaryConfig config = new SummaryConfig();
        config.setSplit(false);
        SummaryPart part = service.callBuildSummaryPart(config, defaultWorkflowTask(), "any content");
        Assert.assertNotNull(part);
        Assert.assertEquals("any content", part.getContent());
        Assert.assertNull(part.getPairs());
    }

    /** getSplit 未设置时默认为 true，pairs 由 buildPairs 解析 */
    @Test
    public void testBuildSummaryPartSplitDefaultTrue() throws Exception {
        SummaryServiceImplForBuildSummaryPartTest service = new SummaryServiceImplForBuildSummaryPartTest();
        SummaryConfig config = new SummaryConfig();
        SummaryPart part = service.callBuildSummaryPart(config, defaultWorkflowTask(), "key=value");
        Assert.assertNotNull(part);
        Assert.assertEquals("key=value", part.getContent());
        Assert.assertNotNull(part.getPairs());
        Assert.assertEquals(1, part.getPairs().size());
    }

    /** buildPairs(SummaryConfig, WorkflowTask, String)：JSON 解析路径，pairs 的 chat、conversation 来自 workTask */
    @Test
    public void testBuildPairs_jsonSetsChatAndConversation() throws Exception {
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        SummaryServiceImpl service = new SummaryServiceImpl();
        String content = "[{\"query\":\"q1\",\"answer\":\"a1\"}]";
        List<HistoryPair> pairs = service.buildPairs(defaultSummaryConfig(), workTask, content);
        Assert.assertEquals(1, pairs.size());
        Assert.assertEquals("q1", pairs.get(0).getQuery());
        Assert.assertEquals("a1", pairs.get(0).getAnswer());
        Assert.assertEquals(workTask.getChat(), pairs.get(0).getChat());
        Assert.assertEquals(workTask.getConversation(), pairs.get(0).getConversation());
    }

    /** buildPairs(SummaryConfig, WorkflowTask, String)：key=value 解析路径，pair 的 chat、conversation 来自 workTask */
    @Test
    public void testBuildPairs_keyValueSetsChatAndConversation() throws Exception {
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        SummaryServiceImpl service = new SummaryServiceImpl();
        List<HistoryPair> pairs = service.buildPairs(defaultSummaryConfig(), workTask, "queryKey=answerVal");
        Assert.assertEquals(1, pairs.size());
        Assert.assertEquals("queryKey", pairs.get(0).getQuery());
        Assert.assertEquals("answerVal", pairs.get(0).getAnswer());
        Assert.assertEquals(workTask.getChat(), pairs.get(0).getChat());
        Assert.assertEquals(workTask.getConversation(), pairs.get(0).getConversation());
    }

    /** buildPairs：JSON 多对，每对均带上 workTask 的 chat 与 conversation */
    @Test
    public void testBuildPairs_jsonMultiplePairsChatConversation() throws Exception {
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        SummaryServiceImpl service = new SummaryServiceImpl();
        String content = "[{\"query\":\"q1\",\"answer\":\"a1\"},{\"query\":\"q2\",\"answer\":\"a2\"}]";
        List<HistoryPair> pairs = service.buildPairs(defaultSummaryConfig(), workTask, content);
        Assert.assertEquals(2, pairs.size());
        for (HistoryPair p : pairs) {
            Assert.assertEquals(workTask.getChat(), p.getChat());
            Assert.assertEquals(workTask.getConversation(), p.getConversation());
        }
    }

    /** 用于测试 protected selectHistories 的子类 */
    private static class SummaryServiceImplForSelectHistoriesTest extends SummaryServiceImpl {
        List<History> callSelectHistories(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories) throws Exception {
            return selectHistories(summaryConfig, workTask, histories);
        }
    }

    /** 用于测试 protected dropOnFailed 的子类 */
    private static class SummaryServiceImplForDropOnFailedTest extends SummaryServiceImpl {
        void callDropOnFailed(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories) throws Exception {
            dropOnFailed(summaryConfig, workTask, histories);
        }
    }

    @Test
    public void testSelectHistories_emptyOrNull_returnsAsIs() throws Exception {
        SummaryServiceImplForSelectHistoriesTest service = new SummaryServiceImplForSelectHistoriesTest();
        SummaryConfig config = new SummaryConfig();
        config.setIncludeFunCall(false);
        List<History> empty = new ArrayList<>();
        Assert.assertSame(empty, service.callSelectHistories(config, defaultWorkflowTask(), empty));
        Assert.assertNull(service.callSelectHistories(config, defaultWorkflowTask(), null));
    }

    @Test
    public void testSelectHistories_includeFunCallTrue_returnsSameList() throws Exception {
        SummaryServiceImplForSelectHistoriesTest service = new SummaryServiceImplForSelectHistoriesTest();
        SummaryConfig config = new SummaryConfig();
        config.setIncludeFunCall(true);
        List<History> histories = new ArrayList<>();
        History chat = new History();
        chat.setFunction(History.FUN_CHAT);
        History funCall = new History();
        funCall.setFunction(History.FUN_FUNCALL);
        histories.add(chat);
        histories.add(funCall);
        List<History> result = service.callSelectHistories(config, defaultWorkflowTask(), histories);
        Assert.assertSame(histories, result);
        Assert.assertEquals(2, result.size());
    }

    @Test
    public void testSelectHistories_includeFunCallFalse_filtersOutFunCall() throws Exception {
        SummaryServiceImplForSelectHistoriesTest service = new SummaryServiceImplForSelectHistoriesTest();
        SummaryConfig config = new SummaryConfig();
        config.setIncludeFunCall(false);
        List<History> histories = new ArrayList<>();
        History chat1 = new History();
        chat1.setFunction(History.FUN_CHAT);
        History funCall = new History();
        funCall.setFunction(History.FUN_FUNCALL);
        History chat2 = new History();
        chat2.setFunction(History.FUN_CHAT);
        histories.add(chat1);
        histories.add(funCall);
        histories.add(chat2);
        List<History> result = service.callSelectHistories(config, defaultWorkflowTask(), histories);
        Assert.assertNotSame(histories, result);
        Assert.assertEquals(2, result.size());
        Assert.assertEquals(chat1, result.get(0));
        Assert.assertEquals(chat2, result.get(1));
    }

    @Test
    public void testSelectHistories_includeFunCallFalse_allFunCall_returnsEmpty() throws Exception {
        SummaryServiceImplForSelectHistoriesTest service = new SummaryServiceImplForSelectHistoriesTest();
        SummaryConfig config = new SummaryConfig();
        config.setIncludeFunCall(false);
        List<History> histories = new ArrayList<>();
        History funCall = new History();
        funCall.setFunction(History.FUN_FUNCALL);
        histories.add(funCall);
        List<History> result = service.callSelectHistories(config, defaultWorkflowTask(), histories);
        Assert.assertTrue(result.isEmpty());
    }

    @Test
    public void testDropOnFailed_whenDropOnFailedFalse_doesNotClear() throws Exception {
        SummaryServiceImplForDropOnFailedTest service = new SummaryServiceImplForDropOnFailedTest();
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.replay(historyStore);
        service.setHistoryStore(historyStore);
        SummaryConfig config = new SummaryConfig();
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(100L);
        History h = new History();
        h.setCreated(50L);
        workTask.setHistories(Collections.singletonList(h));
        service.callDropOnFailed(config, workTask, workTask.getHistories());
        EasyMock.verify(historyStore);
    }

    @Test
    public void testDropOnFailed_whenDropOnFailedTrue_callsClearWithNegatedLastTimeline() throws Exception {
        SummaryServiceImplForDropOnFailedTest service = new SummaryServiceImplForDropOnFailedTest();
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        SummaryConfig config = new SummaryConfig();
        config.setDropOnFailed(true);
        config.setDesc(true);
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(9999L);
        History h = new History();
        h.setCreated(3000L);
        workTask.setHistories(Collections.singletonList(h));
        historyStore.clear(workTask, Arrays.asList("UNKNOWN"), true, -3000L);
        EasyMock.expectLastCall();
        EasyMock.replay(historyStore);
        service.setHistoryStore(historyStore);
        service.callDropOnFailed(config, workTask, workTask.getHistories());
        EasyMock.verify(historyStore);
    }

    /** histories 非 null 时 buildLastTimeline 以该列表为准（与 workTask 上记忆可不一致） */
    @Test
    public void testDropOnFailed_whenHistoriesArgNonNull_usesArgForLastTimeline() throws Exception {
        SummaryServiceImplForDropOnFailedTest service = new SummaryServiceImplForDropOnFailedTest();
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        SummaryConfig config = new SummaryConfig();
        config.setDropOnFailed(true);
        config.setDesc(false);
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(9999L);
        workTask.setHistories(Collections.emptyList());
        List<History> argHistories = new ArrayList<>();
        History h = new History();
        h.setCreated(4000L);
        argHistories.add(h);
        historyStore.clear(workTask, Arrays.asList("UNKNOWN"), false, -4000L);
        EasyMock.expectLastCall();
        EasyMock.replay(historyStore);
        service.setHistoryStore(historyStore);
        service.callDropOnFailed(config, workTask, argHistories);
        EasyMock.verify(historyStore);
    }

    /** histories 为 null 时回退到 workTask.getHistories() 计算时间线 */
    @Test
    public void testDropOnFailed_whenHistoriesArgNull_usesWorkTaskHistories() throws Exception {
        SummaryServiceImplForDropOnFailedTest service = new SummaryServiceImplForDropOnFailedTest();
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        SummaryConfig config = new SummaryConfig();
        config.setDropOnFailed(true);
        config.setDesc(false);
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(12000L);
        History wh = new History();
        wh.setCreated(77L);
        workTask.setHistories(new ArrayList<>(Collections.singletonList(wh)));
        historyStore.clear(workTask, Arrays.asList("UNKNOWN"), false, -77L);
        EasyMock.expectLastCall();
        EasyMock.replay(historyStore);
        service.setHistoryStore(historyStore);
        service.callDropOnFailed(config, workTask, null);
        EasyMock.verify(historyStore);
    }

    /** clear 抛错时被内部 catch 吞掉并走 WorkflowException.dolog，不再向外抛 */
    @Test
    public void testDropOnFailed_whenClearThrows_swallowsAndDoesNotPropagate() throws Exception {
        SummaryServiceImplForDropOnFailedTest service = new SummaryServiceImplForDropOnFailedTest();
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        SummaryConfig config = new SummaryConfig();
        config.setDropOnFailed(true);
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(1L);
        List<History> list = new ArrayList<>();
        History h = new History();
        h.setCreated(10L);
        list.add(h);
        historyStore.clear(workTask, Arrays.asList("UNKNOWN"), false, -10L);
        EasyMock.expectLastCall().andThrow(new RuntimeException("clear-boom"));
        EasyMock.replay(historyStore);
        service.setHistoryStore(historyStore);
        service.callDropOnFailed(config, workTask, list);
        EasyMock.verify(historyStore);
    }

    /**
     * summarize 异常路径：catch 中调用 dropOnFailed(summaryConfig, workTask, histories)；
     * 原异常仍会抛出；dropOnFailed 内部 clear 失败时仅 dolog，不掩盖外层异常。
     */
    @Test
    public void testSummarize_exceptionPath_dropOnFailedTrue_clearsAndRethrows() throws Exception {
        SummaryConfig summaryConfig = new SummaryConfig();
        summaryConfig.setDropOnFailed(true);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(20000L);
        History onTask = new History();
        onTask.setCreated(888L);
        workflowTask.setHistories(new ArrayList<>(Collections.singletonList(onTask)));

        List<History> restored = new ArrayList<>();
        History rh = new History();
        rh.setCreated(1L);
        restored.add(rh);

        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(workflowTask, "UNKNOWN", Integer.MAX_VALUE, false, null)).andReturn(restored).anyTimes();
        // catch 传入的 histories 为 restore 后经 selectHistories/updateHistories 的结果，时间线取自 rh.created=1
        historyStore.clear(workflowTask, Arrays.asList("UNKNOWN"), false, -1L);
        EasyMock.expectLastCall();
        EasyMock.replay(historyStore);

        SummaryServiceImpl impl = new SummaryServiceImpl() {
            @Override
            protected String buildQuery(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories, String append) throws Exception {
                throw new RuntimeException("summary-fail");
            }
        };
        impl.setHistoryStore(historyStore);

        try {
            impl.summarize(summaryConfig, workflowTask);
            Assert.fail("expected RuntimeException");
        } catch (RuntimeException e) {
            Assert.assertEquals("summary-fail", e.getMessage());
        }
        EasyMock.verify(historyStore);
    }

    @Test
    public void testSummarize_exceptionPath_dropOnFailedFalse_noClear() throws Exception {
        SummaryConfig summaryConfig = new SummaryConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(20000L);
        History onTask = new History();
        onTask.setCreated(1L);
        workflowTask.setHistories(new ArrayList<>(Collections.singletonList(onTask)));

        List<History> restored = new ArrayList<>();
        restored.add(new History());

        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(workflowTask, "UNKNOWN", Integer.MAX_VALUE, false, null)).andReturn(restored).anyTimes();
        EasyMock.replay(historyStore);

        SummaryServiceImpl impl = new SummaryServiceImpl() {
            @Override
            protected String buildQuery(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories, String append) throws Exception {
                throw new RuntimeException("no-clear");
            }
        };
        impl.setHistoryStore(historyStore);

        try {
            impl.summarize(summaryConfig, workflowTask);
            Assert.fail("expected RuntimeException");
        } catch (RuntimeException e) {
            Assert.assertEquals("no-clear", e.getMessage());
        }
        EasyMock.verify(historyStore);
    }

    /** summarize 失败时若 dropOnFailed 内 clear 抛错，仍向上抛出 summarize 链路上的原异常 */
    @Test
    public void testSummarize_exceptionPath_dropOnFailedClearThrows_stillRethrowsOriginal() throws Exception {
        SummaryConfig summaryConfig = new SummaryConfig();
        summaryConfig.setDropOnFailed(true);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(20000L);

        List<History> restored = new ArrayList<>();
        History rh = new History();
        rh.setCreated(2L);
        restored.add(rh);

        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(workflowTask, "UNKNOWN", Integer.MAX_VALUE, false, null)).andReturn(restored).anyTimes();
        historyStore.clear(workflowTask, Arrays.asList("UNKNOWN"), false, -2L);
        EasyMock.expectLastCall().andThrow(new RuntimeException("clear-fail"));
        EasyMock.replay(historyStore);

        SummaryServiceImpl impl = new SummaryServiceImpl() {
            @Override
            protected String buildQuery(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories, String append) throws Exception {
                throw new RuntimeException("upstream-fail");
            }
        };
        impl.setHistoryStore(historyStore);

        try {
            impl.summarize(summaryConfig, workflowTask);
            Assert.fail("expected RuntimeException");
        } catch (RuntimeException e) {
            Assert.assertEquals("upstream-fail", e.getMessage());
        }
        EasyMock.verify(historyStore);
    }

    /** 子类：暴露 protected updateHistories */
    private static class SummaryServiceImplForUpdateHistoriesTest extends SummaryServiceImpl {
        List<History> callUpdateHistories(SummaryConfig summaryConfig, List<History> histories) throws Exception {
            return updateHistories(summaryConfig, histories);
        }
    }

    @Test
    public void testUpdateHistories_includeReasonFalse_clearsReasoning() throws Exception {
        SummaryServiceImplForUpdateHistoriesTest service = new SummaryServiceImplForUpdateHistoriesTest();
        SummaryConfig config = new SummaryConfig();
        config.setIncludeReason(false);
        History h = new History();
        h.setReason("keep-or-drop");
        h.setSource("src");
        List<History> list = new ArrayList<>(Collections.singletonList(h));
        Assert.assertSame(list, service.callUpdateHistories(config, list));
        Assert.assertNull(h.getReason());
        Assert.assertNull(h.getSource());
    }

    @Test
    public void testUpdateHistories_includeReasonTrue_preservesReasoning() throws Exception {
        SummaryServiceImplForUpdateHistoriesTest service = new SummaryServiceImplForUpdateHistoriesTest();
        SummaryConfig config = new SummaryConfig();
        config.setIncludeReason(true);
        History h = new History();
        h.setReason("r1");
        h.setSource("s1");
        service.callUpdateHistories(config, Collections.singletonList(h));
        Assert.assertEquals("r1", h.getReason());
        Assert.assertNull(h.getSource());
    }

    @Test
    public void testUpdateHistories_includeReasonDefault_preservesReasoning() throws Exception {
        SummaryServiceImplForUpdateHistoriesTest service = new SummaryServiceImplForUpdateHistoriesTest();
        SummaryConfig config = new SummaryConfig();
        History h = new History();
        h.setReason("default-keep");
        service.callUpdateHistories(config, Collections.singletonList(h));
        Assert.assertEquals("default-keep", h.getReason());
    }

    /**
     * summarize 重载委托：两参 List 版本传入 append 为 ""（与四参一致）。
     */
    @Test
    public void testSummarize_overloadListHistories_delegatesEmptyAppend() throws Exception {
        final List<String> capturedAppends = new CopyOnWriteArrayList<>();
        SummaryConfig summaryConfig = new SummaryConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        SummaryServiceImpl impl = new SummaryServiceImpl() {
            @Override
            public SummaryPart summarize(SummaryConfig c, WorkflowTask t, List<History> h, String append) throws Exception {
                capturedAppends.add(append);
                return super.summarize(c, t, h, append);
            }
        };
        impl.summarize(summaryConfig, workflowTask, new ArrayList<>());
        Assert.assertEquals(Collections.singletonList(""), capturedAppends);
    }

    /**
     * summarize 重载委托：两参 task 经 restore 后，四参 append 为 ""。
     */
    @Test
    public void testSummarize_overloadTaskOnly_delegatesEmptyAppend() throws Exception {
        final List<String> capturedAppends = new CopyOnWriteArrayList<>();
        SummaryConfig summaryConfig = new SummaryConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(workflowTask, "UNKNOWN", Integer.MAX_VALUE, false, null)).andReturn(new ArrayList<>()).anyTimes();
        EasyMock.replay(historyStore);
        SummaryServiceImpl impl = new SummaryServiceImpl() {
            @Override
            public SummaryPart summarize(SummaryConfig c, WorkflowTask t, List<History> h, String append) throws Exception {
                capturedAppends.add(append);
                return super.summarize(c, t, h, append);
            }
        };
        impl.setHistoryStore(historyStore);
        Assert.assertNull(impl.summarize(summaryConfig, workflowTask));
        Assert.assertEquals(Collections.singletonList(""), capturedAppends);
        EasyMock.verify(historyStore);
    }

    /**
     * summarize 重载委托：task + append 将 append 原样传入四参。
     */
    @Test
    public void testSummarize_overloadTaskAndAppend_passesAppend() throws Exception {
        final List<String> capturedAppends = new CopyOnWriteArrayList<>();
        SummaryConfig summaryConfig = new SummaryConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(workflowTask, "UNKNOWN", Integer.MAX_VALUE, false, null)).andReturn(new ArrayList<>()).anyTimes();
        EasyMock.replay(historyStore);
        SummaryServiceImpl impl = new SummaryServiceImpl() {
            @Override
            public SummaryPart summarize(SummaryConfig c, WorkflowTask t, List<History> h, String append) throws Exception {
                capturedAppends.add(append);
                return super.summarize(c, t, h, append);
            }
        };
        impl.setHistoryStore(historyStore);
        Assert.assertNull(impl.summarize(summaryConfig, workflowTask, "tail"));
        Assert.assertEquals(Collections.singletonList("tail"), capturedAppends);
        EasyMock.verify(historyStore);
    }

    /**
     * 四参 summarize：显式传入 histories 时同样经过 updateHistories（includeReason=false 时清空 reasoning）。
     */
    @Test
    public void testSummarize_fourArg_appliesUpdateHistoriesIncludeReasonFalse() throws Exception {
        SummaryConfig summaryConfig = new SummaryConfig();
        summaryConfig.setIncludeReason(false);
        summaryConfig.setDynamic("DYNAMIC");
        summaryConfig.setStore(false);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        History h = new History();
        h.setRole(History.ROLE_ASSISTANT);
        h.setType(History.TYPE_ANSWER);
        h.setReason("strip-me");
        List<History> histories = new ArrayList<>(Collections.singletonList(h));
        SummaryServiceImpl impl = new SummaryServiceImpl() {
            @Override
            protected String buildQuery(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories, String append) throws Exception {
                Assert.assertNull(histories.get(0).getReason());
                return super.buildQuery(summaryConfig, workTask, histories, append);
            }

            @Override
            protected List<HistoryPair> buildPairs(SummaryConfig summaryConfig, WorkflowTask workTask, String content) throws Exception {
                return List.of(new HistoryPair());
            }
        };
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.replay(historyStore);
        impl.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("X"));
        impl.setHistoryStore(historyStore);
        impl.setMaxSize(Integer.MAX_VALUE);
        impl.setStore("");
        Assert.assertNotNull(impl.summarize(summaryConfig, workflowTask, histories, ""));
        EasyMock.verify(historyStore);
    }
}
