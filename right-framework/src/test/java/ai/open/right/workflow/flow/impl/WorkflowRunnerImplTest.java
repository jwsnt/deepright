package ai.open.right.workflow.flow.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.Workflow;
import ai.open.right.workflow.flow.WorkflowQueue;
import ai.open.right.workflow.flow.WorkflowTask;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

public class WorkflowRunnerImplTest {

    @Test
    public void test() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(10);
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQuery();
        WorkflowQueue workflowQueue = EasyMock.createMock(WorkflowQueue.class);
        EasyMock.expect(workflowQueue.get()).andReturn(workflowTask).anyTimes();
        Workflow workflow = EasyMock.createMock(Workflow.class);
        workflow.async(workflowTask);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(workflowQueue, workflow);
        WorkflowRunnerImpl worker = new WorkflowRunnerImpl();
        worker.setExecutorService(executorService);
        worker.setWorkflowQueue(workflowQueue);
        worker.setWorkflow(workflow);
        new Thread(new Runnable() {
            @Override
            public void run() {
                try {
                    Thread.sleep(2000);
                } catch (InterruptedException e) {
                    throw new RuntimeException(e);
                }
                worker.destroy();
            }
        }).start();
        worker.run();
        EasyMock.verify(workflowQueue, workflow);
        executorService.shutdown();
    }

    @Test
    public void testWithException() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(10);
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQuery();
        WorkflowQueue workflowQueue = EasyMock.createMock(WorkflowQueue.class);
        EasyMock.expect(workflowQueue.get()).andThrow(new RuntimeException()).anyTimes();
        Workflow workflow = EasyMock.createMock(Workflow.class);
        workflow.async(workflowTask);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(workflowQueue, workflow);
        WorkflowRunnerImpl worker = new WorkflowRunnerImpl();
        worker.setExecutorService(executorService);
        worker.setWorkflowQueue(workflowQueue);
        worker.setWorkflow(workflow);
        new Thread(new Runnable() {
            @Override
            public void run() {
                try {
                    Thread.sleep(2000);
                } catch (InterruptedException e) {
                    throw new RuntimeException(e);
                }
                worker.destroy();
            }
        }).start();
        worker.run();
        EasyMock.verify(workflowQueue, workflow);
        executorService.shutdown();
    }

    @Test
    public void testInit() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(10);
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQuery();
        WorkflowQueue workflowQueue = EasyMock.createMock(WorkflowQueue.class);
        EasyMock.expect(workflowQueue.get()).andReturn(workflowTask).anyTimes();
        Workflow workflow = EasyMock.createMock(Workflow.class);
        workflow.async(workflowTask);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(workflowQueue, workflow);
        WorkflowRunnerImpl worker = new WorkflowRunnerImpl();
        worker.setExecutorService(executorService);
        worker.setWorkflowQueue(workflowQueue);
        worker.setWorkflow(workflow);
        worker.setThreads(10);
        worker.init();
        Thread.sleep(1000);
        worker.destroy();
        EasyMock.verify(workflowQueue, workflow);
        executorService.shutdown();
    }

    @Test
    public void testBuild() throws Exception {
        ExecutorService executorService = EasyMock.createMock(ExecutorService.class);
        WorkflowQueue workflowQueue = EasyMock.createMock(WorkflowQueue.class);
        Workflow workflow = EasyMock.createMock(Workflow.class);
        EasyMock.replay(executorService, workflow, workflowQueue);
        WorkflowRunnerImpl.InitConfig workflowRunner = new WorkflowRunnerImpl.InitConfig();
        workflowRunner.setExecutorService(executorService);
        workflowRunner.setWorkflow(workflow);
        workflowRunner.setWorkflowQueue(workflowQueue);
        workflowRunner.setThreads(100);
        WorkflowRunnerImpl empty = (WorkflowRunnerImpl) workflowRunner.workflowRunner();
        Assert.assertEquals(workflow, empty.getWorkflow());
        Assert.assertEquals(executorService, empty.getExecutorService());
        Assert.assertEquals(workflowQueue, empty.getWorkflowQueue());
        Assert.assertEquals(Integer.valueOf(100), empty.getThreads());
    }

    @org.junit.jupiter.api.Test
    public void testRunnerAdditional() {
        WorkflowRunnerImpl runner = new WorkflowRunnerImpl();
        org.junit.jupiter.api.Assertions.assertNotNull(runner);
    }

    @org.junit.jupiter.api.Test
    public void testRunShutdown() throws Exception {
        WorkflowRunnerImpl runner = new WorkflowRunnerImpl();
        runner.setShutdown(true);
        // 应该立即退出循环
        runner.run();
    }

}
