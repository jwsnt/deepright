package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.llm.store.Dimension;
import ai.open.right.workflow.flow.llm.store.history.*;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.protocol.Protocol;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.summary.SummaryConfig;
import ai.open.right.workflow.flow.summary.SummaryPart;
import ai.open.right.workflow.flow.summary.SummaryService;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class HistoryAssistantTest {

    @Test
    public void testWithSummary() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        HistoryConfig historyConfig = new HistoryConfig();
        historyConfig.setSummaryConfig(new SummaryConfig());
        workflowConfig.setHistoryConfig(historyConfig);
        SummaryService summaryService = EasyMock.createMock(SummaryService.class);
        HistoryAssistant historyAssistant = new HistoryAssistant() {
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String protocol, String content) throws Exception {

            }
        };
        historyAssistant.setSummaryService(summaryService);
        historyAssistant.execute(workflowConfig, workflowTask);
    }

    /**
     * 覆盖 summarize 中 summaryPart 非 null 时调用 chainOr2Endpoint(workflowConfig, workTask, Protocol.CHAT, JsonUtils.write(summaryPart))。
     */
    @Test
    public void testSummarizeCallsChainOr2EndpointWhenSummaryPartNonNull() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        HistoryConfig historyConfig = new HistoryConfig();
        historyConfig.setSummaryConfig(new SummaryConfig());
        workflowConfig.setHistoryConfig(historyConfig);
        SummaryPart summaryPart = SummaryPart.builder().content("summary-content").build();
        SummaryService summaryService = EasyMock.createMock(SummaryService.class);
        EasyMock.expect(summaryService.summarize(EasyMock.anyObject(SummaryConfig.class), EasyMock.eq(workflowTask))).andReturn(summaryPart).once();
        EasyMock.replay(summaryService);
        final String[] capturedContent = new String[1];
        HistoryAssistant historyAssistant = new HistoryAssistant() {
            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String protocol, String content) throws Exception {
                Assert.assertEquals(Protocol.CHAT, protocol);
                capturedContent[0] = content;
            }
        };
        historyAssistant.setSummaryService(summaryService);
        historyAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(summaryService);
        Assert.assertEquals(JsonUtils.write(summaryPart), capturedContent[0]);
    }


    @Test(expected = IllegalArgumentException.class)
    public void testWithException() throws Exception {
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        notifierManager.notify(EasyMock.anyObject(Segment.class), EasyMock.anyObject(RedirectContext.class), EasyMock.anyObject(NotifierWriteBack.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(notifierManager);
        HistoryAssistant internalAssistant = new HistoryAssistant();
        internalAssistant.setNotifierService(notifierManager);
        internalAssistant.execute(new WorkflowConfig(), ObjectBuilder.buildWorkflowTask());
        EasyMock.verify(notifierManager);
    }

    @Test
    public void testWithClean1() throws Exception {
        HistoryConfig historyConfig = new HistoryConfig();
        HistoryClearConfig historyClearConfig = new HistoryClearConfig();
        historyConfig.setClearConfig(historyClearConfig);
        historyClearConfig.setOffset(1000);
        historyClearConfig.setRepositories(Arrays.asList("WORKFLOW"));
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setHistoryConfig(historyConfig);
        HistoryAssistant historyAssistant = new HistoryAssistant() {
            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, Map<String, Object> metadata, List<MediaContext> mediaContext, String protocol, String content, Integer code) throws Exception {

            }
        };
        historyAssistant.setHistoryStore(new HistoryStore() {
            @Override
            public List<History> restore(Dimension dimension, String scene, Integer nums) {
                return null;
            }

            @Override
            public void clear(Dimension dimension, List<String> repositories, Boolean desc, Long now) throws Exception {
                Assert.assertEquals("UNKNOWN", dimension.getBiz());
                Assert.assertEquals("UNKNOWN", dimension.getChat());
                Assert.assertEquals("UNKNOWN", dimension.getDevice());
                Assert.assertEquals("[WORKFLOW]", repositories.toString());
                Assert.assertEquals(Long.valueOf(-(10086 - 1000)), now);
            }

            @Override
            public void clear(Dimension dimension, List<String> repositories, Long now) throws Exception {
                this.clear(dimension, repositories, false, now);
            }

            @Override
            public void store(Dimension dimension, List<String> repositories, String query, String answer, String reasoning, Integer expire, Integer nums, Long now) throws Exception {

            }

            @Override
            public void store(Dimension dimension, List<String> repositories, String query, String answer, Integer expired, Integer nums, Long time) {
            }

            @Override
            public void store(Dimension dimension, List<String> repositories, List<HistoryPair> pairs, Integer expire, Integer nums) throws Exception {

            }

            @Override
            public void store(Dimension dimension, List<String> repositories, HistoryPair pair, Integer expire, Integer nums) throws Exception {

            }

            @Override
            public List<History> restore(Dimension dimension, String scene, Integer nums, Boolean desc, Long now, Long offset) throws Exception {
                return List.of();
            }

            @Override
            public List<History> restore(Dimension dimension, String scene, Integer nums, Boolean desc, Long now) throws Exception {
                return null;
            }

            @Override
            public List<History> restore(Dimension dimension, String scene, Integer nums, Boolean desc) throws Exception {
                return null;
            }

            @Override
            public List<History> restore(Dimension dimension, String scene, Integer nums, Long now) throws Exception {
                return null;
            }
        });
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(10086L);
        historyAssistant.clear(workflowConfig, workflowTask);
    }

    @Test
    public void testWithClean2() throws Exception {
        HistoryConfig historyConfig = new HistoryConfig();
        HistoryClearConfig historyClearConfig = new HistoryClearConfig();
        historyConfig.setClearConfig(historyClearConfig);
        historyClearConfig.setOffset(1000);
        historyClearConfig.setRepositories(Arrays.asList("WORKFLOW"));
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setHistoryConfig(historyConfig);
        HistoryAssistant historyAssistant = new HistoryAssistant() {
            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, Map<String, Object> metadata, List<MediaContext> mediaContext, String protocol, String content, Integer code) throws Exception {

            }
        };
        historyAssistant.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect());
        historyAssistant.setHistoryStore(new HistoryStore() {
            @Override
            public List<History> restore(Dimension dimension, String scene, Integer nums) {
                return null;
            }

            @Override
            public void clear(Dimension dimension, List<String> repositories, Boolean desc, Long now) throws Exception {
                Assert.assertEquals("UNKNOWN", dimension.getBiz());
                Assert.assertEquals("UNKNOWN", dimension.getChat());
                Assert.assertEquals("UNKNOWN", dimension.getDevice());
                Assert.assertEquals("[WORKFLOW]", repositories.toString());
                Assert.assertEquals(Long.valueOf(-(10086 - 1000)), now);
            }

            @Override
            public void clear(Dimension dimension, List<String> repositories, Long now) throws Exception {
                this.clear(dimension, repositories, false, now);
            }

            @Override
            public void store(Dimension dimension, List<String> repositories, String query, String answer, String reasoning, Integer expire, Integer nums, Long now) throws Exception {

            }

            @Override
            public void store(Dimension dimension, List<String> repositories, String query, String answer, Integer expired, Integer nums, Long time) {
            }

            @Override
            public void store(Dimension dimension, List<String> repositories, List<HistoryPair> pairs, Integer expire, Integer nums) throws Exception {

            }

            @Override
            public void store(Dimension dimension, List<String> repositories, HistoryPair pair, Integer expire, Integer nums) throws Exception {

            }

            @Override
            public List<History> restore(Dimension dimension, String scene, Integer nums, Boolean desc, Long now, Long offset) throws Exception {
                return List.of();
            }

            @Override
            public List<History> restore(Dimension dimension, String scene, Integer nums, Boolean desc, Long now) throws Exception {
                return null;
            }

            @Override
            public List<History> restore(Dimension dimension, String scene, Integer nums, Boolean desc) throws Exception {
                return null;
            }

            @Override
            public List<History> restore(Dimension dimension, String scene, Integer nums, Long now) throws Exception {
                return null;
            }
        });
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(10086L);
        historyAssistant.execute(workflowConfig, workflowTask);
    }

    @Test
    public void testWithCleanWithScene() throws Exception {
        HistoryConfig historyConfig = new HistoryConfig();
        HistoryClearConfig historyClearConfig = new HistoryClearConfig();
        historyConfig.setClearConfig(historyClearConfig);
        historyClearConfig.setOffset(1000);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setHistoryConfig(historyConfig);
        HistoryAssistant historyAssistant = new HistoryAssistant() {
            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, Map<String, Object> metadata, List<MediaContext> mediaContext, String protocol, String content, Integer code) throws Exception {

            }
        };
        historyAssistant.setHistoryStore(new HistoryStore() {
            @Override
            public List<History> restore(Dimension dimension, String scene, Integer nums) {
                return null;
            }

            @Override
            public void clear(Dimension dimension, List<String> repositories, Boolean desc, Long now) throws Exception {
                Assert.assertEquals("UNKNOWN", dimension.getBiz());
                Assert.assertEquals("UNKNOWN", dimension.getChat());
                Assert.assertEquals("UNKNOWN", dimension.getDevice());
                Assert.assertEquals("[UNKNOWN]", repositories.toString());
                Assert.assertEquals(Long.valueOf(-(10086 - 1000)), now);
            }

            @Override
            public void clear(Dimension dimension, List<String> repositories, Long now) throws Exception {
                this.clear(dimension, repositories, false, now);
            }

            @Override
            public void store(Dimension dimension, List<String> repositories, String query, String answer, String reasoning, Integer expire, Integer nums, Long now) throws Exception {

            }

            @Override
            public void store(Dimension dimension, List<String> repositories, String query, String answer, Integer expired, Integer nums, Long time) {
            }

            @Override
            public void store(Dimension dimension, List<String> repositories, List<HistoryPair> pairs, Integer expire, Integer nums) throws Exception {

            }

            @Override
            public void store(Dimension dimension, List<String> repositories, HistoryPair pair, Integer expire, Integer nums) throws Exception {

            }

            @Override
            public List<History> restore(Dimension dimension, String scene, Integer nums, Boolean desc, Long now, Long offset) throws Exception {
                return List.of();
            }

            @Override
            public List<History> restore(Dimension dimension, String scene, Integer nums, Boolean desc, Long now) throws Exception {
                return null;
            }

            @Override
            public List<History> restore(Dimension dimension, String scene, Integer nums, Boolean desc) throws Exception {
                return null;
            }

            @Override
            public List<History> restore(Dimension dimension, String scene, Integer nums, Long now) throws Exception {
                return null;
            }
        });
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithTimestamp(10086L);
        historyAssistant.clear(workflowConfig, workflowTask);
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);
        Map<String, LLMQueryService> llmQueryServices = new HashMap<>();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        MediaConfig mediaConfig = new MediaConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        HistoryStore service1 = EasyMock.createMock(HistoryStore.class);
        SummaryService service2 = EasyMock.createMock(SummaryService.class);
        EasyMock.replay(service1, service2, notifierManager, signalFactory);
        HistoryAssistant.InitConfig assistant = new HistoryAssistant.InitConfig();
        assistant.setNotifierService(notifierManager);
        assistant.setLlmQueryService(llmQueryServices);
        assistant.setSignalFactory(signalFactory);
        assistant.setHistoryStore(service1);
        assistant.setSummaryService(service2);
        HistoryAssistant empty = assistant.historyAssistant();
        Assert.assertEquals(service1, empty.getHistoryStore());
        Assert.assertEquals(service2, empty.getSummaryService());
        Assert.assertEquals(empty.getLlmQueryService(), llmQueryServices);
        Assert.assertEquals(empty.getNotifierService(), notifierManager);
        Assert.assertEquals(empty.getSignalFactory(), signalFactory);
        EasyMock.verify(service1, service2, notifierManager, signalFactory);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = HistoryAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = HistoryAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testWithFetch() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        HistoryConfig historyConfig = new HistoryConfig();
        HistoryFetchConfig historyFetchConfig = new HistoryFetchConfig();
        historyFetchConfig.setScene("X");
        historyFetchConfig.setNums(1);
        historyConfig.setFetchConfig(historyFetchConfig);
        workflowConfig.setHistoryConfig(historyConfig);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(workflowTask, "X", 1)).andReturn(null).anyTimes();
        EasyMock.replay(historyStore);
        HistoryAssistant historyAssistant = new HistoryAssistant() {
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String protocol, String content) throws Exception {

            }
        };
        historyAssistant.setHistoryStore(historyStore);
        historyAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(historyStore);
    }

    @Test
    public void testWithFetchAndScene() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        HistoryConfig historyConfig = new HistoryConfig();
        HistoryFetchConfig historyFetchConfig = new HistoryFetchConfig();
        historyConfig.setFetchConfig(historyFetchConfig);
        workflowConfig.setHistoryConfig(historyConfig);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(workflowTask, "UNKNOWN", 100)).andReturn(null).anyTimes();
        EasyMock.replay(historyStore);
        HistoryAssistant historyAssistant = new HistoryAssistant() {
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String protocol, String content) throws Exception {

            }
        };
        historyAssistant.setHistoryStore(historyStore);
        historyAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(historyStore);
    }
}
