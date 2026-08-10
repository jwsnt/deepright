package ai.open.right.workflow.flow.resource;

import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

public class ResourceConfigTest {

    @Test
    public void testGetSet() {
        ResourceConfig config = new ResourceConfig();
        config.setTimeout(1000);
        Assertions.assertEquals(Integer.valueOf(1000), config.getTimeout());
    }

    @Test
    public void testMerge() throws Exception {
        ResourceConfig config1 = new ResourceConfig();
        config1.setTimeout(1000);
        
        ResourceConfig config2 = new ResourceConfig();
        config2.setTimeout(2000);
        
        config1.merge(config2);
        Assertions.assertEquals(Integer.valueOf(1000), config1.getTimeout());
        
        ResourceConfig config3 = new ResourceConfig();
        config3.merge(config2);
        Assertions.assertEquals(Integer.valueOf(2000), config3.getTimeout());
        
        config3.merge(null);
        Assertions.assertEquals(Integer.valueOf(2000), config3.getTimeout());
    }

    /** 覆盖 headers getter/setter：初始为 null，设置后返回设置的值。 */
    @Test
    public void testHeaders_getSet() {
        ResourceConfig config = new ResourceConfig();
        Assertions.assertNull(config.getHeaders());
        
        Map<String, String> headers = new HashMap<>();
        headers.put("Authorization", "Bearer token123");
        headers.put("Content-Type", "application/json");
        config.setHeaders(headers);
        
        Assertions.assertNotNull(config.getHeaders());
        Assertions.assertEquals(2, config.getHeaders().size());
        Assertions.assertEquals("Bearer token123", config.getHeaders().get("Authorization"));
        Assertions.assertEquals("application/json", config.getHeaders().get("Content-Type"));
    }

    /** 覆盖 hasHeaders()：headers 为 null 时返回 false。 */
    @Test
    public void testHasHeaders_nullReturnsFalse() {
        ResourceConfig config = new ResourceConfig();
        Assertions.assertNull(config.getHeaders());
        Assertions.assertFalse(config.hasHeaders());
    }

    /** 覆盖 hasHeaders()：headers 为空 Map 时返回 false。 */
    @Test
    public void testHasHeaders_emptyMapReturnsFalse() {
        ResourceConfig config = new ResourceConfig();
        config.setHeaders(new HashMap<>());
        Assertions.assertFalse(config.hasHeaders());
    }

    /** 覆盖 hasHeaders()：headers 有元素时返回 true。 */
    @Test
    public void testHasHeaders_withElementsReturnsTrue() {
        ResourceConfig config = new ResourceConfig();
        Map<String, String> headers = new HashMap<>();
        headers.put("Key", "Value");
        config.setHeaders(headers);
        Assertions.assertTrue(config.hasHeaders());
    }

    /** 覆盖 hasHeaders()：设置 headers 后，清空为 null 时返回 false。 */
    @Test
    public void testHasHeaders_setThenClearToNull() {
        ResourceConfig config = new ResourceConfig();
        Map<String, String> headers = new HashMap<>();
        headers.put("Key", "Value");
        config.setHeaders(headers);
        Assertions.assertTrue(config.hasHeaders());
        
        config.setHeaders(null);
        Assertions.assertFalse(config.hasHeaders());
    }
}
