package ai.open.right.workflow.flow.function;

import ai.open.right.protocol.ProtocolCode;
import org.junit.Assert;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;

class FunctionResponseTest {

    @Test
    public void testFunctionResponse() {
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("key", "value");
        Object content = new Object();
        FunctionResponse response = FunctionResponse.builder()
                .metadata(metadata)
                .content(content)
                .code(ProtocolCode.C200)
                .build();
        assertNotNull(response);
        assertEquals(metadata, response.getMetadata());
        assertEquals(content, response.getContent());
        assertEquals(ProtocolCode.C200, response.getCode());
    }

    @Test
    public void testDefaultCode() {
        FunctionResponse response = FunctionResponse.builder().build();
        assertEquals(ProtocolCode.C200, response.getCode());
    }

    @Test
    public void testSetGet() {
        FunctionResponse response = FunctionResponse.builder().build();
        response.setMetadata(new HashMap<>());
        response.setCode(200);
        response.setContent("OK");
        assertNotNull(response.getMetadata());
        assertEquals(Integer.valueOf(200), response.getCode());
        assertEquals("OK", response.getContent());
    }

    @Test
    public void testToString() {
        FunctionResponse response = FunctionResponse.builder().build();
        response.setMetadata(new HashMap<>());
        response.setCode(200);
        response.setContent("OK");
        Assert.assertEquals("FunctionResponse(metadata={}, content=OK, code=200)", response.toString());
    }
}