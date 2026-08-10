package ai.open.right.workflow.a2a.protocol;

import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.Assert.*;


public class DataPartTest {

    @Test
    void testMethods() throws Exception {
        Map<String, Object> data = new HashMap<>();
        data.put("str", "test");
        data.put("num", 123);
        NestedObj nested = new NestedObj();
        nested.setField("nestedVal");
        data.put("nested", nested);
        DataPart dataPart = new DataPart(data);
        assertEquals("test", dataPart.getObject("str"));
        assertEquals(123, dataPart.getObject("num"));
        assertEquals(nested, dataPart.getObject("nested"));
        assertTrue(dataPart.equals("str", "test"));
        assertFalse(dataPart.equals("num", 456));
        String strVal = dataPart.getObject("str", String.class);
        assertEquals("test", strVal);
        Integer numVal = dataPart.getObject("num", Integer.class);
        assertEquals(Integer.valueOf(123), numVal);
        NestedObj nestedVal = dataPart.getObject("nested", NestedObj.class);
        assertEquals(nested, nestedVal);
        Map<String, Object> nestedMap = new HashMap<>();
        nestedMap.put("field", "mapVal");
        data.put("nestedMap", nestedMap);
        NestedObj mappedNested = dataPart.getObject("nestedMap", NestedObj.class);
        assertEquals("mapVal", mappedNested.getField());
        assertNotNull(dataPart.getData());
        dataPart.setData(null);
        assertNull(dataPart.getData());
    }

    static class NestedObj {
        private String field;

        public String getField() {
            return field;
        }

        public void setField(String field) {
            this.field = field;
        }
    }
}