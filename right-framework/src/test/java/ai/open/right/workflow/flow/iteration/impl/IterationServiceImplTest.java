package ai.open.right.workflow.flow.iteration.impl;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.iteration.IterationConfig;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.track.TrackDimension;
import ai.open.right.workflow.flow.track.TrackFunCall;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.List;

public class IterationServiceImplTest {

    @Test
    public void testShouldContinueWithFalse1() throws Exception {
        IterationConfig iterationConfig = new IterationConfig();
        IterationServiceImpl iterationService = new IterationServiceImpl();
        iterationConfig.setProcessor("Processor");
        iterationService.setMaxTimes(10);
        iterationService.setTimeout(1000);
        Assert.assertFalse(iterationService.checkCondition(iterationConfig, ObjectBuilder.buildWorkflowTask(), new StringBuffer("HELLO"), "A", 0));
    }

    @Test
    public void testShouldContinueWithTrue() throws Exception {
        IterationConfig iterationConfig = new IterationConfig();
        iterationConfig.setCondition("CONDITION");
        IterationServiceImpl iterationService = new IterationServiceImpl();
        iterationConfig.setProcessor("Processor");
        iterationService.setMaxTimes(10);
        iterationService.setTimeout(1000);
        iterationService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("TRUE"));
        Assert.assertTrue(iterationService.checkCondition(iterationConfig, ObjectBuilder.buildWorkflowTask(), new StringBuffer("HELLO"), "A", 0));
    }

    @Test
    public void testShouldContinueWithFalse2() throws Exception {
        IterationConfig iterationConfig = new IterationConfig();
        iterationConfig.setCondition("CONDITION");
        IterationServiceImpl iterationService = new IterationServiceImpl();
        iterationConfig.setProcessor("Processor");
        iterationService.setMaxTimes(10);
        iterationService.setTimeout(1000);
        iterationService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("FALSE"));
        Assert.assertFalse(iterationService.checkCondition(iterationConfig, ObjectBuilder.buildWorkflowTask(), new StringBuffer("HELLO"), "A", 0));
    }

    @Test
    public void testIteratorWithOutCondition() throws Exception {
        IterationServiceImpl iterationService = new IterationServiceImpl() {
            @Override
            protected Boolean checkCondition(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer, Integer idx) throws Exception {
                return false;
            }
        };
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMConfig llmConfig = new LLMConfig();
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        historyStore.store(workflowTask, llmConfig.buildRepositories(workflowTask.getWorkflow()), "UNKNOWN", "OK123", llmConfig.getExpired(), llmConfig.getHistories(), workflowTask.getCreated());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(historyStore);
        iterationService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("OK123"));
        iterationService.setHistoryStore(historyStore);
        iterationService.setMaxTimes(10);
        iterationService.setTimeout(1000);
        iterationService.setMaxSize(10);
        IterationConfig iterationConfig = new IterationConfig();
        iterationConfig.setContainHistories(true);
        iterationConfig.setLlmConfig(new LLMConfig());
        iterationConfig.setProcessor("Processor");
        Assert.assertEquals("OK123", iterationService.iterate(iterationConfig, workflowTask));
        EasyMock.verify(historyStore);
    }

    @Test
    public void testIteratorWithConditionAndWithOutRefection() throws Exception {
        IterationServiceImpl iterationService = new IterationServiceImpl() {

            private int i = 0;

            @Override
            protected Boolean checkCondition(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer, Integer idx) throws Exception {
                return (i++) < 2;
            }
        };
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMConfig llmConfig = new LLMConfig();
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        historyStore.store(workflowTask, llmConfig.buildRepositories(workflowTask.getWorkflow()), "UNKNOWN", "OK123", llmConfig.getExpired(), llmConfig.getHistories(), workflowTask.getCreated());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(historyStore);
        iterationService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("OK123"));
        iterationService.setHistoryStore(historyStore);
        iterationService.setMaxTimes(10);
        iterationService.setTimeout(1000);
        iterationService.setMaxSize(10);
        IterationConfig iterationConfig = new IterationConfig();
        iterationConfig.setLlmConfig(llmConfig);
        iterationConfig.setContainHistories(true);
        iterationConfig.setProcessor("Processor");
        iterationConfig.setTimes(5);
        Assert.assertEquals("OK123", iterationService.iterate(iterationConfig, workflowTask));
        EasyMock.verify(historyStore);
    }

    @Test
    public void testIteratorWithConditionAndWithRefection() throws Exception {
        IterationServiceImpl iterationService = new IterationServiceImpl() {

            private int i = 0;

            @Override
            protected Boolean checkCondition(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer, Integer idx) throws Exception {
                return (i++) < 2;
            }
        };
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMConfig llmConfig = new LLMConfig();
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        historyStore.store(workflowTask, llmConfig.buildRepositories(workflowTask.getWorkflow()), "UNKNOWN", "OK123", llmConfig.getExpired(), llmConfig.getHistories(), workflowTask.getCreated());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(historyStore);
        iterationService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("OK123"));
        iterationService.setHistoryStore(historyStore);
        iterationService.setMaxTimes(10);
        iterationService.setTimeout(1000);
        iterationService.setMaxSize(10);
        IterationConfig iterationConfig = new IterationConfig();
        iterationConfig.setContainHistories(true);
        iterationConfig.setLlmConfig(new LLMConfig());
        iterationConfig.setRefection("Refection");
        iterationConfig.setProcessor("Processor");
        iterationConfig.setTimes(5);
        Assert.assertEquals("OK123", iterationService.iterate(iterationConfig, workflowTask));
        EasyMock.verify(historyStore);
    }

    @Test(expected = RuntimeException.class)
    public void testIteratorWithConditionAndRetry() throws Exception {
        IterationServiceImpl iterationService = new IterationServiceImpl() {

            private int i = 0;

            @Override
            protected String buildRefection(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer, Integer idx) throws Exception {
                return "OK";
            }

            @Override
            protected Boolean checkCondition(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer, Integer idx) throws Exception {
                return (i++) < 10;
            }
        };
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMConfig llmConfig = new LLMConfig();
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        historyStore.store(workflowTask, llmConfig.buildRepositories(workflowTask.getWorkflow()), "UNKNOWN", "OK123", llmConfig.getExpired(), llmConfig.getHistories(), workflowTask.getCreated());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(historyStore);
        iterationService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("OK123"));
        iterationService.setHistoryStore(historyStore);
        iterationService.setMaxTimes(10);
        iterationService.setTimeout(1000);
        IterationConfig iterationConfig = new IterationConfig();
        iterationConfig.setContainHistories(true);
        iterationConfig.setRefection("Refection");
        iterationConfig.setProcessor("Processor");
        iterationConfig.setTimes(5);
        iterationService.iterate(iterationConfig, workflowTask);
        EasyMock.verify(historyStore);
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        IterationServiceImpl.InitConfig service = new IterationServiceImpl.InitConfig();
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.replay(historyStore);
        service.setNotifierService(notifierManager);
        service.setHistoryStore(historyStore);
        service.setTimeout(1000);
        service.setMaxSize(10);
        service.setMaxTimes(20);
        IterationServiceImpl empty = (IterationServiceImpl) service.iterationService();
        Assert.assertEquals(historyStore, empty.getHistoryStore());
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(Integer.valueOf(1000), empty.getTimeout());
        Assert.assertEquals(Integer.valueOf(20), empty.getMaxTimes());
        Assert.assertEquals(Integer.valueOf(10), empty.getMaxSize());
        EasyMock.verify(historyStore);
    }

    @Test
    public void testStoreFunCalls() throws Exception {
        IterationConfig iterationConfig = new IterationConfig();
        iterationConfig.setFunCallTrack(true);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        Assert.assertNull(workflowTask.getFunCallTrack());
        IterationServiceImpl iterationService = new IterationServiceImpl();
        iterationService.storeFunCalls(iterationConfig, workflowTask);
        Assert.assertNotNull(workflowTask.getFunCallTrack());
    }

    @Test
    public void testRestoreFunCalls() throws Exception {
        IterationConfig iterationConfig = new IterationConfig();
        iterationConfig.setFunCallTrack(true);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        TrackFunCallService trackFunCallService = EasyMock.createMock(TrackFunCallService.class);
        TrackDimension trackDimension = new TrackDimension(workflowTask, workflowTask.getFunCallTrack());
        List<TrackFunCall> trackFunCalls = new ArrayList<>();
        TrackFunCall funCall1 = new TrackFunCall();
        funCall1.setTrackDimension(trackDimension);
        funCall1.setRequest("REQ1");
        funCall1.setResponse("RES1");
        trackFunCalls.add(funCall1);
        TrackFunCall funCall2 = new TrackFunCall();
        funCall2.setTrackDimension(trackDimension);
        funCall2.setRequest("REQ2");
        funCall2.setResponse("RES2");
        trackFunCalls.add(funCall2);
        EasyMock.expect(trackFunCallService.restore(trackDimension)).andReturn(trackFunCalls).anyTimes();
        EasyMock.replay(trackFunCallService);
        IterationServiceImpl iterationService = new IterationServiceImpl() {
            @Override
            protected TrackDimension buildTrackDimension(WorkflowTask workTask) {
                return trackDimension;
            }
        };
        iterationService.setTrackFunCallService(trackFunCallService);
        iterationService.restoreFunCalls(iterationConfig, workflowTask);
        List<TrackFunCall> funCalls = workflowTask.getMetadata(IterationServiceImpl.KEY_TRACK, List.class);
        Assert.assertEquals(Integer.valueOf(2), Integer.valueOf(funCalls.size()));
        Assert.assertEquals("REQ1", funCalls.get(0).getRequest());
        Assert.assertEquals("RES1", funCalls.get(0).getResponse());
        Assert.assertEquals("REQ2", funCalls.get(1).getRequest());
        Assert.assertEquals("RES2", funCalls.get(1).getResponse());
        EasyMock.verify(trackFunCallService);
    }

    @Test
    public void testBuildTrackDimension() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.beginFunCallTrack("ABC");
        IterationServiceImpl iterationService = new IterationServiceImpl();
        TrackDimension trackDimension = iterationService.buildTrackDimension(workflowTask);
        Assert.assertEquals("ABC", trackDimension.getTrack());
        Assert.assertEquals(workflowTask.getDevice(), trackDimension.getDevice());
        Assert.assertEquals(workflowTask.getBiz(), trackDimension.getBiz());
        Assert.assertEquals(workflowTask.getChat(), trackDimension.getChat());
        Assert.assertEquals(workflowTask.getDimension(), trackDimension.getDimension());
    }

    @Test
    public void testSetGet() {
        TrackFunCallService trackFunCallService = EasyMock.createMock(TrackFunCallService.class);
        IterationServiceImpl iterationService = new IterationServiceImpl();
        iterationService.setTrackFunCallService(trackFunCallService);
        iterationService.setMaxTimes(1000);
        Assert.assertEquals(trackFunCallService, iterationService.getTrackFunCallService());
        Assert.assertEquals(Integer.valueOf(1000), iterationService.getMaxTimes());
    }

    @Test
    public void testIterationWithOutRefection() throws Exception {
        IterationConfig iterationConfig = new IterationConfig();
        iterationConfig.setCondition("CONDITION");
        IterationServiceImpl iterationService = new IterationServiceImpl();
        iterationConfig.setProcessor("Processor");
        iterationService.setMaxTimes(10);
        iterationService.setTimeout(1000);
        iterationService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("TRUE"));
        String query = iterationService.buildRefection(iterationConfig, ObjectBuilder.buildWorkflowTask(), new StringBuffer(""), "HELLO WORLD", 0);
        Assert.assertEquals("UNKNOWN", query);
    }

    @Test
    public void testExecOneTimeWithOutCondition() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("Hello World");
        IterationConfig iterationConfig = new IterationConfig();
        iterationConfig.setProcessor("PROCESSOR");
        IterationServiceImpl iterationService = new IterationServiceImpl() {

            private String[] content = new String[]{
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n"
            };

            private int i = 0;

            @Override
            protected String buildProcess(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer, Integer idx) throws Exception {
                String historybody = super.buildProcess(iterationConfig, workTask, history, answer, 0);
                Assert.assertEquals(this.content[this.i++], historybody);
                return historybody;
            }

            @Override
            protected void storeHistories(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer) throws Exception {
                String expect = "The user's original query: Hello World\n" +
                        "##################\n" +
                        "The query round 1: Hello World\n" +
                        "The answer round 1: Return Answer 1\n" +
                        "##################\n";
                Assert.assertEquals(expect, history.toString());
            }
        };
        iterationService.setMaxTimes(10);
        iterationService.setTimeout(100);
        iterationService.setMaxSize(Integer.MAX_VALUE);
        List<Object> content = new ArrayList<Object>();
        content.add("Return Answer 1");
        iterationService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent(content.toArray(new Object[]{})));
        iterationService.iterate(iterationConfig, workflowTask);
    }

    @Test
    public void testExecOneSuccessTwoExceptionWithOutCondition() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("Hello World");
        IterationConfig iterationConfig = new IterationConfig();
        iterationConfig.setProcessor("PROCESSOR");
        IterationServiceImpl iterationService = new IterationServiceImpl() {

            private String[] content = new String[]{
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The error round 1: Return Exception1\n" +
                            "##################\n" +
                            "The query round 2: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The error round 1: Return Exception1\n" +
                            "##################\n" +
                            "The query round 2: Hello World\n" +
                            "The error round 2: Return Exception2\n" +
                            "##################\n" +
                            "The query round 3: Hello World\n"
            };

            private int i = 0;

            @Override
            protected String buildProcess(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer, Integer idx) throws Exception {
                String historyBody = super.buildProcess(iterationConfig, workTask, history, answer, idx);
                Assert.assertEquals(this.content[this.i++], historyBody);
                return historyBody;
            }

            @Override
            protected void storeHistories(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer) throws Exception {
                String expect = "The user's original query: Hello World\n" +
                        "##################\n" +
                        "The query round 1: Hello World\n" +
                        "The error round 1: Return Exception1\n" +
                        "##################\n" +
                        "The query round 2: Hello World\n" +
                        "The error round 2: Return Exception2\n" +
                        "##################\n" +
                        "The query round 3: Hello World\n" +
                        "The answer round 3: Return Answer2\n" +
                        "##################\n";
                Assert.assertEquals(expect, history.toString());
            }
        };
        iterationService.setMaxTimes(10);
        iterationService.setTimeout(100);
        iterationService.setMaxSize(Integer.MAX_VALUE);
        List<Object> content = new ArrayList<Object>();
        content.add(new RuntimeException("Return Exception1"));
        content.add(new RuntimeException("Return Exception2"));
        content.add("Return Answer2");
        iterationService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent(content.toArray(new Object[]{})));
        iterationService.iterate(iterationConfig, workflowTask);
    }

    @Test
    public void testExecOneSuccessOneSuccessConditionOneFailedConditionWithCondition() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("Hello World");
        IterationConfig iterationConfig = new IterationConfig();
        iterationConfig.setCondition("CONDITION");
        iterationConfig.setProcessor("PROCESSOR");
        IterationServiceImpl iterationService = new IterationServiceImpl() {

            private String[] content = new String[]{
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The answer round 1: Return Answer1\n" +
                            "##################\n" +
                            "The query round 2: Hello World\n"
            };

            private int i = 0;

            @Override
            protected String buildProcess(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer, Integer idx) throws Exception {
                String historyBody = super.buildProcess(iterationConfig, workTask, history, answer, idx);
                Assert.assertEquals(this.content[this.i++], historyBody);
                return historyBody;
            }

            @Override
            protected void storeHistories(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer) throws Exception {
                String expect = "The user's original query: Hello World\n" +
                        "##################\n" +
                        "The query round 1: Hello World\n" +
                        "The answer round 1: Return Answer1\n" +
                        "##################\n" +
                        "The query round 2: Hello World\n" +
                        "The answer round 2: Return Answer2\n" +
                        "##################\n";
                Assert.assertEquals(expect, history.toString());
            }
        };
        iterationService.setMaxTimes(10);
        iterationService.setTimeout(100);
        iterationService.setMaxSize(Integer.MAX_VALUE);
        List<Object> content = new ArrayList<Object>();
        content.add("Return Answer1");
        content.add("true");
        content.add("Return Answer2");
        content.add("false");
        iterationService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent(content.toArray(new Object[]{})));
        iterationService.iterate(iterationConfig, workflowTask);
    }

    @Test
    public void testExecOneSuccessOneSuccessConditionOneFailedConditionOneExceptionWithCondition() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("Hello World");
        IterationConfig iterationConfig = new IterationConfig();
        iterationConfig.setCondition("CONDITION");
        iterationConfig.setProcessor("PROCESSOR");
        IterationServiceImpl iterationService = new IterationServiceImpl() {

            private String[] content = new String[]{
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The answer round 1: Return Answer1\n" +
                            "##################\n" +
                            "The query round 2: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The answer round 1: Return Answer1\n" +
                            "##################\n" +
                            "The query round 2: Hello World\n" +
                            "The error round 2: Return Exception2\n" +
                            "##################\n" +
                            "The query round 3: Hello World\n"
            };

            private int i = 0;

            @Override
            protected String buildProcess(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer, Integer idx) throws Exception {
                String historyBody = super.buildProcess(iterationConfig, workTask, history, answer, idx);
                Assert.assertEquals(this.content[this.i++], historyBody);
                return historyBody;
            }

            @Override
            protected void storeHistories(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer) throws Exception {
                String expect = "The user's original query: Hello World\n" +
                        "##################\n" +
                        "The query round 1: Hello World\n" +
                        "The answer round 1: Return Answer1\n" +
                        "##################\n" +
                        "The query round 2: Hello World\n" +
                        "The error round 2: Return Exception2\n" +
                        "##################\n" +
                        "The query round 3: Hello World\n" +
                        "The answer round 3: Return Answer3\n" +
                        "##################\n";
                Assert.assertEquals(expect, history.toString());
            }
        };
        iterationService.setMaxTimes(10);
        iterationService.setTimeout(100);
        iterationService.setMaxSize(Integer.MAX_VALUE);
        List<Object> content = new ArrayList<Object>();
        content.add("Return Answer1");
        content.add("true");
        content.add(new RuntimeException("Return Exception2"));
        content.add("Return Answer3");
        content.add("false");
        iterationService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent(content.toArray(new Object[]{})));
        iterationService.iterate(iterationConfig, workflowTask);
    }

    @Test
    public void testExecOneSuccessFourExceptionWithOutCondition() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("Hello World");
        IterationConfig iterationConfig = new IterationConfig();
        iterationConfig.setProcessor("PROCESSOR");
        iterationConfig.setTimes(6);
        IterationServiceImpl iterationService = new IterationServiceImpl() {

            private String[] content = new String[]{
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The error round 1: Return Exception1\n" +
                            "##################\n" +
                            "The query round 2: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The error round 1: Return Exception1\n" +
                            "##################\n" +
                            "The query round 2: Hello World\n" +
                            "The error round 2: Return Exception2\n" +
                            "##################\n" +
                            "The query round 3: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The error round 1: Return Exception1\n" +
                            "##################\n" +
                            "The query round 2: Hello World\n" +
                            "The error round 2: Return Exception2\n" +
                            "##################\n" +
                            "The query round 3: Hello World\n" +
                            "The error round 3: Return Exception3\n" +
                            "##################\n" +
                            "The query round 4: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The error round 1: Return Exception1\n" +
                            "##################\n" +
                            "The query round 2: Hello World\n" +
                            "The error round 2: Return Exception2\n" +
                            "##################\n" +
                            "The query round 3: Hello World\n" +
                            "The error round 3: Return Exception3\n" +
                            "##################\n" +
                            "The query round 4: Hello World\n" +
                            "The error round 4: Return Exception4\n" +
                            "##################\n" +
                            "The query round 5: Hello World\n"
            };

            private int i = 0;

            @Override
            protected String buildProcess(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer, Integer idx) throws Exception {
                String historyBody = super.buildProcess(iterationConfig, workTask, history, answer, idx);
                Assert.assertEquals(this.content[this.i++], historyBody);
                return historyBody;
            }

            @Override
            protected void storeHistories(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer) throws Exception {
                String expect = "The user's original query: Hello World\n" +
                        "##################\n" +
                        "The query round 1: Hello World\n" +
                        "The error round 1: Return Exception1\n" +
                        "##################\n" +
                        "The query round 2: Hello World\n" +
                        "The error round 2: Return Exception2\n" +
                        "##################\n" +
                        "The query round 3: Hello World\n" +
                        "The error round 3: Return Exception3\n" +
                        "##################\n" +
                        "The query round 4: Hello World\n" +
                        "The error round 4: Return Exception4\n" +
                        "##################\n" +
                        "The query round 5: Hello World\n" +
                        "The answer round 5: Return Answer5\n" +
                        "##################\n";
                Assert.assertEquals(expect, history.toString());
            }
        };
        iterationService.setMaxTimes(10);
        iterationService.setMaxSize(Integer.MAX_VALUE);
        iterationService.setTimeout(100);
        List<Object> content = new ArrayList<Object>();
        content.add(new RuntimeException("Return Exception1"));
        content.add(new RuntimeException("Return Exception2"));
        content.add(new RuntimeException("Return Exception3"));
        content.add(new RuntimeException("Return Exception4"));
        content.add("Return Answer5");
        iterationService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent(content.toArray(new Object[]{})));
        iterationService.iterate(iterationConfig, workflowTask);
    }


    @Test
    public void testExecOneSuccessFourExceptionWithCondition() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("Hello World");
        IterationConfig iterationConfig = new IterationConfig();
        iterationConfig.setProcessor("PROCESSOR");
        iterationConfig.setCondition("CONDITION");
        iterationConfig.setTimes(6);
        IterationServiceImpl iterationService = new IterationServiceImpl() {

            private String[] content = new String[]{
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The error round 1: Return Exception1\n" +
                            "##################\n" +
                            "The query round 2: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The error round 1: Return Exception1\n" +
                            "##################\n" +
                            "The query round 2: Hello World\n" +
                            "The error round 2: Return Exception2\n" +
                            "##################\n" +
                            "The query round 3: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The error round 1: Return Exception1\n" +
                            "##################\n" +
                            "The query round 2: Hello World\n" +
                            "The error round 2: Return Exception2\n" +
                            "##################\n" +
                            "The query round 3: Hello World\n" +
                            "The error round 3: Return Exception3\n" +
                            "##################\n" +
                            "The query round 4: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The error round 1: Return Exception1\n" +
                            "##################\n" +
                            "The query round 2: Hello World\n" +
                            "The error round 2: Return Exception2\n" +
                            "##################\n" +
                            "The query round 3: Hello World\n" +
                            "The error round 3: Return Exception3\n" +
                            "##################\n" +
                            "The query round 4: Hello World\n" +
                            "The error round 4: Return Exception4\n" +
                            "##################\n" +
                            "The query round 5: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The error round 1: Return Exception1\n" +
                            "##################\n" +
                            "The query round 2: Hello World\n" +
                            "The error round 2: Return Exception2\n" +
                            "##################\n" +
                            "The query round 3: Hello World\n" +
                            "The error round 3: Return Exception3\n" +
                            "##################\n" +
                            "The query round 4: Hello World\n" +
                            "The error round 4: Return Exception4\n" +
                            "##################\n" +
                            "The query round 5: Hello World\n" +
                            "The answer round 5: Return Answer5\n" +
                            "##################\n" +
                            "The query round 6: Hello World\n"
            };

            private int i = 0;

            @Override
            protected String buildProcess(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer, Integer idx) throws Exception {
                String historyBody = super.buildProcess(iterationConfig, workTask, history, answer, idx);
                Assert.assertEquals(this.content[this.i++], historyBody);
                return historyBody;
            }

            @Override
            protected void storeHistories(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer) throws Exception {
                String expect = "The user's original query: Hello World\n" +
                        "##################\n" +
                        "The query round 1: Hello World\n" +
                        "The error round 1: Return Exception1\n" +
                        "##################\n" +
                        "The query round 2: Hello World\n" +
                        "The error round 2: Return Exception2\n" +
                        "##################\n" +
                        "The query round 3: Hello World\n" +
                        "The error round 3: Return Exception3\n" +
                        "##################\n" +
                        "The query round 4: Hello World\n" +
                        "The error round 4: Return Exception4\n" +
                        "##################\n" +
                        "The query round 5: Hello World\n" +
                        "The answer round 5: Return Answer5\n" +
                        "##################\n" +
                        "The query round 6: Hello World\n" +
                        "The answer round 6: Return Answer6\n" +
                        "##################\n";
                Assert.assertEquals(expect, history.toString());
            }
        };
        iterationService.setMaxTimes(10);
        iterationService.setTimeout(100);
        iterationService.setMaxSize(Integer.MAX_VALUE);
        List<Object> content = new ArrayList<Object>();
        content.add(new RuntimeException("Return Exception1"));
        content.add(new RuntimeException("Return Exception2"));
        content.add(new RuntimeException("Return Exception3"));
        content.add(new RuntimeException("Return Exception4"));
        content.add("Return Answer5");
        content.add("true");
        content.add("Return Answer6");
        content.add("false");
        iterationService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent(content.toArray(new Object[]{})));
        iterationService.iterate(iterationConfig, workflowTask);
    }

    @Test(expected = RuntimeException.class)
    public void testFailedWithOverTimes() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("Hello World");
        IterationConfig iterationConfig = new IterationConfig();
        iterationConfig.setProcessor("PROCESSOR");
        iterationConfig.setCondition("CONDITION");
        iterationConfig.setTimes(2);
        IterationServiceImpl iterationService = new IterationServiceImpl() {

            private String[] content = new String[]{
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The error round 1: java.lang.RuntimeException: Return Exception1\n" +
                            "##################\n" +
                            "The query round 2: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The error round 1: java.lang.RuntimeException: Return Exception1\n" +
                            "##################\n" +
                            "The query round 2: Hello World\n" +
                            "The error round 2: java.lang.RuntimeException: Return Exception2\n" +
                            "##################\n" +
                            "The query round 3: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The error round 1: java.lang.RuntimeException: Return Exception1\n" +
                            "##################\n" +
                            "The query round 2: Hello World\n" +
                            "The error round 2: java.lang.RuntimeException: Return Exception2\n" +
                            "##################\n" +
                            "The query round 3: Hello World\n" +
                            "The error round 3: java.lang.RuntimeException: Return Exception3\n" +
                            "##################\n" +
                            "The query round 4: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The error round 1: java.lang.RuntimeException: Return Exception1\n" +
                            "##################\n" +
                            "The query round 2: Hello World\n" +
                            "The error round 2: java.lang.RuntimeException: Return Exception2\n" +
                            "##################\n" +
                            "The query round 3: Hello World\n" +
                            "The error round 3: java.lang.RuntimeException: Return Exception3\n" +
                            "##################\n" +
                            "The query round 4: Hello World\n" +
                            "The error round 4: java.lang.RuntimeException: Return Exception4\n" +
                            "##################\n" +
                            "The query round 5: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The error round 1: java.lang.RuntimeException: Return Exception1\n" +
                            "##################\n" +
                            "The query round 2: Hello World\n" +
                            "The error round 2: java.lang.RuntimeException: Return Exception2\n" +
                            "##################\n" +
                            "The query round 3: Hello World\n" +
                            "The error round 3: java.lang.RuntimeException: Return Exception3\n" +
                            "##################\n" +
                            "The query round 4: Hello World\n" +
                            "The error round 4: java.lang.RuntimeException: Return Exception4\n" +
                            "##################\n" +
                            "The query round 5: Hello World\n" +
                            "The answer round 5: Return Answer5\n" +
                            "##################\n" +
                            "The query round 6: Hello World\n"
            };

            private int i = 0;

            @Override
            protected String buildProcess(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer, Integer idx) throws Exception {
                String historyBody = super.buildProcess(iterationConfig, workTask, history, answer, idx);
                Assert.assertEquals(this.content[this.i++], historyBody);
                return historyBody;
            }


            @Override
            protected void storeHistories(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer) throws Exception {
                String expect = "The user's original query: Hello World\n" +
                        "##################\n" +
                        "The query round 1: Hello World\n" +
                        "The error round 1: java.lang.RuntimeException: Return Exception1\n" +
                        "##################\n" +
                        "The query round 2: Hello World\n" +
                        "The error round 2: java.lang.RuntimeException: Return Exception2\n" +
                        "##################\n" +
                        "The query round 3: Hello World\n" +
                        "The error round 3: java.lang.RuntimeException: Return Exception3\n" +
                        "##################\n" +
                        "The query round 4: Hello World\n" +
                        "The error round 4: java.lang.RuntimeException: Return Exception4\n" +
                        "##################\n" +
                        "The query round 5: Hello World\n" +
                        "The answer round 5: Return Answer5\n" +
                        "##################\n" +
                        "The query round 6: Hello World\n" +
                        "The answer round 6: Return Answer6\n" +
                        "##################\n";
                Assert.assertEquals(expect, history.toString());
            }
        };
        iterationService.setMaxTimes(10);
        iterationService.setTimeout(100);
        List<Object> content = new ArrayList<Object>();
        content.add(new RuntimeException("Return Exception1"));
        content.add(new RuntimeException("Return Exception2"));
        content.add(new RuntimeException("Return Exception3"));
        content.add(new RuntimeException("Return Exception4"));
        content.add("Return Answer5");
        content.add("true");
        content.add("Return Answer6");
        content.add("false");
        iterationService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent(content.toArray(new Object[]{})));
        iterationService.iterate(iterationConfig, workflowTask);
    }

    @Test
    public void testRefection() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("Hello World");
        IterationConfig iterationConfig = new IterationConfig();
        iterationConfig.setCondition("CONDITION");
        iterationConfig.setRefection("REFECTION");
        iterationConfig.setProcessor("PROCESSOR");
        IterationServiceImpl iterationService = new IterationServiceImpl() {

            private String[] content = new String[]{
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The answer round 1: Return Answer1\n" +
                            "##################\n" +
                            "The query round 2: Refection Query\n"
            };

            private int i = 0;

            @Override
            protected String buildProcess(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer, Integer idx) throws Exception {
                String historyBody = super.buildProcess(iterationConfig, workTask, history, answer, idx);
                Assert.assertEquals(this.content[this.i++], historyBody);
                return historyBody;
            }

            @Override
            protected void storeHistories(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer) throws Exception {
                String expect = "The user's original query: Hello World\n" +
                        "##################\n" +
                        "The query round 1: Hello World\n" +
                        "The answer round 1: Return Answer1\n" +
                        "##################\n" +
                        "The query round 2: Refection Query\n" +
                        "The answer round 2: Return Answer2\n" +
                        "##################\n";
                Assert.assertEquals(expect, history.toString());
            }
        };
        iterationService.setMaxTimes(10);
        iterationService.setTimeout(100);
        iterationService.setMaxSize(Integer.MAX_VALUE);
        List<Object> content = new ArrayList<Object>();
        content.add("Return Answer1");
        content.add("true");
        content.add("Refection Query");
        content.add("Return Answer2");
        content.add("false");
        iterationService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent(content.toArray(new Object[]{})));
        iterationService.iterate(iterationConfig, workflowTask);
    }

    @Test
    public void testRefectionWithCondition() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("Hello World");
        IterationConfig iterationConfig = new IterationConfig();
        iterationConfig.setCondition("CONDITION");
        iterationConfig.setRefection("REFECTION");
        iterationConfig.setProcessor("PROCESSOR");
        IterationServiceImpl iterationService = new IterationServiceImpl() {

            private String[] content = new String[]{
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n",
                    "The user's original query: Hello World\n" +
                            "##################\n" +
                            "The query round 1: Hello World\n" +
                            "The answer round 1: Return Answer1\n" +
                            "##################\n" +
                            "The check recommendations round 2: hello\n" +
                            "##################\n" +
                            "The query round 2: Refection Query\n"
            };

            private int i = 0;

            @Override
            protected String buildProcess(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer, Integer idx) throws Exception {
                String historyBody = super.buildProcess(iterationConfig, workTask, history, answer, idx);
                Assert.assertEquals(this.content[this.i++], historyBody);
                return historyBody;
            }

            @Override
            protected void storeHistories(IterationConfig iterationConfig, WorkflowTask workTask, StringBuffer history, String answer) throws Exception {
                String expect = "The user's original query: Hello World\n" +
                        "##################\n" +
                        "The query round 1: Hello World\n" +
                        "The answer round 1: Return Answer1\n" +
                        "##################\n" +
                        "The check recommendations round 2: hello\n" +
                        "##################\n" +
                        "The query round 2: Refection Query\n" +
                        "The answer round 2: Return Answer2\n" +
                        "##################\n";
                Assert.assertEquals(expect, history.toString());
            }
        };
        iterationService.setMaxTimes(10);
        iterationService.setTimeout(100);
        iterationService.setMaxSize(Integer.MAX_VALUE);
        List<Object> content = new ArrayList<Object>();
        content.add("Return Answer1");
        content.add("{\"condition\":true,\"content\":\"HELLO\"}");
        content.add("Refection Query");
        content.add("Return Answer2");
        content.add("{\"condition\":false,\"content\":\"HELLO\"}");
        iterationService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent(content.toArray(new Object[]{})));
        iterationService.iterate(iterationConfig, workflowTask);
    }

    @Test
    public void testIteratorWithStore() throws Exception {
        IterationServiceImpl iterationService = new IterationServiceImpl();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMConfig llmConfig = new LLMConfig();
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        historyStore.store(workflowTask, llmConfig.buildRepositories(workflowTask.getWorkflow()), "UNKNOWN", "OK123", llmConfig.getExpired(), llmConfig.getHistories(), workflowTask.getCreated());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(historyStore);
        iterationService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("OK123"));
        iterationService.setHistoryStore(historyStore);
        iterationService.setMaxTimes(10);
        iterationService.setTimeout(1000);
        iterationService.setMaxSize(10);
        IterationConfig iterationConfig = new IterationConfig();
        iterationConfig.setLlmConfig(new LLMConfig());
        iterationConfig.setContainHistories(true);
        iterationConfig.setProcessor("Processor");
        Assert.assertEquals("OK123", iterationService.iterate(iterationConfig, workflowTask));
        EasyMock.verify(historyStore);
    }
    @Test(expected = IllegalArgumentException.class)
    public void testIterateMaxTimesExceeded() throws Exception {
        IterationServiceImpl iterationService = new IterationServiceImpl();
        iterationService.setMaxTimes(5);
        IterationConfig config = new IterationConfig();
        config.setTimes(10);
        iterationService.iterate(config, ObjectBuilder.buildWorkflowTask());
    }

    @Test(expected = IllegalArgumentException.class)
    public void testIterateNoProcessor() throws Exception {
        IterationServiceImpl iterationService = new IterationServiceImpl();
        iterationService.setMaxTimes(10);
        IterationConfig config = new IterationConfig();
        config.setTimes(5);
        iterationService.iterate(config, ObjectBuilder.buildWorkflowTask());
    }

    @Test(expected = IllegalArgumentException.class)
    public void testIterateFunMergeNotEmpty() throws Exception {
        IterationServiceImpl iterationService = new IterationServiceImpl();
        iterationService.setMaxTimes(10);
        IterationConfig config = new IterationConfig();
        config.setTimes(5);
        config.setProcessor("P");
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        task.putMetadata(ProviderRequestService.KEY_FUN_MERGE, "DATA");
        iterationService.iterate(config, task);
    }

    @Test(expected = RuntimeException.class)
    public void testIterateLastRoundException() throws Exception {
        IterationServiceImpl iterationService = new IterationServiceImpl();
        iterationService.setMaxTimes(10);
        iterationService.setTimeout(100);
        IterationConfig config = new IterationConfig();
        config.setTimes(2);
        config.setProcessor("P");
        
        NotifierServiceImpl notifierService = EasyMock.createMock(NotifierServiceImpl.class);
        // First round succeeds, second round (index 1, which is times-1) fails
        notifierService.notify(EasyMock.anyObject(ai.open.right.workflow.flow.llm.Segment.class), EasyMock.anyObject(ai.open.right.context.RedirectContext.class), EasyMock.anyObject(ai.open.right.workflow.notify.NotifierWriteBack.class)); EasyMock.expectLastCall().times(1);
        notifierService.notify(EasyMock.anyObject(ai.open.right.workflow.flow.llm.Segment.class), EasyMock.anyObject(ai.open.right.context.RedirectContext.class), EasyMock.anyObject(ai.open.right.workflow.notify.NotifierWriteBack.class)); EasyMock.expectLastCall().andThrow(new RuntimeException("LAST ROUND ERROR")).times(1);
        
        iterationService.setNotifierService(notifierService);
        EasyMock.replay(notifierService);
        iterationService.iterate(config, ObjectBuilder.buildWorkflowTask());
    }

    @Test
    public void testBuildProcessMaxSize() throws Exception {
        IterationServiceImpl iterationService = new IterationServiceImpl();
        iterationService.setMaxSize(5);
        StringBuffer history = new StringBuffer("1234567890");
        String result = iterationService.buildProcess(null, null, history, null, 0);
        Assert.assertEquals("67890", result);
    }
}
