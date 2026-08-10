package ai.open.right.workflow.flow.fork.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.fork.ForkConfig;
import ai.open.right.workflow.flow.fork.ForkTarget;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.concurrent.atomic.AtomicInteger;

public class ForkServiceImplTest {

    @Test
    public void testTarget() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        ForkServiceImpl forkService = new ForkServiceImpl();
        forkService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        forkService.setTimeout(10000);
        forkService.fork2Target(workflowTask, "Target");
    }

    @Test
    public void testCheck2Fork() throws Exception {
        WorkflowConfig workflowConfig = new WorkflowConfig();
        ForkConfig forkConfig = new ForkConfig();
        workflowConfig.setForkConfig(forkConfig);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        List<ForkServiceImpl.ForkFuture> forkFutures = new ArrayList<>();
        SyncWorkflowTask syncWorkflowTask1 = EasyMock.createMock(SyncWorkflowTask.class);
        EasyMock.expect(syncWorkflowTask1.get()).andReturn("TRUE");
        ForkTarget forkTarget1 = new ForkTarget();
        forkTarget1.setDynamic("DYNAMIC1");
        forkFutures.add(ForkServiceImpl.ForkFuture.builder().conditionTask(syncWorkflowTask1).target(forkTarget1).build());
        SyncWorkflowTask syncWorkflowTask2 = EasyMock.createMock(SyncWorkflowTask.class);
        EasyMock.expect(syncWorkflowTask2.get()).andReturn("FALSE");
        ForkTarget forkTarget2 = new ForkTarget();
        forkTarget2.setDynamic("DYNAMIC2");
        forkFutures.add(ForkServiceImpl.ForkFuture.builder().conditionTask(syncWorkflowTask2).target(forkTarget2).build());
        EasyMock.replay(syncWorkflowTask1, syncWorkflowTask2);
        ForkServiceImpl forkService = new ForkServiceImpl() {
            protected void fork2Target(WorkflowTask workTask, String workflow) throws Exception {

            }
        };
        forkService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        forkService.setTimeout(10000);
        Assert.assertEquals(Integer.valueOf(1), forkService.checkAndFork(workflowConfig, workflowTask, forkFutures));
        EasyMock.verify(syncWorkflowTask1, syncWorkflowTask2);
    }


    @Test
    public void testCheck2ForkAndException() throws Exception {
        WorkflowConfig workflowConfig = new WorkflowConfig();
        ForkConfig forkConfig = new ForkConfig();
        workflowConfig.setForkConfig(forkConfig);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        List<ForkServiceImpl.ForkFuture> forkFutures = new ArrayList<>();
        SyncWorkflowTask syncWorkflowTask1 = EasyMock.createMock(SyncWorkflowTask.class);
        EasyMock.expect(syncWorkflowTask1.get()).andReturn("TRUE");
        ForkTarget forkTarget1 = new ForkTarget();
        forkTarget1.setDynamic("DYNAMIC1");
        forkFutures.add(ForkServiceImpl.ForkFuture.builder().conditionTask(syncWorkflowTask1).target(forkTarget1).build());
        SyncWorkflowTask syncWorkflowTask2 = EasyMock.createMock(SyncWorkflowTask.class);
        EasyMock.expect(syncWorkflowTask2.get()).andThrow(new RuntimeException("ERROR"));
        ForkTarget forkTarget2 = new ForkTarget();
        forkTarget2.setDynamic("DYNAMIC2");
        forkFutures.add(ForkServiceImpl.ForkFuture.builder().conditionTask(syncWorkflowTask2).target(forkTarget2).build());
        EasyMock.replay(syncWorkflowTask1, syncWorkflowTask2);
        ForkServiceImpl forkService = new ForkServiceImpl() {
            protected void fork2Target(WorkflowTask workTask, String workflow) throws Exception {

            }
        };
        forkService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        forkService.setTimeout(10000);
        Assert.assertEquals(Integer.valueOf(1), forkService.checkAndFork(workflowConfig, workflowTask, forkFutures));
        EasyMock.verify(syncWorkflowTask1, syncWorkflowTask2);
    }

    @Test(expected = RuntimeException.class)
    public void testCheck2ForkAndExceptionStop() throws Exception {
        WorkflowConfig workflowConfig = new WorkflowConfig();
        ForkConfig forkConfig = new ForkConfig();
        forkConfig.setStopOnFailed(true);
        workflowConfig.setForkConfig(forkConfig);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        List<ForkServiceImpl.ForkFuture> forkFutures = new ArrayList<>();
        SyncWorkflowTask syncWorkflowTask1 = EasyMock.createMock(SyncWorkflowTask.class);
        EasyMock.expect(syncWorkflowTask1.get()).andReturn("TRUE");
        ForkTarget forkTarget1 = new ForkTarget();
        forkTarget1.setDynamic("DYNAMIC1");
        forkFutures.add(ForkServiceImpl.ForkFuture.builder().conditionTask(syncWorkflowTask1).target(forkTarget1).build());
        SyncWorkflowTask syncWorkflowTask2 = EasyMock.createMock(SyncWorkflowTask.class);
        EasyMock.expect(syncWorkflowTask2.get()).andThrow(new RuntimeException("ERROR"));
        ForkTarget forkTarget2 = new ForkTarget();
        forkTarget2.setDynamic("DYNAMIC2");
        forkFutures.add(ForkServiceImpl.ForkFuture.builder().conditionTask(syncWorkflowTask2).target(forkTarget2).build());
        EasyMock.replay(syncWorkflowTask1, syncWorkflowTask2);
        ForkServiceImpl forkService = new ForkServiceImpl() {
            protected void fork2Target(WorkflowTask workTask, String workflow) throws Exception {

            }
        };
        forkService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        forkService.setTimeout(10000);
        try {
            forkService.checkAndFork(workflowConfig, workflowTask, forkFutures);
        } finally {
            EasyMock.verify(syncWorkflowTask1, syncWorkflowTask2);
        }
    }

    @Test
    public void testFork() throws Exception {
        WorkflowConfig workflowConfig = new WorkflowConfig();
        ForkConfig forkConfig = new ForkConfig();
        forkConfig.setStopOnFailed(true);
        ForkTarget t1 = new ForkTarget();
        t1.setDynamic("D1");
        t1.setCondition("C1");
        ForkTarget t2 = new ForkTarget();
        t2.setDynamic("D1");
        forkConfig.setTarget(Arrays.asList(t1, t2));
        workflowConfig.setForkConfig(forkConfig);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        ForkServiceImpl forkService = new ForkServiceImpl() {
            protected Integer checkAndFork(WorkflowConfig workflowConfig, WorkflowTask workTask, List<ForkFuture> forkFutures) throws Exception {
                return 100;
            }

            protected void fork2Target(WorkflowTask workTask, String workflow) throws Exception {

            }
        };
        forkService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        forkService.setTimeout(1000);
        forkService.fork(workflowConfig, workflowTask);
    }

    @Test
    public void testDefault() throws Exception {
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setChain("CHAIN");
        ForkConfig forkConfig = new ForkConfig();
        forkConfig.setStopOnFailed(true);
        ForkTarget t1 = new ForkTarget();
        t1.setDynamic("D1");
        t1.setCondition("C1");
        ForkTarget t2 = new ForkTarget();
        t2.setCondition("C1");
        t2.setDynamic("D1");
        forkConfig.setTarget(Arrays.asList(t1, t2));
        workflowConfig.setForkConfig(forkConfig);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        AtomicInteger count = new AtomicInteger(0);
        ForkServiceImpl forkService = new ForkServiceImpl() {
            protected Integer checkAndFork(WorkflowConfig workflowConfig, WorkflowTask workTask, List<ForkFuture> forkFutures) throws Exception {
                return 0;
            }

            protected void fork2Target(WorkflowTask workTask, String workflow) throws Exception {
                if (workflow.equalsIgnoreCase("CHAIN")) {
                    count.incrementAndGet();
                }
            }
        };
        forkService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        forkService.setTimeout(1000);
        forkService.fork(workflowConfig, workflowTask);
        Assert.assertTrue(count.get() > 0);
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        ForkServiceImpl.InitConfig service = new ForkServiceImpl.InitConfig();
        service.setNotifierService(notifierManager);
        service.setTimeout(1000);
        ForkServiceImpl empty = (ForkServiceImpl) service.forkService();
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(Integer.valueOf(1000), empty.getTimeout());
    }
    @Test
    public void testForkEmptyTarget() throws Exception {
        ForkServiceImpl service = new ForkServiceImpl();
        WorkflowConfig config = new WorkflowConfig();
        config.setForkConfig(new ForkConfig());
        service.fork(config, ObjectBuilder.buildWorkflowTask());
    }

    @Test
    public void testCheckAndForkNull() throws Exception {
        ForkServiceImpl service = new ForkServiceImpl();
        Assert.assertEquals(Integer.valueOf(0), service.checkAndFork(null, null, null));
    }
}
