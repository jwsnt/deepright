package ai.open.right.workflow.a2a.protocol;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

import java.util.HashMap;
import java.util.Map;

public class MessageRequestTest {

    @Test
    void testGetSetAndPutIfAbsent() {
        MessageRequest request = new MessageRequest();
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("key1", "val1");
        request.setMetadata(metadata);
        assertSame(metadata, request.getMetadata());
        Message message = new Message();
        request.setMessage(message);
        assertSame(message, request.getMessage());
        Map<String, Object> newMeta = new HashMap<>();
        newMeta.put("key1", "newVal1");
        newMeta.put("key2", "val2");
        request.putIfAbsent(newMeta);
        assertEquals("val1", request.getMetadata().get("key1"));
        assertEquals("val2", request.getMetadata().get("key2"));
        request.putIfAbsent(null);
        request.putIfAbsent(new HashMap<>());
        assertEquals(2, request.getMetadata().size());
    }
}