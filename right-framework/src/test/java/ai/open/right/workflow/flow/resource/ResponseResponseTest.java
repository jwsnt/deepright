package ai.open.right.workflow.flow.resource;

import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import java.util.HashMap;
import java.util.Map;

public class ResponseResponseTest {

    @Test
    public void testGetSet() {
        ResourceResponse response = new ResourceResponse();
        response.setContent("test content");
        Map<String, String> headers = new HashMap<>();
        headers.put("h1", "v1");
        response.setHeaders(headers);

        Assertions.assertEquals("test content", response.getContent());
        Assertions.assertEquals(headers, response.getHeaders());
    }

    @Test
    public void testAddHeader() {
        ResourceResponse response = new ResourceResponse();
        response.addHeader("h1", "v1");
        Assertions.assertEquals("v1", response.getHeaders().get("h1"));
        
        response.addHeader("h2", "v2");
        Assertions.assertEquals("v2", response.getHeaders().get("h2"));
        Assertions.assertEquals(2, response.getHeaders().size());
    }
}
