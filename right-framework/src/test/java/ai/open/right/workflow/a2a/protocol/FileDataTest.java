package ai.open.right.workflow.a2a.protocol;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

public class FileDataTest {

    @Test
    void testMethods() {
        FileData fileData = new FileData();
        fileData.setMimeType("application/pdf");
        assertEquals("application/pdf", fileData.getMimeType());
        fileData.setBytes("base64data");
        assertEquals("base64data", fileData.getBytes());
        fileData.setName("test.pdf");
        assertEquals("test.pdf", fileData.getName());
        fileData.setUri(null);
        assertNull(fileData.getUri());
        assertEquals("base64data", fileData.getContent());
        assertFalse(fileData.isBytes());
        assertTrue(fileData.isUri());
        fileData.setBytes(null);
        fileData.setUri("file://test.pdf");
        assertEquals("file://test.pdf", fileData.getUri());
        assertEquals("file://test.pdf", fileData.getContent());
        assertTrue(fileData.isBytes());
        assertFalse(fileData.isUri());
        fileData.setBytes("");
        fileData.setUri("");
        assertEquals("", fileData.getContent());
        assertTrue(fileData.isBytes());
        assertTrue(fileData.isUri());
    }

    @Test
    public void testBuild() {
        FileData fileData = FileData.builder()
                .mimeType("application/pdf")
                .bytes("base64data")
                .name("test.pdf")
                .uri(null)
                .build();
        assertEquals("application/pdf", fileData.getMimeType());
        assertEquals("base64data", fileData.getBytes());
        assertEquals("test.pdf", fileData.getName());
        fileData.setUri(null);
        assertNull(fileData.getUri());
        assertEquals("base64data", fileData.getContent());
        assertFalse(fileData.isBytes());
        assertTrue(fileData.isUri());
        fileData.setBytes(null);
        fileData.setUri("file://test.pdf");
        assertEquals("file://test.pdf", fileData.getUri());
        assertEquals("file://test.pdf", fileData.getContent());
        assertTrue(fileData.isBytes());
        assertFalse(fileData.isUri());
        fileData.setBytes("");
        fileData.setUri("");
        assertEquals("", fileData.getContent());
        assertTrue(fileData.isBytes());
        assertTrue(fileData.isUri());
    }
}