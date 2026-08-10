package ai.open.right.workflow.a2a.protocol;

import com.google.common.collect.ImmutableMap;
import org.junit.Assert;
import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

public class ArtifactTest {

    @Test
    void testGetSet() {
        Artifact artifact = Artifact.builder().build();
        Map<String, Object> metadata = Map.of("key", "value");
        artifact.setMetadata(metadata);
        assertSame(metadata, artifact.getMetadata());
        artifact.setDescription("desc");
        assertEquals("desc", artifact.getDescription());
        artifact.setArtifactId("1");
        assertEquals("1", artifact.getArtifactId());
        List<Part> parts = List.of(new Part());
        artifact.setParts(parts);
        assertSame(parts, artifact.getParts());
        artifact.setName("name");
        assertEquals("name", artifact.getName());
        Artifact built = Artifact.builder().build();
        assertEquals(null, built.getArtifactId());
        assertEquals(artifact.getInternal(), Artifact.PROTOCOL);
        artifact.reset();
        assertNull(artifact.getInternal());
    }

    @Test
    void testSet() {
        Artifact artifact = Artifact.builder().build();
        assertNull(artifact.getArtifactId());
        artifact.artifactId("ART");
        assertEquals("ART", artifact.getArtifactId());
        artifact.artifactId("ART1");
        assertEquals("ART", artifact.getArtifactId());
        assertNull(artifact.getMetadata());
        artifact.metadata(ImmutableMap.of("A", "B"));
        assertEquals("B", artifact.getMetadata().get("A"));
        artifact.metadata(ImmutableMap.of("A", "B1"));
        artifact.metadata(ImmutableMap.of("C", "D"));
        assertEquals("B", artifact.getMetadata().get("A"));
        assertEquals("D", artifact.getMetadata().get("C"));
        assertFalse(artifact.isSupport("A"));
        assertTrue(artifact.isSupport(Artifact.PROTOCOL));
    }
}