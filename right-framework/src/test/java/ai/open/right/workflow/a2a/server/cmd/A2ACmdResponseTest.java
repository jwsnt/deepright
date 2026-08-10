package ai.open.right.workflow.a2a.server.cmd;

import ai.open.right.protocol.ProtocolCode;
import org.junit.Test;

import static org.junit.Assert.*;

public class A2ACmdResponseTest {

    @Test
    public void testGetterAndSetter() {
        A2ACmdResponse response = A2ACmdResponse.builder().build();
        response.setFinished(true);
        assertTrue(response.getFinished());
        assertTrue(response.isFinished());
        response.setJsonrpc("3.0");
        assertEquals("3.0", response.getJsonrpc());
        Object result = new Object();
        response.setResult(result);
        assertSame(result, response.getResult());
        Object id = new Object();
        response.setId(id);
        assertSame(id, response.getId());
        assertEquals(ProtocolCode.C200, response.getCode());
    }

    /** 覆盖 @NoArgsConstructor：无参构造 */
    @Test
    public void testNoArgsConstructor() {
        A2ACmdResponse response = new A2ACmdResponse();
        assertFalse(response.getFinished());
        assertFalse(response.isFinished());
        assertEquals("2.0", response.getJsonrpc());
        assertNull(response.getResult());
        assertNull(response.getId());
        assertEquals(ProtocolCode.C200, response.getCode());
    }

    /** 覆盖 @Builder：全字段构建 */
    @Test
    public void testBuilderAllFields() {
        Object id = 100L;
        Object result = new Object();
        A2ACmdResponse response = A2ACmdResponse.builder()
                .finished(true)
                .jsonrpc("2.0")
                .result(result)
                .id(id)
                .build();
        assertTrue(response.getFinished());
        assertEquals("2.0", response.getJsonrpc());
        assertSame(result, response.getResult());
        assertSame(id, response.getId());
    }
}