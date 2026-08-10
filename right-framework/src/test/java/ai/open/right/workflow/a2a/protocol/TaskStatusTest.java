package ai.open.right.workflow.a2a.protocol;

import org.junit.jupiter.api.Test;

import static org.junit.Assert.assertEquals;

public class TaskStatusTest {

    @Test
    void testGetSet() {
        TaskStatus status = TaskStatus.builder().build();
        status.setState(TaskStatus.STATUS_INPUT_REQUIRED);
        assertEquals(TaskStatus.STATUS_INPUT_REQUIRED, status.getState());
        status.setState(TaskStatus.STATUS_AUTH_REQUIRED);
        assertEquals(TaskStatus.STATUS_AUTH_REQUIRED, status.getState());
        status.setState(TaskStatus.STATUS_SUBMITTED);
        assertEquals(TaskStatus.STATUS_SUBMITTED, status.getState());
        status.setState(TaskStatus.STATUS_COMPLETED);
        assertEquals(TaskStatus.STATUS_COMPLETED, status.getState());
        status.setState(TaskStatus.STATUS_REJECTED);
        assertEquals(TaskStatus.STATUS_REJECTED, status.getState());
        status.setState(TaskStatus.STATUS_CANCELED);
        assertEquals(TaskStatus.STATUS_CANCELED, status.getState());
        status.setState(TaskStatus.STATUS_WORKING);
        assertEquals(TaskStatus.STATUS_WORKING, status.getState());
        status.setState(TaskStatus.STATUS_UNKNOWN);
        assertEquals(TaskStatus.STATUS_UNKNOWN, status.getState());
        status.setState(TaskStatus.STATUS_FAILED);
        assertEquals(TaskStatus.STATUS_FAILED, status.getState());
    }

    @Test
    void testBuilder() {
        TaskStatus status = TaskStatus.builder().state(TaskStatus.STATUS_INPUT_REQUIRED).build();
        assertEquals(TaskStatus.STATUS_INPUT_REQUIRED, status.getState());
        
        TaskStatus defaultStatus = TaskStatus.builder().build();
        assertEquals(TaskStatus.STATUS_COMPLETED, defaultStatus.getState());
    }
}