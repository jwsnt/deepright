package ai.open.right.workflow.flow.llm.signal.impl;

import ai.open.right.workflow.flow.llm.signal.SignalConfig;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.Assert.*;

public class SignalConfigTest {

    @Test
    public void testMergeWithNull() throws Exception {
        SignalConfig config = new SignalConfig();
        SignalConfig result = config.merge(null);
        assertSame(config, result);
        assertNull(result.getTimeout4Llm());
        assertNull(result.getSynthesizer());
        assertTrue(result.getSilent());
        assertTrue(result.getConfigs().isEmpty());
    }

    @Test
    public void testMergeTargetAllNull() throws Exception {
        SignalConfig source = new SignalConfig();
        Map<String, String> configs = new HashMap<>();
        configs.put("key", "value");
        source.setConfigs(configs);
        source.setTimeout4Llm(100);
        source.setSynthesizer("synth");
        source.setSilent(false);
        SignalConfig target = new SignalConfig();
        SignalConfig result = target.merge(source);
        assertEquals(configs, result.getConfigs());
        assertEquals(100, (int)result.getTimeout4Llm());
        assertEquals("synth", result.getSynthesizer());
        assertFalse(result.getSilent());
    }

    @Test
    public void testMergeSourceAllNull() throws Exception {
        SignalConfig target = new SignalConfig();
        Map<String, String> configs = new HashMap<>();
        configs.put("key", "value");
        target.setConfigs(configs);
        target.setTimeout4Llm(100);
        target.setSynthesizer("synth");
        target.setSilent(false);
        SignalConfig source = new SignalConfig();
        SignalConfig result = target.merge(source);
        assertEquals(configs, result.getConfigs());
        assertEquals(100, (int)result.getTimeout4Llm());
        assertEquals("synth", result.getSynthesizer());
        assertFalse(result.getSilent());
    }

    @Test
    public void testMergePartialOverlap() throws Exception {
        SignalConfig target = new SignalConfig();
        Map<String, String> targetConfigs = new HashMap<>();
        targetConfigs.put("tkey", "tvalue");
        target.setConfigs(targetConfigs);
        target.setTimeout4Llm(200);
        SignalConfig source = new SignalConfig();
        Map<String, String> sourceConfigs = new HashMap<>();
        sourceConfigs.put("skey", "svalue");
        source.setConfigs(sourceConfigs);
        source.setSynthesizer("ssynth");
        source.setSilent(true);
        SignalConfig result = target.merge(source);
        assertEquals("tvalue", result.getConfigs().get("tkey"));
        assertEquals("svalue", result.getConfigs().get("skey"));
        assertEquals(200, (int)result.getTimeout4Llm());
        assertEquals("ssynth", result.getSynthesizer());
        assertTrue(result.getSilent());
    }
}