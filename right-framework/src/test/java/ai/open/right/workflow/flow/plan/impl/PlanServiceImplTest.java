package ai.open.right.workflow.flow.plan.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.iteration.IterationConfig;
import ai.open.right.workflow.flow.iteration.IterationService;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.plan.PlanConfig;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class PlanServiceImplTest {

    @Test
    public void testPlan() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        PlanServiceImpl planService = new PlanServiceImpl();
        planService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("Plan1\nPlan2"));
        PlanConfig config = new PlanConfig();
        config.setPlan("Plan");
        String plan = planService.buildPlan(config, workflowTask);
        Assert.assertEquals("Plan1\nPlan2", plan);
    }

    @Test
    public void testExecute() throws Exception {
        IterationConfig iterationConfig = new IterationConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        IterationService iterationService = EasyMock.createMock(IterationService.class);
        EasyMock.expect(iterationService.iterate(iterationConfig, workflowTask)).andReturn("HELLO WORLD").anyTimes();
        EasyMock.replay(iterationService);
        PlanServiceImpl planService = new PlanServiceImpl();
        planService.setIterationService(iterationService);
        planService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("Plan1\nPlan2"));
        PlanConfig config = new PlanConfig();
        config.setIterationConfig(iterationConfig);
        config.setPlan("Plan");
        Assert.assertEquals("HELLO WORLD", planService.plan(config, workflowTask));
        EasyMock.verify(iterationService);
    }

    @Test
    public void testExecuteSummary() throws Exception {
        IterationConfig iterationConfig = new IterationConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        IterationService iterationService = EasyMock.createMock(IterationService.class);
        EasyMock.expect(iterationService.iterate(iterationConfig, workflowTask)).andReturn("HELLO WORLD").anyTimes();
        EasyMock.replay(iterationService);
        PlanServiceImpl planService = new PlanServiceImpl();
        planService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("SUCCESS"));
        planService.setIterationService(iterationService);
        PlanConfig config = new PlanConfig();
        config.setIterationConfig(iterationConfig);
        config.setSummary("Summary");
        config.setPlan("Plan");
        Assert.assertEquals("SUCCESS", planService.plan(config, workflowTask));
        EasyMock.verify(iterationService);
    }

    @Test
    public void testExecuteSummaryWithStore() throws Exception {
        IterationConfig iterationConfig = new IterationConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        IterationService iterationService = EasyMock.createMock(IterationService.class);
        EasyMock.expect(iterationService.iterate(iterationConfig, workflowTask)).andReturn("HELLO WORLD").anyTimes();
        EasyMock.replay(iterationService);
        LLMConfig llmConfig = new LLMConfig();
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        historyStore.store(workflowTask, llmConfig.buildRepositories(workflowTask.getWorkflow()), "QUERY", "SUCCESS", llmConfig.getExpired(), llmConfig.getHistories(), workflowTask.getCreated());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(historyStore);
        PlanServiceImpl planService = new PlanServiceImpl();
        planService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("QUERY", "SUCCESS"));
        planService.setIterationService(iterationService);
        planService.setHistoryStore(historyStore);
        PlanConfig config = new PlanConfig();
        config.init(llmConfig);
        config.setIterationConfig(iterationConfig);
        config.setContainHistories(true);
        config.setSummary("Summary");
        config.setPlan("Plan");
        Assert.assertEquals("SUCCESS", planService.plan(config, workflowTask));
        EasyMock.verify(iterationService, historyStore);
    }

    @Test(expected = RuntimeException.class)
    public void testExecuteException() throws Exception {
        IterationConfig iterationConfig = new IterationConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        IterationService iterationService = EasyMock.createMock(IterationService.class);
        EasyMock.expect(iterationService.iterate(iterationConfig, workflowTask)).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(iterationService);
        PlanServiceImpl planService = new PlanServiceImpl();
        planService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("Plan1\nPlan2"));
        planService.setIterationService(iterationService);
        PlanConfig config = new PlanConfig();
        config.setIterationConfig(iterationConfig);
        config.setPlan("Plan");
        planService.plan(config, workflowTask);
        EasyMock.verify(iterationService);
    }

    @Test
    public void testExecuteExceptionSummary() throws Exception {
        IterationConfig iterationConfig = new IterationConfig();
        LLMConfig llmConfig = new LLMConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        IterationService iterationService = EasyMock.createMock(IterationService.class);
        EasyMock.expect(iterationService.iterate(iterationConfig, workflowTask)).andThrow(new RuntimeException("NOT HAPPY")).anyTimes();
        EasyMock.replay(iterationService);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        historyStore.store(workflowTask, llmConfig.buildRepositories(workflowTask.getWorkflow()), "EXCEPTION", "FAILED", llmConfig.getExpired(), llmConfig.getHistories(), workflowTask.getCreated());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(historyStore);
        PlanServiceImpl planService = new PlanServiceImpl();
        planService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("EXCEPTION", "FAILED"));
        planService.setIterationService(iterationService);
        planService.setHistoryStore(historyStore);
        PlanConfig config = new PlanConfig();
        config.setLlmConfig(llmConfig);
        config.setContainHistories(true);
        config.setIterationConfig(iterationConfig);
        config.setException("Exception");
        config.setPlan("Plan");
        Assert.assertEquals("FAILED", planService.plan(config, workflowTask));
        EasyMock.verify(iterationService);
    }

    @Test
    public void testExecuteExceptionSummaryWithStore() throws Exception {
        IterationConfig iterationConfig = new IterationConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        IterationService iterationService = EasyMock.createMock(IterationService.class);
        EasyMock.expect(iterationService.iterate(iterationConfig, workflowTask)).andThrow(new RuntimeException("NOT HAPPY")).anyTimes();
        EasyMock.replay(iterationService);
        PlanServiceImpl planService = new PlanServiceImpl();
        planService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("FAILED"));
        planService.setIterationService(iterationService);
        PlanConfig config = new PlanConfig();
        config.setIterationConfig(iterationConfig);
        config.setException("Exception");
        config.setPlan("Plan");
        Assert.assertEquals("FAILED", planService.plan(config, workflowTask));
        EasyMock.verify(iterationService);
    }

    @Test
    public void testInit() throws Exception {
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.replay(historyStore);
        IterationService iterationService = EasyMock.createMock(IterationService.class);
        TrackFunCallService trackFunCallService = EasyMock.createMock(TrackFunCallService.class);
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        EasyMock.replay(trackFunCallService, iterationService);
        PlanServiceImpl.InitConfig service = new PlanServiceImpl.InitConfig();
        service.setIterationService(iterationService);
        service.setNotifierService(notifierManager);
        service.setHistoryStore(historyStore);
        service.setTimeout4Llm(1000);
        PlanServiceImpl empty = (PlanServiceImpl) service.planService();
        Assert.assertEquals(iterationService, empty.getIterationService());
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(Integer.valueOf(1000), empty.getTimeout4Llm());
        Assert.assertEquals(historyStore, empty.getHistoryStore());
        EasyMock.verify(trackFunCallService, historyStore);
    }

    @Test
    public void testExecuteIterationWithStore() throws Exception {
        IterationConfig iterationConfig = new IterationConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        IterationService iterationService = EasyMock.createMock(IterationService.class);
        EasyMock.expect(iterationService.iterate(iterationConfig, workflowTask)).andReturn("HELLO WORLD").anyTimes();
        EasyMock.replay(iterationService);
        LLMConfig llmConfig = new LLMConfig();
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        historyStore.store(workflowTask, llmConfig.buildRepositories(workflowTask.getWorkflow()), "QUERY", "HELLO WORLD", llmConfig.getExpired(), llmConfig.getHistories(), workflowTask.getCreated());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(historyStore);
        PlanServiceImpl planService = new PlanServiceImpl();
        planService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("QUERY"));
        planService.setIterationService(iterationService);
        planService.setHistoryStore(historyStore);
        PlanConfig config = new PlanConfig();
        config.init(llmConfig);
        config.setIterationConfig(iterationConfig);
        config.setContainHistories(true);
        config.setPlan("Plan");
        Assert.assertEquals("HELLO WORLD", planService.plan(config, workflowTask));
        EasyMock.verify(iterationService, historyStore);
    }
    @Test
    public void testStoreHistoriesDisabled() throws Exception {
        PlanServiceImpl service = new PlanServiceImpl();
        PlanConfig config = new PlanConfig();
        config.setContainHistories(false);
        service.storeHistories(config, ObjectBuilder.buildWorkflowTask(), "A");
    }
}
