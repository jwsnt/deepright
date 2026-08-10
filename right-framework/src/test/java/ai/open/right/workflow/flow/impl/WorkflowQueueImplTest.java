package ai.open.right.workflow.flow.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.WorkflowTask;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.junit.jupiter.api.Assertions;

import java.util.concurrent.BlockingQueue;

public class WorkflowQueueImplTest {

    @Test
    public void testInit() throws Exception {
        WorkflowQueueImpl queue = new WorkflowQueueImpl();
        queue.setQueue(1024);
        queue.init();
        Assert.assertNotNull(queue.getWorkflowTasks());
    }

    @Test
    public void testPutGet() throws Exception {
        WorkflowQueueImpl queue = new WorkflowQueueImpl();
        queue.setTimeout(10);
        queue.setQueue(1024);
        queue.init();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        queue.put(workflowTask);
        Assert.assertEquals(queue.get(), workflowTask);
    }

    @Test
    public void testGetNull() throws Exception {
        WorkflowQueueImpl queue = new WorkflowQueueImpl();
        queue.setTimeout(2);
        queue.setQueue(1024);
        queue.init();
        Assert.assertNull(queue.get());
    }

    @Test
    public void testMonitor() throws Exception {
        WorkflowQueueImpl queue = new WorkflowQueueImpl();
        queue.setTimeout(10);
        queue.setQueue(1024);
        queue.init();
        queue.put(ObjectBuilder.buildWorkflowTask());
        queue.monitor();
    }

    @Test
    public void testGetWithException() throws Exception {
        BlockingQueue<?> blockingQueue = EasyMock.createMock(BlockingQueue.class);
        EasyMock.expect(blockingQueue.poll(EasyMock.anyLong(), EasyMock.anyObject())).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(blockingQueue);
        WorkflowQueueImpl queue = new WorkflowQueueImpl();
        queue.setWorkflowTasks((BlockingQueue<WorkflowTask>) blockingQueue);
        try {
            queue.get();
        } finally {
            EasyMock.verify(blockingQueue);
        }
    }

    @Test(expected = RuntimeException.class)
    public void testOfferWithException() throws Exception {
        BlockingQueue<?> blockingQueue = EasyMock.createMock(BlockingQueue.class);
        EasyMock.expect(blockingQueue.offer(EasyMock.anyObject())).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(blockingQueue);
        WorkflowQueueImpl queue = new WorkflowQueueImpl();
        queue.setWorkflowTasks((BlockingQueue<WorkflowTask>) blockingQueue);
        try {
            queue.put(null);
        } finally {
            EasyMock.verify(blockingQueue);
        }
    }

    @Test(expected = WorkflowException.class)
    public void testPutFalse() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        BlockingQueue<WorkflowTask> blockingQueue = EasyMock.createMock(BlockingQueue.class);
        EasyMock.expect(blockingQueue.offer(workflowTask)).andReturn(false).anyTimes();
        EasyMock.replay(blockingQueue);
        WorkflowQueueImpl queue = new WorkflowQueueImpl();
        queue.setWorkflowTasks((BlockingQueue<WorkflowTask>) blockingQueue);
        queue.put(workflowTask);
    }

    @Test
    public void testBuild() throws Exception {
        WorkflowQueueImpl.InitConfig workflowQueue = new WorkflowQueueImpl.InitConfig();
        workflowQueue.setTimeout(100);
        workflowQueue.setQueue(200);
        WorkflowQueueImpl empty = (WorkflowQueueImpl) workflowQueue.workflowQueue();
        empty.init();
        Assert.assertNotNull(empty.getWorkflowTasks());
        Assert.assertEquals(Integer.valueOf(100), empty.getTimeout());
        Assert.assertEquals(Integer.valueOf(200), empty.getQueue());
    }

    @Test
    public void testMonitorDetailed() {
        // 1. 初始化队列
        WorkflowQueueImpl queue = new WorkflowQueueImpl();
        queue.setTimeout(10);
        queue.setQueue(1024);
        queue.init();
        // 2. 调用 monitor()
        String monitorInfo = queue.monitor();
        // 3. 验证返回的字符串是否包含 "WorkflowQueue size="
        Assert.assertTrue(monitorInfo.contains("WorkflowQueue size="));
    }

    /**
     * 新增 JUnit 5 测试方法，验证 monitor 详细信息
     */
    @org.junit.jupiter.api.Test
    public void testMonitorDetailedJUnit5() throws Exception {
        // 1. 初始化队列
        WorkflowQueueImpl queue = new WorkflowQueueImpl();
        queue.setTimeout(10);
        queue.setQueue(1024);
        queue.init();
        // 2. 放入一个任务
        queue.put(ObjectBuilder.buildWorkflowTask());
        // 3. 调用 monitor()
        String monitorInfo = queue.monitor();
        // 4. 验证返回的字符串包含 "WorkflowQueue size=1"
        Assertions.assertTrue(monitorInfo.contains("WorkflowQueue size=1"));
    }
}

