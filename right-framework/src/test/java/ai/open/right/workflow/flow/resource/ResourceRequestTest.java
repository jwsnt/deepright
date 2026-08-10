package ai.open.right.workflow.flow.resource;

import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import java.util.HashMap;
import java.util.Map;

public class ResourceRequestTest {

    @Test
    public void testGetSet() {
        ResourceRequest request = new ResourceRequest();
        request.setUrl("http://test.com");
        request.setMethod("POST");
        Map<String, String> headers = new HashMap<>();
        headers.put("key", "value");
        request.setHeaders(headers);
        Map<String, Object> content = new HashMap<>();
        content.put("a", "b");
        request.setContent(content);

        Assertions.assertEquals("http://test.com", request.getUrl());
        Assertions.assertEquals("POST", request.getMethod());
        Assertions.assertEquals(headers, request.getHeaders());
        Assertions.assertEquals(content, request.getContent());
    }

    @Test
    public void testIsValid() {
        ResourceRequest request = new ResourceRequest();
        Assertions.assertFalse(request.isValid());
        
        request.setUrl("http://test.com");
        Assertions.assertTrue(request.isValid());
        
        request.setMethod("");
        Assertions.assertFalse(request.isValid());
        
        request.setMethod("POST");
        Assertions.assertTrue(request.isValid());
    }

    /** 覆盖 putContent：content 为 null 时，创建新 Map 并 put。 */
    @Test
    public void testPutContent_nullContent_createsNewMap() {
        ResourceRequest request = new ResourceRequest();
        Assertions.assertNull(request.getContent());
        
        request.putContent("key1", "value1");
        
        Assertions.assertNotNull(request.getContent());
        Assertions.assertEquals("value1", request.getContent().get("key1"));
    }

    /** 覆盖 putContent：content 不为 null 时，添加到现有 Map。 */
    @Test
    public void testPutContent_existingContent_addsToMap() {
        ResourceRequest request = new ResourceRequest();
        Map<String, Object> existingContent = new HashMap<>();
        existingContent.put("existing", "value");
        request.setContent(existingContent);
        
        request.putContent("newKey", "newValue");
        
        Assertions.assertEquals(2, request.getContent().size());
        Assertions.assertEquals("value", request.getContent().get("existing"));
        Assertions.assertEquals("newValue", request.getContent().get("newKey"));
    }

    /** 覆盖 putContent：多次调用 put，覆盖已有 key。 */
    @Test
    public void testPutContent_multiplePuts_overwritesExisting() {
        ResourceRequest request = new ResourceRequest();
        request.putContent("key", "value1");
        Assertions.assertEquals("value1", request.getContent().get("key"));
        
        request.putContent("key", "value2");
        Assertions.assertEquals("value2", request.getContent().get("key"));
        Assertions.assertEquals(1, request.getContent().size());
    }

    /** 覆盖 putContent：put 不同类型的 value。 */
    @Test
    public void testPutContent_differentValueTypes() {
        ResourceRequest request = new ResourceRequest();
        request.putContent("string", "text");
        request.putContent("number", 123);
        request.putContent("boolean", true);
        request.putContent("null", null);
        
        Assertions.assertEquals("text", request.getContent().get("string"));
        Assertions.assertEquals(123, request.getContent().get("number"));
        Assertions.assertEquals(true, request.getContent().get("boolean"));
        Assertions.assertNull(request.getContent().get("null"));
    }

    /** 覆盖 putHeader：headers 为 null 时，创建新 Map 并 put。 */
    @Test
    public void testPutHeader_nullHeaders_createsNewMap() {
        ResourceRequest request = new ResourceRequest();
        Assertions.assertNull(request.getHeaders());
        
        request.putHeader("Authorization", "Bearer token");
        
        Assertions.assertNotNull(request.getHeaders());
        Assertions.assertEquals("Bearer token", request.getHeaders().get("Authorization"));
    }

    /** 覆盖 putHeader：headers 不为 null 时，添加到现有 Map。 */
    @Test
    public void testPutHeader_existingHeaders_addsToMap() {
        ResourceRequest request = new ResourceRequest();
        Map<String, String> existingHeaders = new HashMap<>();
        existingHeaders.put("Content-Type", "application/json");
        request.setHeaders(existingHeaders);
        
        request.putHeader("Authorization", "Bearer token");
        
        Assertions.assertEquals(2, request.getHeaders().size());
        Assertions.assertEquals("application/json", request.getHeaders().get("Content-Type"));
        Assertions.assertEquals("Bearer token", request.getHeaders().get("Authorization"));
    }

    /** 覆盖 putHeader：多次调用 put，覆盖已有 key。 */
    @Test
    public void testPutHeader_multiplePuts_overwritesExisting() {
        ResourceRequest request = new ResourceRequest();
        request.putHeader("X-Custom", "value1");
        Assertions.assertEquals("value1", request.getHeaders().get("X-Custom"));
        
        request.putHeader("X-Custom", "value2");
        Assertions.assertEquals("value2", request.getHeaders().get("X-Custom"));
        Assertions.assertEquals(1, request.getHeaders().size());
    }

    /** 覆盖 putHeader：put 多个不同的 headers。 */
    @Test
    public void testPutHeader_multipleHeaders() {
        ResourceRequest request = new ResourceRequest();
        request.putHeader("Header1", "value1");
        request.putHeader("Header2", "value2");
        request.putHeader("Header3", "value3");
        
        Assertions.assertEquals(3, request.getHeaders().size());
        Assertions.assertEquals("value1", request.getHeaders().get("Header1"));
        Assertions.assertEquals("value2", request.getHeaders().get("Header2"));
        Assertions.assertEquals("value3", request.getHeaders().get("Header3"));
    }

    /** 覆盖 hasHeader：headers 为 null 时返回 false。 */
    @Test
    public void hasHeader_nullHeaders_returnsFalse() {
        ResourceRequest request = new ResourceRequest();
        request.setHeaders(null);
        Assertions.assertFalse(request.hasHeaders());
    }

    /** 覆盖 hasHeader：headers 为空 Map 时返回 false。 */
    @Test
    public void hasHeader_emptyHeaders_returnsFalse() {
        ResourceRequest request = new ResourceRequest();
        request.setHeaders(new HashMap<>());
        Assertions.assertFalse(request.hasHeaders());
    }

    /** 覆盖 hasHeader：headers 有至少一个 entry 时返回 true。 */
    @Test
    public void hasHeader_withHeaders_returnsTrue() {
        ResourceRequest request = new ResourceRequest();
        Map<String, String> headers = new HashMap<>();
        headers.put("Authorization", "Bearer token");
        request.setHeaders(headers);
        Assertions.assertTrue(request.hasHeaders());
    }

    /** 覆盖 hasHeader：通过 putHeader 添加后返回 true。 */
    @Test
    public void hasHeader_afterPutHeader_returnsTrue() {
        ResourceRequest request = new ResourceRequest();
        Assertions.assertFalse(request.hasHeaders());
        request.putHeader("X-Custom", "value");
        Assertions.assertTrue(request.hasHeaders());
    }
}
