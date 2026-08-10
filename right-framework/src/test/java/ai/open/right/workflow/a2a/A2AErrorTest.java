package ai.open.right.workflow.a2a;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class A2AErrorTest {

    @Test
    void testA2AErrorConstants() {
        assertEquals(-32601, A2AError.METHOD_NOT_FOUND);
        assertEquals(-32600, A2AError.INVALID_REQUEST);
        assertEquals(-32602, A2AError.INVALID_PARAMS);
        assertEquals(-32603, A2AError.INTERNAL_ERROR);
        assertEquals(-32700, A2AError.PARSE_ERROR);
    }

    @Test
    void testA2AErrorBuilder() {
        String testMessage = "测试错误信息";
        Integer testCode = A2AError.INVALID_PARAMS;
        String testData = "错误详情";
        A2AError error = A2AError.builder()
                .message(testMessage)
                .code(testCode)
                .data(testData)
                .build();
        assertEquals(testMessage, error.getMessage());
        assertEquals(testCode, error.getCode());
        assertEquals(testData, error.getData());
    }

    @Test
    void testSettersAndGetters() {
        A2AError error = A2AError.builder().build();
        String newMessage = "新错误信息";
        Integer newCode = A2AError.PARSE_ERROR;
        Object newData = new Object();
        error.setMessage(newMessage);
        error.setCode(newCode);
        error.setData(newData);
        assertEquals(newMessage, error.getMessage());
        assertEquals(newCode, error.getCode());
        assertSame(newData, error.getData());
    }
}