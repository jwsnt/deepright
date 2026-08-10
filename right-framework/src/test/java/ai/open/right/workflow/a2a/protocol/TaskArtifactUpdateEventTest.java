package ai.open.right.workflow.a2a.protocol;

import com.google.common.collect.ImmutableMap;
import org.junit.Assert;
import org.junit.jupiter.api.Test;

import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

public class TaskArtifactUpdateEventTest {

    @Test
    void testGetSet() {
        TaskArtifactUpdateEvent event = TaskArtifactUpdateEvent.builder().build();
        Map<String, Object> metadata = Map.of("key", "val");
        event.setMetadata(metadata);
        assertSame(metadata, event.getMetadata());
        Artifact artifact = Artifact.builder().build();
        event.setArtifact(artifact);
        assertSame(artifact, event.getArtifact());
        event.setContextId("ctx123");
        assertEquals("ctx123", event.getContextId());
        event.setLastChunk(true);
        assertTrue(event.getLastChunk());
        event.setAppend(true);
        assertTrue(event.getAppend());
        event.setTaskId("task123");
        assertEquals("task123", event.getTaskId());
        event.setKind("custom-kind");
        assertEquals("custom-kind", event.getKind());
        assertEquals(event.getInternal(), TaskArtifactUpdateEvent.PROTOCOL);
        event.reset();
        assertNull(event.getInternal());
    }

    @Test
    void testBuilder() {
        Map<String, Object> metadata = Map.of("key", "val");
        Artifact artifact = Artifact.builder().build();
        TaskArtifactUpdateEvent event = TaskArtifactUpdateEvent.builder()
                .metadata(metadata)
                .artifact(artifact)
                .contextId("ctx123")
                .lastChunk(true)
                .append(true)
                .taskId("task123")
                .kind("custom-kind")
                .build();
        assertSame(metadata, event.getMetadata());
        assertSame(artifact, event.getArtifact());
        assertEquals("ctx123", event.getContextId());
        assertTrue(event.getLastChunk());
        assertTrue(event.getAppend());
        assertEquals("task123", event.getTaskId());
        assertEquals("custom-kind", event.getKind());
        TaskArtifactUpdateEvent defaultEvent = TaskArtifactUpdateEvent.builder().build();
        assertFalse(defaultEvent.getLastChunk());
        assertFalse(defaultEvent.getAppend());
        assertEquals("artifact-update", defaultEvent.getKind());
    }

    @Test
    void testSet() {
        TaskArtifactUpdateEvent task = TaskArtifactUpdateEvent.builder().build();
        assertNull(task.getTaskId());
        task.taskId("ART");
        assertEquals("ART", task.getTaskId());
        task.taskId("ART1");
        assertEquals("ART", task.getTaskId());
        assertNull(task.getMetadata());
        task.metadata(ImmutableMap.of("A", "B"));
        assertEquals("B", task.getMetadata().get("A"));
        task.metadata(ImmutableMap.of("A", "B1"));
        task.metadata(ImmutableMap.of("C", "D"));
        assertEquals("B", task.getMetadata().get("A"));
        assertEquals("D", task.getMetadata().get("C"));
        assertFalse(task.isSupport("A"));
        assertTrue(task.isSupport(TaskArtifactUpdateEvent.PROTOCOL));
        assertEquals(false, task.getLastChunk());
        task.lastChunk(true);
        assertEquals(false, task.getLastChunk());
        task.setLastChunk(null);
        task.lastChunk(true);
        assertEquals(true, task.getLastChunk());
        assertEquals(false, task.getAppend());
        task.append(true);
        Assert.assertEquals(false, task.getAppend());
        task.setAppend(null);
        task.append(true);
        Assert.assertEquals(true, task.getAppend());
        assertNull(task.getContextId());
        task.contextId("CONTEXT");
        assertEquals("CONTEXT", task.getContextId());
        task.contextId("CONTEXT2");
        assertEquals("CONTEXT", task.getContextId());
    }
}