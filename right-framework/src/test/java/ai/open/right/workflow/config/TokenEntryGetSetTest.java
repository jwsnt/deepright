package ai.open.right.workflow.config;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

class TokenEntryGetSetTest {

    @Test
    void testBuilderAndGetters() {
        TokenEntry entry = TokenEntry.builder()
                .workflow("test-workflow")
                .biz("test-biz")
                .build();
        assertEquals("test-workflow", entry.getWorkflow());
        assertEquals("test-biz", entry.getBiz());
    }

    @Test
    void testSetters() {
        TokenEntry entry = TokenEntry.builder().build();
        entry.setWorkflow("new-workflow");
        entry.setBiz("new-biz");
        assertEquals("new-workflow", entry.getWorkflow());
        assertEquals("new-biz", entry.getBiz());
    }

    @Test
    void testToString() {
        TokenEntry entry = TokenEntry.builder()
                .workflow("test-wf")
                .biz("test-bz")
                .build();
        String str = entry.toString();
        assertTrue(str.contains("workflow=test-wf"));
        assertTrue(str.contains("biz=test-bz"));
    }

    @Test
    void testNullValues() {
        TokenEntry entry = TokenEntry.builder().build();
        assertNull(entry.getWorkflow());
        assertNull(entry.getBiz());
        entry.setWorkflow(null);
        entry.setBiz(null);
        assertNull(entry.getWorkflow());
        assertNull(entry.getBiz());
    }
}