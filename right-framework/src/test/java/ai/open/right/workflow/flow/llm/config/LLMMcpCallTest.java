package ai.open.right.workflow.flow.llm.config;

import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class LLMMcpCallTest {

    @Test
    public void testGetSet() {
        LLMMcpCall llmMcpCall = new LLMMcpCall();
        llmMcpCall.setName("NAME");
        llmMcpCall.setQuery("QUERY");
        llmMcpCall.setReplace(true);
        llmMcpCall.setDynamic("DYNAMIC");
        llmMcpCall.setTimeout(1000);
        Map<String, Object> args = new HashMap<>();
        llmMcpCall.setArguments(args);
        Assert.assertEquals("NAME", llmMcpCall.getName());
        Assert.assertEquals("QUERY", llmMcpCall.getQuery());
        Assert.assertEquals("DYNAMIC", llmMcpCall.getDynamic());
        Assert.assertEquals(Integer.valueOf(1000), llmMcpCall.getTimeout());
        Assert.assertEquals(args, llmMcpCall.getArguments());
        Assert.assertTrue(llmMcpCall.getReplace());
    }

    @Test
    public void testArgs() {
        LLMMcpCall llmMcpCall = new LLMMcpCall();
        Map<String, Object> args = new HashMap<>();
        llmMcpCall.setArguments(args);
        Assert.assertEquals(args, llmMcpCall.arguments("HELLO"));
    }

    @Test
    public void testArgsWithQuery() {
        LLMMcpCall llmMcpCall = new LLMMcpCall();
        Map<String, Object> args = new HashMap<>();
        llmMcpCall.setArguments(args);
        llmMcpCall.setQuery("QUERY");
        Assert.assertEquals("HELLO", llmMcpCall.arguments("HELLO").get("QUERY"));
    }

    @Test
    public void testHashCode() throws Exception {
        Object object = LLMMcpCall.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testMerge() throws Exception {
        LLMMcpCall original = new LLMMcpCall();
        original.setName("ORIGINAL");
        original.setQuery("ORIGINAL_QUERY");
        original.setReplace(true);
        Map<String, Object> originalArgs = new HashMap<>();
        originalArgs.put("key1", "value1");
        original.setArguments(originalArgs);
        original.setRewriter("ORIGINAL_REWRITER");
        original.setClient("ORIGINAL_CLIENT");
        original.setTimeout(500);
        LLMMcpCall other = new LLMMcpCall();
        other.setName("OTHER");
        other.setQuery("OTHER_QUERY");
        other.setReplace(false);
        Map<String, Object> otherArgs = new HashMap<>();
        otherArgs.put("key2", "value2");
        other.setArguments(otherArgs);
        other.setRewriter("OTHER_REWRITER");
        other.setClient("OTHER_CLIENT");
        other.setTimeout(1000);
        LLMMcpCall merged = original.merge(other);
        Assert.assertEquals("ORIGINAL", merged.getName());
        Assert.assertEquals("ORIGINAL_QUERY", merged.getQuery());
        Assert.assertTrue(merged.getReplace());
        Assert.assertEquals("value1", merged.getArguments().get("key1"));
        Assert.assertEquals("value2", merged.getArguments().get("key2"));
        Assert.assertEquals("ORIGINAL_REWRITER", merged.getRewriter());
        Assert.assertEquals("ORIGINAL_CLIENT", merged.getClient());
        Assert.assertEquals(Integer.valueOf(500), merged.getTimeout());
    }

    @Test
    public void testMergeWithNullOriginalFields() throws Exception {
        LLMMcpCall original = new LLMMcpCall();
        original.setName(null);
        original.setQuery(null);
        original.setReplace(null);
        original.setArguments(null);
        original.setRewriter(null);
        original.setClient(null);
        original.setTimeout(null);
        LLMMcpCall other = new LLMMcpCall();
        other.setName("OTHER");
        other.setQuery("OTHER_QUERY");
        other.setReplace(false);
        Map<String, Object> otherArgs = new HashMap<>();
        otherArgs.put("key2", "value2");
        other.setArguments(otherArgs);
        other.setRewriter("OTHER_REWRITER");
        other.setClient("OTHER_CLIENT");
        other.setTimeout(1000);
        LLMMcpCall merged = original.merge(other);
        Assert.assertEquals("OTHER", merged.getName());
        Assert.assertEquals("OTHER_QUERY", merged.getQuery());
        Assert.assertFalse(merged.getReplace());
        Assert.assertEquals(otherArgs, merged.getArguments());
        Assert.assertEquals("OTHER_REWRITER", merged.getRewriter());
        Assert.assertEquals("OTHER_CLIENT", merged.getClient());
        Assert.assertEquals(Integer.valueOf(1000), merged.getTimeout());
    }

    @Test
    public void testMergeWithNullOther() throws Exception {
        LLMMcpCall original = new LLMMcpCall();
        original.setName("ORIGINAL");
        original.setQuery("ORIGINAL_QUERY");
        original.setReplace(true);
        Map<String, Object> originalArgs = new HashMap<>();
        originalArgs.put("key1", "value1");
        original.setArguments(originalArgs);
        original.setRewriter("ORIGINAL_REWRITER");
        original.setClient("ORIGINAL_CLIENT");
        original.setTimeout(500);
        LLMMcpCall merged = original.merge(null);
        Assert.assertEquals("ORIGINAL", merged.getName());
        Assert.assertEquals("ORIGINAL_QUERY", merged.getQuery());
        Assert.assertTrue(merged.getReplace());
        Assert.assertEquals(originalArgs, merged.getArguments());
        Assert.assertEquals("ORIGINAL_REWRITER", merged.getRewriter());
        Assert.assertEquals("ORIGINAL_CLIENT", merged.getClient());
        Assert.assertEquals(Integer.valueOf(500), merged.getTimeout());
    }

    @Test
    public void testHasRewriter() {
        LLMMcpCall call = new LLMMcpCall();
        call.setRewriter("");
        Assert.assertFalse(call.hasRewriter());
        call.setRewriter("R");
        Assert.assertTrue(call.hasRewriter());
    }

    @Test
    public void testGetReplaceNull() {
        LLMMcpCall call = new LLMMcpCall();
        call.setReplace(null);
        Assert.assertFalse(call.getReplace());
    }
}
