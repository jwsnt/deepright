package ai.open.right.workflow.flow.impl;

import java.lang.reflect.Field;

import org.junit.Assert;
import org.junit.Test;

public class WorkflowQueueImplInitConfigTest {

    @Test
    public void shouldCreateWorkflowQueueBeanAndCopyProperties() throws Exception {
        WorkflowQueueImpl.InitConfig init = new WorkflowQueueImpl.InitConfig();
        setPrivate(init, "timeout", 20);
        setPrivate(init, "queue", 100);

        WorkflowQueueImpl bean = (WorkflowQueueImpl) init.workflowQueue();

        Assert.assertEquals(Integer.valueOf(20), readPrivate(bean, "timeout", Integer.class));
        Assert.assertEquals(Integer.valueOf(100), readPrivate(bean, "queue", Integer.class));
    }

    private static <T> T readPrivate(Object target, String fieldName, Class<T> type) {
        try {
            Field f = target.getClass().getDeclaredField(fieldName);
            f.setAccessible(true);
            Object value = f.get(target);
            return type.cast(value);
        } catch (Exception e) {
            throw new AssertionError("Failed to read field '" + fieldName + "'", e);
        }
    }

    private static void setPrivate(Object target, String fieldName, Object value) throws Exception {
        Field f = target.getClass().getDeclaredField(fieldName);
        f.setAccessible(true);
        f.set(target, value);
    }
}
