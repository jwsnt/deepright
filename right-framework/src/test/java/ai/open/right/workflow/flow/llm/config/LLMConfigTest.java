package ai.open.right.workflow.flow.llm.config;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.mcp.RagMcpConfig;
import org.junit.Assert;
import org.junit.Test;

import java.util.*;

public class LLMConfigTest {

    @Test
    public void test() throws Exception {
        LLMConfig config = new LLMConfig();
        Assert.assertFalse(config.hasChain());
        Assert.assertFalse(config.hasNotifier());
        Assert.assertFalse(config.getStoreFunCall());
        Assert.assertFalse(config.getRecallFunCall());
        Map<String, Object> additional = new HashMap<String, Object>();
        additional.put("Additional_K", "Additional_B");
        Assert.assertFalse(config.getRecallDesc());
        Assert.assertEquals(Integer.valueOf(0), config.getRecallOffset());
        config.setRecallDesc(true);
        config.setRecallOffset(1000);
        config.setPrintReason(true);
        config.setAdditional(additional);
        config.setProvider("Provider");
        config.setNotifier("NOTIFIER");
        config.setStoreFunCall(true);
        config.setRecallFunCall(true);
        config.setChain("CHAIN");
        Assert.assertTrue(config.hasNotifier());
        RagConfig ragConfig = new RagConfig();
        ragConfig.setMode(RagConfig.MODE_XML);
        config.setRagConfig(Arrays.asList(ragConfig));
        config.setDynamic(new LLMDynamic());
        config.setContainHistories(false);
        config.setStream(true);
        config.setMaxError(1024);
        config.setTokenBuffer(100);
        config.setTokenFirst(100);
        config.setHistories(100);
        Assert.assertFalse(config.hasNetworkBuffer());
        config.setNetworkBuffer(10240);
        Assert.assertEquals(Integer.valueOf(100), config.getRecallNums());
        config.setRecallNums(1000);
        Assert.assertEquals(Integer.valueOf(1000), config.getRecallNums());
        Assert.assertEquals(Integer.valueOf(10240), config.getNetworkBuffer());
        Assert.assertTrue(config.hasNetworkBuffer());
        Assert.assertTrue(config.getPrintReason());
        Assert.assertTrue(config.hasChain());
        Assert.assertTrue(config.getStoreFunCall());
        Assert.assertTrue(config.getRecallFunCall());
        Assert.assertTrue(config.getRecallDesc());
        Assert.assertEquals(Integer.valueOf(1000), config.getRecallOffset());
        String expected = "{\"additional\":{\"Additional_K\":\"Additional_B\"},\"dynamic\":{\"stopOnFailed\":true},\"containHistories\":false,\"clientHistories\":true,\"clientDowngrade\":true,\"regularProvider\":false,\"funCallHeritage\":false,\"storeCompleted\":true,\"networkBuffer\":10240,\"recallOffset\":1000,\"recallFunCall\":true,\"storeFunCall\":true,\"printReason\":true,\"tokenBuffer\":100,\"tokenFirst\":100,\"recallDesc\":true,\"storeQuery\":true,\"recallNums\":1000,\"histories\":100,\"pureQuery\":true,\"maxError\":1024,\"bridged\":true,\"discard\":true,\"notifier\":\"NOTIFIER\",\"provider\":\"Provider\",\"stream\":true,\"chain\":\"CHAIN\",\"rag\":[{\"stopOnFailed\":false,\"method\":\"POST\",\"mode\":\"xml\",\"sort\":10}]}";
        Assert.assertEquals(expected, JsonUtils.write(config));
    }

    @Test
    public void testIsDynamicPrompt() {
        LLMConfig config = new LLMConfig();
        config.setDynamic(new LLMDynamic());
        Assert.assertTrue(config.hasDynamicPrompt());
    }

    @Test
    public void testHasChain() {
        LLMConfig config = new LLMConfig();
        config.init("Chain", "CUSTOMER_NOTIFIER", "CUSTOMER_TRIGGER", "CUSTOMER_LISTENER");
        Assert.assertEquals("Chain", config.getChain());
        Assert.assertEquals("CUSTOMER_NOTIFIER", config.getNotifier());
        Assert.assertEquals("CUSTOMER_LISTENER", config.getMcpCall().getRewriter());
        Assert.assertEquals("CUSTOMER_TRIGGER", config.getMcpCall().getTrigger());
    }

    @Test
    public void testNotifier1() {
        LLMConfig config = new LLMConfig();
        config.init("Chain", "CUSTOMER_NOTIFIER", "CUSTOMER_TRIGGER", "CUSTOMER_LISTENER");
        Assert.assertEquals("Chain", config.getChain());
        Assert.assertEquals("CUSTOMER_NOTIFIER", config.getNotifier());
        Assert.assertEquals("CUSTOMER_LISTENER", config.getMcpCall().getRewriter());
        Assert.assertEquals("CUSTOMER_TRIGGER", config.getMcpCall().getTrigger());
    }

    @Test
    public void testNotifier2() {
        LLMConfig config = new LLMConfig();
        config.setNotifier("MY_NOTIFIER");
        config.init("Chain", "CUSTOMER_NOTIFIER", "CUSTOMER_TRIGGER", "CUSTOMER_LISTENER");
        Assert.assertEquals("Chain", config.getChain());
        Assert.assertEquals("MY_NOTIFIER", config.getNotifier());
        Assert.assertEquals("CUSTOMER_LISTENER", config.getMcpCall().getRewriter());
        Assert.assertEquals("CUSTOMER_TRIGGER", config.getMcpCall().getTrigger());
    }

    @Test
    public void testGetRecallFunCall() {
        LLMConfig config = new LLMConfig();
        Assert.assertFalse(config.getRecallFunCall());
        config.setStoreFunCall(true);
        Assert.assertTrue(config.getRecallFunCall());
        config.setRecallFunCall(false);
        Assert.assertFalse(config.getRecallFunCall());
        config.setRecallFunCall(true);
        Assert.assertTrue(config.getRecallFunCall());
    }

    @Test
    public void testTimeout() {
        LLMConfig config = new LLMConfig();
        Assert.assertEquals(config.getTimeout(10086), Integer.valueOf(10086));
        config.setTimeout(10087);
        Assert.assertEquals(config.getTimeout(10087), Integer.valueOf(10087));
    }

    @Test
    public void testInitWithStream() {
        LLMConfig config = new LLMConfig();
        config.setStream(true);
        config.init("", "CUSTOMER_NOTIFIER", "CUSTOMER_TRIGGER", null);
        Assert.assertTrue(config.getStream());
        Assert.assertEquals("CUSTOMER_NOTIFIER", config.getNotifier());
        Assert.assertNotNull(config.getMcpCall());
    }

    @Test
    public void testInitWithChainAndStream1() {
        LLMConfig config = new LLMConfig();
        LLMMcpCall mcpCall = new LLMMcpCall();
        mcpCall.setRewriter("MY_LISTENER");
        mcpCall.setTrigger("MY_TRIGGER");
        config.setMcpCall(mcpCall);
        config.setStream(true);
        config.init("Chain", "CUSTOMER_NOTIFIER", "CUSTOMER_TRIGGER", "CUSTOMER_LISTENER");
        Assert.assertTrue(config.getStream());
        Assert.assertEquals("CUSTOMER_NOTIFIER", config.getNotifier());
    }

    @Test
    public void testInitWithChainAndStream2() {
        LLMConfig config = new LLMConfig();
        LLMMcpCall mcpCall = new LLMMcpCall();
        mcpCall.setRewriter("MY_LISTENER");
        mcpCall.setTrigger("MY_TRIGGER");
        config.setMcpCall(mcpCall);
        config.setStream(true);
        config.init("Chain", null, "CUSTOMER_TRIGGER", "CUSTOMER_LISTENER");
        Assert.assertTrue(config.getStream());
        Assert.assertNull(config.getNotifier());
        Assert.assertEquals("MY_LISTENER", config.getMcpCall().getRewriter());
        Assert.assertEquals("MY_TRIGGER", config.getMcpCall().getTrigger());
    }

    @Test
    public void testGetRepoWithValue() {
        LLMConfig config = new LLMConfig();
        List<String> array = new ArrayList<String>();
        array.add("HELLO");
        config.setRepositories(array);
        Assert.assertEquals("HELLO", config.buildRepositories().getFirst());
        Assert.assertEquals("[HELLO, WORLD]", config.buildRepositories("WORLD").toString());
    }

    @Test
    public void testGetRepoWithSame() {
        LLMConfig config = new LLMConfig();
        List<String> array = new ArrayList<String>();
        array.add("HELLO");
        config.setRepositories(array);
        Assert.assertEquals("HELLO", config.buildRepositories().getFirst());
        Assert.assertEquals("[HELLO]", config.buildRepositories("HELLO").toString());
    }


    @Test
    public void testGetRepoWithEmpty() {
        LLMConfig config = new LLMConfig();
        Assert.assertEquals("[WORLD]", config.buildRepositories("WORLD").toString());
    }

    @Test
    public void testInitWithRagConfig() {
        LLMConfig config = new LLMConfig();
        RagConfig ragConfig = new RagConfig();
        config.setRagConfig(Arrays.asList(ragConfig));
        config.setStream(true);
        config.init("Chain", null, "CUSTOMER_TRIGGER", "CUSTOMER_LISTENER");
        Assert.assertEquals("CUSTOMER_LISTENER", ragConfig.getRagMcpConfig().getRewriter());
        Assert.assertEquals("CUSTOMER_TRIGGER", ragConfig.getRagMcpConfig().getTrigger());
    }

    @Test
    public void testInitWithDynamic() {
        LLMConfig config = new LLMConfig();
        LLMDynamic dynamic = new LLMDynamic();
        config.setDynamic(dynamic);
        config.init("Chain", "CUSTOMER_LISTENER", null, null);
        Assert.assertEquals("CUSTOMER_LISTENER", dynamic.getNotifier());
    }

    @Test
    public void testMerge_SourceHasValue_TargetHasNoValue() throws Exception {
        LLMConfig target = new LLMConfig();
        target.setRecallDesc(true);
        target.setRecallOffset(1000);
        List<RagConfig> ragConfigs = Arrays.asList(new RagConfig());
        target.setRagConfig(ragConfigs);
        LLMConfig source = new LLMConfig();
        Assert.assertEquals(LLMConfig.MAX_ERROR, source.getMaxError());
        source.setProvider("OpenAI");
        source.setNotifier("Endpoint");
        source.setPrompt("system-prompt-123");
        source.setChain("workflow-chain-456");
        source.setScene("shared-scene-789");
        source.setMaxError(10);
        Assert.assertEquals(Integer.valueOf(10), source.getMaxError());
        source.setTimeout(30000);
        source.setExpired(86400);
        source.setStream(true);
        source.setContainHistories(true);
        source.setRecallFunCall(true);
        source.setStoreFunCall(true);
        source.setHistories(10);
        source.setPureQuery(false);
        source.setWriteable(false);
        source.setBridged(false);
        source.setTokenBuffer(20);
        source.setTokenFirst(15);
        Map<String, Object> additional = new HashMap<>();
        additional.put("apiKey", "sk-xxxx");
        source.setAdditional(additional);
        Map<String, String> headers = new HashMap<>();
        headers.put("X-Auth", "token-123");
        source.setHeaders(headers);
        List<String> repositories = Arrays.asList("repo1", "repo2");
        source.setRepositories(repositories);
        List<LLMFunCall> funCalls = Arrays.asList(new LLMFunCall());
        source.setFunCalls(funCalls);
        List<RagConfig> ragConfigs1 = Arrays.asList(new RagConfig());
        source.setRagConfig(ragConfigs1);
        LLMDecoration decoration = new LLMDecoration();
        decoration.setPrefix("Prefix: ");
        source.setDecoration(decoration);
        LLMDynamic dynamic = new LLMDynamic();
        dynamic.setStopOnFailed(false);
        source.setDynamic(dynamic);
        LLMMcpCall mcpCall = new LLMMcpCall();
        mcpCall.setTrigger("source-trigger");
        source.setMcpCall(mcpCall);
        LLMConfig result = target.merge(source);
        Assert.assertEquals(Integer.valueOf(10), result.getMaxError());
        Assert.assertEquals("OpenAI", result.getProvider());
        Assert.assertEquals("Endpoint", result.getNotifier());
        // Prompt不覆盖
        Assert.assertNull(result.getPrompt());
        Assert.assertEquals("workflow-chain-456", result.getChain());
        Assert.assertEquals("shared-scene-789", result.getScene());
        Assert.assertEquals(Integer.valueOf(30000), result.getTimeout());
        Assert.assertEquals(Integer.valueOf(86400), result.getExpired());
        Assert.assertTrue(result.getStream());
        Assert.assertTrue(result.getContainHistories());
        Assert.assertEquals(Integer.valueOf(10), result.getHistories());
        Assert.assertFalse(result.getPureQuery());
        Assert.assertFalse(result.getWriteable());
        Assert.assertFalse(result.getBridged());
        Assert.assertEquals(Integer.valueOf(20), result.getTokenBuffer());
        Assert.assertEquals(Integer.valueOf(15), result.getTokenFirst());
        Assert.assertEquals(1, result.getAdditional().size());
        Assert.assertEquals("sk-xxxx", result.getAdditional().get("apiKey"));
        Assert.assertEquals(1, result.getHeaders().size());
        Assert.assertEquals("token-123", result.getHeaders().get("X-Auth"));
        Assert.assertEquals(3, result.buildRepositories().size());
        Assert.assertEquals(2, result.getRepositories().size());
        Assert.assertTrue(result.buildRepositories().containsAll(Arrays.asList("repo1", "repo2")));
        Assert.assertEquals(1, result.getFunCalls().size());
        Assert.assertEquals(2, result.getRagConfig().size());
        Assert.assertNotNull(result.getDecoration());
        Assert.assertEquals("Prefix: ", result.getDecoration().getPrefix());
        Assert.assertNotNull(result.getDynamic());
        Assert.assertFalse(result.getDynamic().getStopOnFailed());
        Assert.assertNotNull(result.getMcpCall());
        Assert.assertEquals("source-trigger", result.getMcpCall().getTrigger());
        Assert.assertTrue(result.getRecallFunCall());
        Assert.assertTrue(result.getRecallDesc());
        Assert.assertEquals(Integer.valueOf(1000), result.getRecallOffset());
    }

    @Test
    public void testMerge_SourceHasNoValue_TargetHasValue() throws Exception {
        LLMConfig target = new LLMConfig();
        target.setProvider("Anthropic");
        target.setNotifier("Localhost");
        target.setPrompt("system-prompt-456");
        target.setChain("workflow-chain-789");
        target.setScene("shared-scene-123");
        target.setTimeout(60000);
        target.setExpired(43200);
        target.setStream(false);
        target.setContainHistories(false);
        target.setHistories(5);
        target.setPureQuery(true);
        target.setWriteable(true);
        target.setBridged(true);
        target.setTokenBuffer(10);
        target.setTokenFirst(5);
        Map<String, Object> additional = new HashMap<>();
        additional.put("apiSecret", "secret-xxxx");
        target.setAdditional(additional);
        Map<String, String> headers = new HashMap<>();
        headers.put("X-Token", "token-456");
        target.setHeaders(headers);
        List<String> repositories = Arrays.asList("repo3", "repo4");
        target.setRepositories(repositories);
        List<LLMFunCall> funCalls = Arrays.asList(new LLMFunCall());
        target.setFunCalls(funCalls);
        List<RagConfig> ragConfigs = Arrays.asList(new RagConfig());
        target.setRagConfig(ragConfigs);
        LLMDecoration decoration = new LLMDecoration();
        decoration.setSuffix(" :Suffix");
        target.setDecoration(decoration);
        LLMDynamic dynamic = new LLMDynamic();
        dynamic.setStopOnFailed(true);
        target.setDynamic(dynamic);
        LLMMcpCall mcpCall = new LLMMcpCall();
        mcpCall.setRewriter("target-rewriter");
        target.setMcpCall(mcpCall);
        target.setRecallFunCall(true);
        LLMConfig source = new LLMConfig();
        LLMConfig result = target.merge(source);
        Assert.assertEquals("Anthropic", result.getProvider());
        Assert.assertEquals("Localhost", result.getNotifier());
        Assert.assertEquals("system-prompt-456", result.getPrompt());
        Assert.assertEquals("workflow-chain-789", result.getChain());
        Assert.assertEquals("shared-scene-123", result.getScene());
        Assert.assertEquals(Integer.valueOf(60000), result.getTimeout());
        Assert.assertEquals(Integer.valueOf(43200), result.getExpired());
        Assert.assertFalse(result.getStream());
        Assert.assertFalse(result.getContainHistories());
        Assert.assertEquals(Integer.valueOf(5), result.getHistories());
        Assert.assertTrue(result.getPureQuery());
        Assert.assertTrue(result.getWriteable());
        Assert.assertTrue(result.getBridged());
        Assert.assertEquals(Integer.valueOf(10), result.getTokenBuffer());
        Assert.assertEquals(Integer.valueOf(5), result.getTokenFirst());
        Assert.assertEquals(1, result.getAdditional().size());
        Assert.assertEquals("secret-xxxx", result.getAdditional().get("apiSecret"));
        Assert.assertEquals(1, result.getHeaders().size());
        Assert.assertEquals("token-456", result.getHeaders().get("X-Token"));
        Assert.assertEquals(3, result.buildRepositories().size());
        Assert.assertEquals(2, result.getRepositories().size());
        Assert.assertTrue(result.buildRepositories().containsAll(Arrays.asList("repo3", "repo4")));
        Assert.assertEquals(1, result.getFunCalls().size());
        Assert.assertEquals(1, result.getRagConfig().size());
        Assert.assertNotNull(result.getDecoration());
        Assert.assertEquals(" :Suffix", result.getDecoration().getSuffix());
        Assert.assertNotNull(result.getDynamic());
        Assert.assertTrue(result.getDynamic().getStopOnFailed());
        Assert.assertNotNull(result.getMcpCall());
        Assert.assertEquals("target-rewriter", result.getMcpCall().getRewriter());
        Assert.assertTrue(result.getRecallFunCall());
    }

    @Test
    public void testMerge_BothHaveValue_TargetPriority() throws Exception {
        LLMConfig target = new LLMConfig();
        target.setProvider("Anthropic");
        target.setNotifier("Localhost");
        target.setTimeout(60000);
        target.setStream(false);
        target.setContainHistories(false);
        target.setRecallFunCall(false);
        Map<String, Object> targetAdditional = new HashMap<>();
        targetAdditional.put("targetKey", "targetValue");
        target.setAdditional(targetAdditional);
        LLMDecoration targetDecoration = new LLMDecoration();
        targetDecoration.setPrefix("TargetPrefix: ");
        target.setDecoration(targetDecoration);
        LLMConfig source = new LLMConfig();
        source.setProvider("OpenAI");
        source.setNotifier("Endpoint");
        source.setTimeout(30000);
        source.setStream(true);
        source.setContainHistories(true);
        source.setRecallFunCall(true);
        source.setPrompt("source-prompt");
        Map<String, Object> sourceAdditional = new HashMap<>();
        sourceAdditional.put("sourceKey", "sourceValue");
        source.setAdditional(sourceAdditional);
        LLMDecoration sourceDecoration = new LLMDecoration();
        sourceDecoration.setSuffix(" :SourceSuffix");
        source.setDecoration(sourceDecoration);
        LLMConfig result = target.merge(source);
        Assert.assertEquals("Anthropic", result.getProvider());
        Assert.assertEquals("Localhost", result.getNotifier());
        Assert.assertEquals(Integer.valueOf(60000), result.getTimeout());
        Assert.assertFalse(result.getStream());
        Assert.assertFalse(result.getContainHistories());
        // Prompt不覆盖
        Assert.assertNull(result.getPrompt());
        Assert.assertEquals(2, result.getAdditional().size());
        Assert.assertEquals("targetValue", result.getAdditional().get("targetKey"));
        Assert.assertNotNull(result.getDecoration());
        Assert.assertEquals("TargetPrefix: ", result.getDecoration().getPrefix());
        Assert.assertEquals(" :SourceSuffix", result.getDecoration().getSuffix());
        Assert.assertFalse(result.getRecallFunCall());
    }

    @Test
    public void testMerge_SourceIsNull_TargetUnchanged() throws Exception {
        LLMConfig target = new LLMConfig();
        target.setProvider("Google");
        target.setNotifier("Source");
        target.setStream(false);
        LLMConfig result = target.merge(null);
        Assert.assertEquals("Google", result.getProvider());
        Assert.assertEquals("Source", result.getNotifier());
        Assert.assertFalse(result.getStream());
    }

    @Test
    public void testMerge_NestedObjects() throws Exception {
        LLMConfig target = new LLMConfig();
        RagConfig targetRag = new RagConfig();
        targetRag.setMode(RagConfig.MODE_XML);
        target.setRagConfig(Arrays.asList(targetRag));
        RagMcpConfig targetRagMcp = new RagMcpConfig();
        targetRagMcp.setTrigger("target-rag-trigger");
        targetRag.setRagMcpConfig(targetRagMcp);
        LLMMcpCall targetMcp = new LLMMcpCall();
        targetMcp.setTrigger("target-mcp-trigger");
        target.setMcpCall(targetMcp);
        LLMConfig source = new LLMConfig();
        RagConfig sourceRag = new RagConfig();
        sourceRag.setMethod("GET");
        RagMcpConfig sourceRagMcp = new RagMcpConfig();
        sourceRagMcp.setRewriter("source-rag-rewriter");
        sourceRag.setRagMcpConfig(sourceRagMcp);
        source.setRagConfig(Arrays.asList(sourceRag));
        LLMMcpCall sourceMcp = new LLMMcpCall();
        sourceMcp.setRewriter("source-mcp-rewriter");
        source.setMcpCall(sourceMcp);
        LLMConfig result = target.merge(source);
        List<RagConfig> resultRags = result.getRagConfig();
        Assert.assertEquals(2, resultRags.size());
        RagConfig resultRag = resultRags.get(0);
        Assert.assertEquals(RagConfig.MODE_XML, resultRag.getMode());
        Assert.assertEquals("POST", resultRag.getMethod());
        Assert.assertEquals("target-rag-trigger", resultRag.getRagMcpConfig().getTrigger());
        Assert.assertNull(resultRag.getRagMcpConfig().getRewriter());
        LLMMcpCall resultMcp = result.getMcpCall();
        Assert.assertEquals("target-mcp-trigger", resultMcp.getTrigger());
        Assert.assertEquals("source-mcp-rewriter", resultMcp.getRewriter());
    }

    @Test
    public void testRag() {
        RagConfig r1 = new RagConfig();
        RagConfig r2 = new RagConfig();
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setRagConfig(List.of(r1, r2));
        llmConfig.init("A");
        Assert.assertEquals(llmConfig, r1.getLlmConfig());
        Assert.assertEquals(llmConfig, r2.getLlmConfig());
    }

    @Test
    public void testRegularProvider() {
        LLMConfig llmConfig = new LLMConfig();
        Assert.assertFalse(llmConfig.getRegularProvider());
        llmConfig.setRegularProvider(true);
        Assert.assertTrue(llmConfig.getRegularProvider());
        llmConfig.setRegularProvider(false);
        llmConfig.replaceProvider("A");
        Assert.assertEquals("A", llmConfig.getProvider());
        llmConfig.replaceProvider("B");
        Assert.assertEquals("B", llmConfig.getProvider());
        llmConfig.setRegularProvider(true);
        llmConfig.replaceProvider("C");
        Assert.assertEquals("B", llmConfig.getProvider());
    }

    @Test
    public void testFunCallAndUpstreamTimeout() {
        LLMConfig llmConfig = new LLMConfig();
        Assert.assertNull(llmConfig.getFunCallTimeout());
        llmConfig.setFunCallTimeout(100);
        llmConfig.setUpstreamTimeout(200);
        Assert.assertEquals(Integer.valueOf(100), llmConfig.getFunCallTimeout());
        Assert.assertEquals(Integer.valueOf(100), llmConfig.getFunCallTimeout(1000));
        llmConfig.setFunCallTimeout(null);
        Assert.assertNull(llmConfig.getFunCallTimeout());
        Assert.assertEquals(Integer.valueOf(999), llmConfig.getFunCallTimeout(999));
        llmConfig.setTimeout(1024);
        Assert.assertEquals(Integer.valueOf(1024), llmConfig.getFunCallTimeout(null));
        Assert.assertEquals(Integer.valueOf(200), llmConfig.getUpstreamTimeout());
        llmConfig.setUpstreamTimeout(null);
        Assert.assertNull(llmConfig.getUpstreamTimeout());
        Assert.assertEquals(Integer.valueOf(999), llmConfig.getUpstreamTimeout(999));
        llmConfig.setTimeout(1025);
        Assert.assertEquals(Integer.valueOf(1025), llmConfig.getUpstreamTimeout(null));
    }

    @Test
    public void testInitTriggerEmpty() {
        LLMConfig config = new LLMConfig();
        config.initTrigger("");
        Assert.assertNull(config.getMcpCall());
    }

    @Test
    public void testInitRewriterEmpty() {
        LLMConfig config = new LLMConfig();
        config.initRewriter(null);
        Assert.assertNull(config.getMcpCall());
    }

    @Test
    public void testReplaceProviderRegular() {
        LLMConfig config = new LLMConfig();
        config.setProvider("P1");
        config.setRegularProvider(true);
        config.replaceProvider("P2");
        Assert.assertEquals("P1", config.getProvider());
    }

    @Test
    public void testGetUpstreamTimeoutNull() {
        LLMConfig config = new LLMConfig();
        config.setTimeout(100);
        Assert.assertEquals(Integer.valueOf(100), config.getUpstreamTimeout(null));
        Assert.assertEquals(Integer.valueOf(50), config.getUpstreamTimeout(50));
    }

    @Test
    public void testGetFunCallTimeoutNull() {
        LLMConfig config = new LLMConfig();
        config.setTimeout(200);
        Assert.assertEquals(Integer.valueOf(200), config.getFunCallTimeout(null));
        Assert.assertEquals(Integer.valueOf(80), config.getFunCallTimeout(80));
    }

    @Test
    public void getDiscard_defaultsToTrueWhenUnset() {
        LLMConfig config = new LLMConfig();
        Assert.assertTrue(config.getDiscard());
    }

    @Test
    public void getDiscard_explicitFalse() {
        LLMConfig config = new LLMConfig();
        config.setDiscard(false);
        Assert.assertFalse(config.getDiscard());
    }

    @Test
    public void getDiscard_explicitTrue() {
        LLMConfig config = new LLMConfig();
        config.setDiscard(true);
        Assert.assertTrue(config.getDiscard());
    }

    @Test
    public void merge_discardTakesSourceWhenTargetUnset() throws Exception {
        LLMConfig target = new LLMConfig();
        LLMConfig source = new LLMConfig();
        source.setDiscard(false);
        target.merge(source);
        Assert.assertFalse(target.getDiscard());
    }

    @Test
    public void merge_discardKeepsTargetWhenTargetSet() throws Exception {
        LLMConfig target = new LLMConfig();
        target.setDiscard(true);
        LLMConfig source = new LLMConfig();
        source.setDiscard(false);
        target.merge(source);
        Assert.assertTrue(target.getDiscard());
    }

    @Test
    public void getClientDowngrade_defaultsToTrueWhenUnset() {
        LLMConfig config = new LLMConfig();
        Assert.assertTrue(config.getClientDowngrade());
    }

    @Test
    public void getClientDowngrade_explicitFalse() {
        LLMConfig config = new LLMConfig();
        config.setClientDowngrade(false);
        Assert.assertFalse(config.getClientDowngrade());
    }

    @Test
    public void getClientDowngrade_explicitTrue() {
        LLMConfig config = new LLMConfig();
        config.setClientDowngrade(true);
        Assert.assertTrue(config.getClientDowngrade());
    }

    @Test
    public void merge_clientDowngradeTakesSourceWhenTargetUnset() throws Exception {
        LLMConfig target = new LLMConfig();
        LLMConfig source = new LLMConfig();
        source.setClientDowngrade(false);
        target.merge(source);
        Assert.assertFalse(target.getClientDowngrade());
    }

    @Test
    public void merge_clientDowngradeKeepsTargetWhenTargetSet() throws Exception {
        LLMConfig target = new LLMConfig();
        target.setClientDowngrade(true);
        LLMConfig source = new LLMConfig();
        source.setClientDowngrade(false);
        target.merge(source);
        Assert.assertTrue(target.getClientDowngrade());
    }

    @Test
    public void hasRecallOffset_falseWhenUnset() {
        LLMConfig config = new LLMConfig();
        Assert.assertFalse(config.hasRecallOffset());
        Assert.assertEquals(Integer.valueOf(0), config.getRecallOffset());
    }

    @Test
    public void hasRecallOffset_trueWhenSetToNonZero() {
        LLMConfig config = new LLMConfig();
        config.setRecallOffset(1000);
        Assert.assertTrue(config.hasRecallOffset());
        Assert.assertEquals(Integer.valueOf(1000), config.getRecallOffset());
    }

    @Test
    public void hasRecallOffset_trueWhenSetToZero() {
        LLMConfig config = new LLMConfig();
        config.setRecallOffset(0);
        Assert.assertTrue(config.hasRecallOffset());
        Assert.assertEquals(Integer.valueOf(0), config.getRecallOffset());
    }
}
