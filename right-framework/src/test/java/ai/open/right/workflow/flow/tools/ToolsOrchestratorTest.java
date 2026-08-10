package ai.open.right.workflow.flow.tools;

import org.junit.Test;

import static org.junit.Assert.*;

public class ToolsOrchestratorTest {

    @Test
    public void testMergeWithNull() throws Exception {
        ToolsOrchestrator target = new ToolsOrchestrator();
        ToolsOrchestrator result = target.merge(null);
        assertSame(target, result);
        assertNull(target.getBefore());
        assertNull(target.getParam());
        assertNull(target.getAfter());
    }

    @Test
    public void testMergeTargetAllNull() throws Exception {
        ToolsOrchestrator target = new ToolsOrchestrator();
        ToolsOrchestrator source = new ToolsOrchestrator();
        source.setBefore("before");
        source.setParam("param");
        source.setAfter("after");
        ToolsOrchestrator result = target.merge(source);
        assertEquals("before", result.getBefore());
        assertEquals("param", result.getParam());
        assertEquals("after", result.getAfter());
    }

    @Test
    public void testMergeSourceAllNull() throws Exception {
        ToolsOrchestrator target = new ToolsOrchestrator();
        target.setBefore("targetBefore");
        target.setParam("targetParam");
        target.setAfter("targetAfter");
        ToolsOrchestrator source = new ToolsOrchestrator();
        ToolsOrchestrator result = target.merge(source);
        assertEquals("targetBefore", result.getBefore());
        assertEquals("targetParam", result.getParam());
        assertEquals("targetAfter", result.getAfter());
    }

    @Test
    public void testMergePartialNull() throws Exception {
        ToolsOrchestrator target = new ToolsOrchestrator();
        target.setBefore("targetBefore");
        target.setParam(null);
        target.setAfter("targetAfter");
        ToolsOrchestrator source = new ToolsOrchestrator();
        source.setBefore(null);
        source.setParam("sourceParam");
        source.setAfter(null);
        ToolsOrchestrator result = target.merge(source);
        assertEquals("targetBefore", result.getBefore());
        assertEquals("sourceParam", result.getParam());
        assertEquals("targetAfter", result.getAfter());
    }

    @Test
    public void testMergeWithEmptyStrings() throws Exception {
        ToolsOrchestrator target = new ToolsOrchestrator();
        target.setBefore("");
        target.setParam("");
        target.setAfter("");
        ToolsOrchestrator source = new ToolsOrchestrator();
        source.setBefore("sourceBefore");
        source.setParam("sourceParam");
        source.setAfter("sourceAfter");
        ToolsOrchestrator result = target.merge(source);
        assertEquals("sourceBefore", result.getBefore());
        assertEquals("sourceParam", result.getParam());
        assertEquals("sourceAfter", result.getAfter());
    }

    @Test
    public void testMergeWithWhitespace() throws Exception {
        ToolsOrchestrator target = new ToolsOrchestrator();
        target.setBefore("   ");
        target.setParam("   ");
        target.setAfter("   ");
        ToolsOrchestrator source = new ToolsOrchestrator();
        source.setBefore("sourceBefore");
        source.setParam("sourceParam");
        source.setAfter("sourceAfter");
        ToolsOrchestrator result = target.merge(source);
        assertEquals("sourceBefore", result.getBefore());
        assertEquals("sourceParam", result.getParam());
        assertEquals("sourceAfter", result.getAfter());
    }
}