package ai.open.right.workflow.a2a.protocol;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;
import java.util.List;
import java.util.Map;

public class MessageTest {

    @Test
    void testGetSet() {
        Message message = new Message();
        Map<String, Object> metadata = Map.of("key", "val");
        message.setMetadata(metadata);
        assertSame(metadata, message.getMetadata());
        List<Part> parts = List.of(new Part());
        message.setParts(parts);
        assertSame(parts, message.getParts());
        message.setMessageId("msg123");
        assertEquals("msg123", message.getMessageId());
        message.setContextId("ctx456");
        assertEquals("ctx456", message.getContextId());
        message.setTaskId("task789");
        assertEquals("task789", message.getTaskId());
        assertEquals("message", message.getKind());
        message.setRole("user");
        assertEquals("user", message.getRole());
    }
}