package ai.open.right.workflow.a2a.protocol;

import com.google.common.collect.ImmutableMap;
import org.junit.Assert;
import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

public class TaskTest {

    @Test
    void testGetSet() {
        Task task = Task.builder().build();
        Map<String, Object> metadata = Map.of("key", "val");
        task.setMetadata(metadata);
        assertSame(metadata, task.getMetadata());
        List<Artifact> artifacts = List.of(Artifact.builder().build());
        task.setArtifacts(artifacts);
        assertSame(artifacts, task.getArtifacts());
        List<Message> history = List.of(new Message());
        task.setHistory(history);
        assertSame(history, task.getHistory());
        TaskStatus status = TaskStatus.builder().build();
        task.setStatus(status);
        assertSame(status, task.getStatus());
        task.setContextId("ctx123");
        assertEquals("ctx123", task.getContextId());
        task.setTimestamp("2024-01-01");
        assertEquals("2024-01-01", task.getTimestamp());
        task.setKind("custom");
        assertEquals("custom", task.getKind());
        task.setId("id123");
        assertEquals("id123", task.getId());
        assertEquals(task.getInternal(), Task.PROTOCOL);
        task.reset();
        assertNull(task.getInternal());
    }

    @Test
    void testBuilder() {
        Map<String, Object> metadata = Map.of("key", "val");
        List<Artifact> artifacts = List.of(Artifact.builder().build());
        List<Message> history = List.of(new Message());
        TaskStatus status = TaskStatus.builder().build();
        Task task = Task.builder()
                .metadata(metadata)
                .artifacts(artifacts)
                .history(history)
                .status(status)
                .contextId("ctx123")
                .timestamp("2024-01-01")
                .kind("custom")
                .id("id123")
                .build();
        assertSame(metadata, task.getMetadata());
        assertSame(artifacts, task.getArtifacts());
        assertSame(history, task.getHistory());
        assertSame(status, task.getStatus());
        assertEquals("ctx123", task.getContextId());
        assertEquals("2024-01-01", task.getTimestamp());
        assertEquals("custom", task.getKind());
        assertEquals("id123", task.getId());
        Task defaultKindTask = Task.builder().build();
        assertEquals("task", defaultKindTask.getKind());
    }

    @Test
    void testToString() {
        Task task = Task.builder().id("test-id").build();
        assertTrue(task.toString().contains("test-id"));
    }

    @Test
    void testSet() {
        Task task = Task.builder().build();
        assertNull(task.getId());
        task.id("ART");
        assertEquals("ART", task.getId());
        task.id("ART1");
        assertEquals("ART", task.getId());
        assertNull(task.getMetadata());
        task.metadata(ImmutableMap.of("A", "B"));
        assertEquals("B", task.getMetadata().get("A"));
        task.metadata(ImmutableMap.of("A", "B1"));
        task.metadata(ImmutableMap.of("C", "D"));
        assertEquals("B", task.getMetadata().get("A"));
        assertEquals("D", task.getMetadata().get("C"));
        assertFalse(task.isSupport("A"));
        assertTrue(task.isSupport(Task.PROTOCOL));
        assertNull(task.getTimestamp());
        task.timestamp("A");
        assertEquals("A", task.getTimestamp());
        task.timestamp("B");
        assertEquals("A", task.getTimestamp());
        assertNull(task.getContextId());
        task.contextId("A");
        assertEquals("A", task.getContextId());
        task.contextId("B");
        assertEquals("A", task.getContextId());
        assertNull(task.getStatus());
        task.status(TaskStatus.builder()
                .state(TaskStatus.STATUS_WORKING).build());
        assertEquals(TaskStatus.STATUS_WORKING, task.getStatus().getState());
        task.status(TaskStatus.builder()
                .state(TaskStatus.STATUS_FAILED).build());
        assertEquals(TaskStatus.STATUS_WORKING, task.getStatus().getState());
    }
}