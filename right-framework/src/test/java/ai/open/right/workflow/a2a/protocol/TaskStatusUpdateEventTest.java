package ai.open.right.workflow.a2a.protocol;

import org.junit.jupiter.api.Test;

import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

public class TaskStatusUpdateEventTest {

    @Test
    void testGetSet() {
        TaskStatusUpdateEvent event = TaskStatusUpdateEvent.builder().build();
        Map<String, Object> metadata = Map.of("key", "val");
        event.setMetadata(metadata);
        assertSame(metadata, event.getMetadata());
        TaskStatus status = TaskStatus.builder().build();
        event.setStatus(status);
        assertSame(status, event.getStatus());
        event.setContextId("ctx123");
        assertEquals("ctx123", event.getContextId());
        event.setFinished(true);
        assertTrue(event.getFinished());
        event.setTaskId("task123");
        assertEquals("task123", event.getTaskId());
        event.setKind("custom-kind");
        assertEquals("custom-kind", event.getKind());
    }

    @Test
    void testBuilder() {
        Map<String, Object> metadata = Map.of("key", "val");
        TaskStatus status = TaskStatus.builder().build();
        TaskStatusUpdateEvent event = TaskStatusUpdateEvent.builder()
                .metadata(metadata)
                .status(status)
                .contextId("ctx123")
                .finished(true)
                .taskId("task123")
                .kind("custom-kind")
                .build();
        assertSame(metadata, event.getMetadata());
        assertSame(status, event.getStatus());
        assertEquals("ctx123", event.getContextId());
        assertTrue(event.getFinished());
        assertEquals("task123", event.getTaskId());
        assertEquals("custom-kind", event.getKind());
        TaskStatusUpdateEvent defaultEvent = TaskStatusUpdateEvent.builder().build();
        assertFalse(defaultEvent.getFinished());
        assertEquals("status-update", defaultEvent.getKind());
    }
}