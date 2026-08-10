package ai.open.right.workflow.a2a.protocol;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

import java.util.Map;

public class PartTest {

    @Test
    void testGetSet() {
        Part part = new Part();
        Map<String, Object> metadata = Map.of("m1", "v1");
        part.setMetadata(metadata);
        assertSame(metadata, part.getMetadata());
        Map<String, Object> data = Map.of("d1", "v1");
        part.setData(data);
        assertSame(data, part.getData());
        FileData file = new FileData();
        part.setFile(file);
        assertSame(file, part.getFile());
        part.setText("text");
        assertEquals("text", part.getText());
        part.setKind(Part.DATA_KIND);
        assertEquals(Part.DATA_KIND, part.getKind());
        assertTrue(part.isKind(Part.DATA_KIND));
        assertFalse(part.isKind(Part.FILE_KIND));
    }

    @Test
    void testConstructors() {
        Map<String, Object> metadata = Map.of("m1", "v1");
        Map<String, Object> data = Map.of("d1", "v1");
        FileData file = new FileData();
        Part part = new Part(metadata, data, file, "text", Part.FILE_KIND);
        assertSame(metadata, part.getMetadata());
        assertSame(data, part.getData());
        assertSame(file, part.getFile());
        assertEquals("text", part.getText());
        assertEquals(Part.FILE_KIND, part.getKind());
        assertTrue(part.isKind(Part.FILE_KIND));
    }

    @Test
    void testBuilder() {
        Map<String, Object> metadata = Map.of("m1", "v1");
        Map<String, Object> data = Map.of("d1", "v1");
        FileData file = new FileData();
        Part part = Part.builder()
                .metadata(metadata)
                .data(data)
                .file(file)
                .text("text")
                .kind(Part.DATA_KIND)
                .build();
        assertSame(metadata, part.getMetadata());
        assertSame(data, part.getData());
        assertSame(file, part.getFile());
        assertEquals("text", part.getText());
        assertEquals(Part.DATA_KIND, part.getKind());
        assertTrue(part.isKind(Part.DATA_KIND));
    }
}