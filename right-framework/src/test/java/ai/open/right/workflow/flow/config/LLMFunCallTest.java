package ai.open.right.workflow.flow.config;

import ai.open.right.workflow.flow.llm.config.LLMFunCall;
import com.google.common.collect.ImmutableMap;
import org.junit.Assert;
import org.junit.Test;

import java.util.*;

public class LLMFunCallTest {

    @Test
    public void test() {
        Map<String, Object> properties = new HashMap<String, Object>();
        List<String> required = new ArrayList<String>();
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setProperties(properties);
        llmFunCall.setRequired(required);
        Assert.assertEquals(llmFunCall.getProperties(), properties);
        Assert.assertEquals(llmFunCall.getRequired(), required);
    }

    @Test
    public void testContain() {
        LLMFunCall llmFunCall = new LLMFunCall();
        Assert.assertTrue(llmFunCall.allowed("HELLO"));
        Assert.assertTrue(llmFunCall.allowed("WORLD"));
        llmFunCall.setWhiteList(Arrays.asList("WORLD"));
        Assert.assertFalse(llmFunCall.allowed("HELLO"));
        Assert.assertTrue(llmFunCall.allowed("WORLD"));
    }

    @Test
    public void testAllowedWithOutList() {
        LLMFunCall llmFunCall = new LLMFunCall();
        Assert.assertTrue(llmFunCall.allowed("OK"));
    }

    @Test
    public void testAllowedWithWhiteList() {
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setWhiteList(Arrays.asList("OK", "YES"));
        Assert.assertTrue(llmFunCall.allowed("OK"));
        Assert.assertTrue(llmFunCall.allowed("YES"));
        Assert.assertFalse(llmFunCall.allowed("NO"));
    }

    @Test
    public void testAllowedWithBlackList() {
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setBlackList(Arrays.asList("OK", "YES"));
        Assert.assertFalse(llmFunCall.allowed("OK"));
        Assert.assertFalse(llmFunCall.allowed("YES"));
        Assert.assertTrue(llmFunCall.allowed("NO"));
    }

    @Test
    public void testAllowedWithWhiteAndBlackList() {
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setWhiteList(Arrays.asList("OK"));
        llmFunCall.setBlackList(Arrays.asList("OK", "YES"));
        Assert.assertTrue(llmFunCall.allowed("OK"));
        Assert.assertFalse(llmFunCall.allowed("YES"));
        Assert.assertFalse(llmFunCall.allowed("NO"));
    }

    @Test
    public void testMerge() throws Exception {
        LLMFunCall llmFunCall1 = new LLMFunCall();
        llmFunCall1.setProperties(ImmutableMap.of("A", "B"));
        llmFunCall1.setSuffix("_suffix");
        llmFunCall1.setPrefix("_prefix");
        llmFunCall1.setName("_name");
        llmFunCall1.setDescription("_desc");
        llmFunCall1.setRefer(false);
        llmFunCall1.setRequired(List.of("C"));
        llmFunCall1.setBlackList(List.of("D"));
        llmFunCall1.setWhiteList(List.of("E"));
        LLMFunCall llmFunCall2 = new LLMFunCall();
        llmFunCall2.setWhiteList(List.of("F"));
        llmFunCall2.setPrefix("prefix");
        llmFunCall2.merge(llmFunCall1);
        Assert.assertEquals("F", llmFunCall2.getWhiteList().getFirst());
        Assert.assertEquals("prefix", llmFunCall2.getPrefix());
        Assert.assertEquals("_suffix", llmFunCall2.getSuffix());
        Assert.assertEquals("_name", llmFunCall2.getName());
        Assert.assertEquals("_desc", llmFunCall2.getDescription());
        Assert.assertEquals("B", llmFunCall2.getProperties().get("A"));
        Assert.assertEquals(false, llmFunCall2.getRefer());
        Assert.assertEquals("C", llmFunCall2.getRequired().getFirst());
        Assert.assertEquals("D", llmFunCall2.getBlackList().getFirst());
    }

    @Test
    public void testLooped() {
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setName("NAME");
        Assert.assertFalse(llmFunCall.isLooped("BIZ", "WORKFLOW"));
        Assert.assertTrue(llmFunCall.isLooped("BIZ", "NAME"));
        llmFunCall.setName("B@NAME");
        Assert.assertFalse(llmFunCall.isLooped("BIZ", "NAME"));
        Assert.assertTrue(llmFunCall.isLooped("B", "NAME"));
    }
}
