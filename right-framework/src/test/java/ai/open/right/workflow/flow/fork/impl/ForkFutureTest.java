package ai.open.right.workflow.flow.fork.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.fork.ForkTarget;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

public class ForkFutureTest {

    @Test
    void testForkFutureBuilder() {
        SyncWorkflowTask mockConditionTask = new SyncWorkflowTask(ObjectBuilder.buildWorkflowTask(), null, 100);
        ForkTarget mockTarget = new ForkTarget();
        ForkServiceImpl.ForkFuture forkFuture = ForkServiceImpl.ForkFuture.builder()
                .conditionTask(mockConditionTask)
                .target(mockTarget)
                .build();
        assertNotNull(forkFuture, "ForkFuture对象不应为null");
        assertSame(mockConditionTask, forkFuture.getConditionTask(), "conditionTask属性不匹配");
        assertSame(mockTarget, forkFuture.getTarget(), "target属性不匹配");
    }

    @Test
    void testForkFutureSettersAndGetters() {
        ForkServiceImpl.ForkFuture forkFuture = ForkServiceImpl.ForkFuture.builder().build();
        SyncWorkflowTask newConditionTask = new SyncWorkflowTask(ObjectBuilder.buildWorkflowTask(), null, 1000);
        ForkTarget newTarget = new ForkTarget();
        forkFuture.setConditionTask(newConditionTask);
        forkFuture.setTarget(newTarget);
        assertSame(newConditionTask, forkFuture.getConditionTask(), "设置conditionTask后获取的值不匹配");
        assertSame(newTarget, forkFuture.getTarget(), "设置target后获取的值不匹配");
    }

    @Test
    void testForkFutureWithNullValues() {
        ForkServiceImpl.ForkFuture forkFuture = ForkServiceImpl.ForkFuture.builder()
                .conditionTask(null)
                .target(null)
                .build();
        assertNull(forkFuture.getConditionTask(), "conditionTask应为null");
        assertNull(forkFuture.getTarget(), "target应为null");
    }
}