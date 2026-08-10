package ai.open.right.workflow.flow.config;

import ai.open.right.workflow.flow.function.FunctionConfig;
import ai.open.right.workflow.flow.iteration.IterationConfig;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.LLMFunCall;
import ai.open.right.workflow.flow.llm.config.LLMMcpCall;
import ai.open.right.workflow.flow.llm.config.LLMTakeover;
import ai.open.right.workflow.flow.llm.signal.SignalConfig;
import ai.open.right.workflow.flow.mapcombine.Combine;
import ai.open.right.workflow.flow.mapcombine.MapCombineConfig;
import ai.open.right.workflow.flow.mapcombine.Mapping;
import ai.open.right.workflow.flow.parallel.ParallelConfig;
import ai.open.right.workflow.flow.plan.PlanConfig;
import ai.open.right.workflow.flow.pubsub.PubSubConfig;
import ai.open.right.workflow.flow.resource.ResourceConfig;
import ai.open.right.workflow.flow.script.ScriptConfig;
import ai.open.right.workflow.flow.select.ChainSelectConfig;
import ai.open.right.workflow.flow.tools.ToolsConfig;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.Collections;
import java.util.HashMap;
import java.util.Map;

public class WorkflowConfigTest {

    @Test
    public void test() {
        WorkflowConfig config = new WorkflowConfig();
        config.setAssistant("Assistant");
        SignalConfig signals = new SignalConfig();
        config.setSignalConfig(signals);
        LLMConfig llmConfig = EasyMock.createMock(LLMConfig.class);
        EasyMock.replay(llmConfig);
        config.setLlmConfig(llmConfig);
        Assert.assertEquals(config.getAssistant(), "Assistant");
        Assert.assertEquals(config.getSignalConfig(), signals);
        Assert.assertEquals(config.getLlmConfig(), llmConfig);
    }

    @Test
    public void testInit() {
        WorkflowConfig config = new WorkflowConfig();
        config.setAssistant("Assistant");
        config.setChain("NEXT-CHAIN");
        SignalConfig signals = new SignalConfig();
        config.setSignalConfig(signals);
        LLMConfig llmConfig = new LLMConfig();
        config.setLlmConfig(llmConfig);
        config.init();
        Assert.assertEquals(config.getLlmConfig().getChain(), "NEXT-CHAIN");
        Assert.assertEquals(config.getAssistant(), "Assistant");
    }

    @Test
    public void testInit2() {
        WorkflowConfig config = new WorkflowConfig();
        LLMConfig llmConfig = new LLMConfig();
        IterationConfig iterationConfig = new IterationConfig();
        PlanConfig planConfig = new PlanConfig();
        PubSubConfig pubSubConfig = new PubSubConfig();
        planConfig.setIterationConfig(iterationConfig);
        config.setPlanConfig(planConfig);
        config.setLlmConfig(llmConfig);
        config.setIterationConfig(iterationConfig);
        config.setPubSubConfig(pubSubConfig);
        config.init();
        Assert.assertEquals(llmConfig, iterationConfig.getLlmConfig());
        Assert.assertEquals(llmConfig, planConfig.getIterationConfig().getLlmConfig());
    }


    @Test
    public void testToString() {
        WorkflowConfig config = new WorkflowConfig();
        config.setAssistant("Assistant");
        SignalConfig signals = new SignalConfig();
        signals.getConfigs().put("SIGNAL_G", "SIGNAL_T");
        config.setSignalConfig(signals);
        LLMConfig llmConfig = new LLMConfig();
        config.setLlmConfig(llmConfig);
        Assert.assertTrue(config.toString().length() > 100);
    }

    @Test
    public void testHasLLM() {
        WorkflowConfig config = new WorkflowConfig();
        config.setLlmConfig(new LLMConfig());
        Assert.assertTrue(config.hasLlm());
    }

    @Test
    public void testHasSignal() {
        WorkflowConfig config = new WorkflowConfig();
        SignalConfig signalConfig = new SignalConfig();
        Map<String, String> configs = new HashMap<String, String>();
        configs.put("SIGNAL_G", "SIGNAL_T");
        signalConfig.setConfigs(configs);
        config.setSignalConfig(signalConfig);
        Assert.assertTrue(config.hasSignals());
    }

    @Test
    public void testHasTools() {
        WorkflowConfig config = new WorkflowConfig();
        config.setToolsConfig(new ToolsConfig());
        Assert.assertTrue(config.hasTools());
    }

    @Test
    public void testIsChain() {
        WorkflowConfig config = new WorkflowConfig();
        config.setChain("NEXT-CHAIN");
        Assert.assertTrue(config.hasChain());
    }

    @Test
    public void testGlobal() {
        WorkflowConfig config = new WorkflowConfig();
        Assert.assertFalse(config.hasGlobal());
        config.setGlobalConfig(Collections.singletonMap("HELLO", "WORLD"));
        Assert.assertTrue(config.hasGlobal());
    }

    @Test
    public void testInitIteration() {
        WorkflowConfig config = new WorkflowConfig();
        config.setLlmConfig(new LLMConfig());
        config.setNotifier("NOTIFIER");
        ResourceConfig resourceConfig = new ResourceConfig();
        config.setResourceConfig(resourceConfig);
        LLMTakeover llmTakeover = new LLMTakeover();
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setTakeover(llmTakeover);
        config.setLlmFunCall(llmFunCall);
        PubSubConfig pubSubConfig = new PubSubConfig();
        config.setPubSubConfig(pubSubConfig);
        IterationConfig iterationConfig = new IterationConfig();
        config.setIterationConfig(iterationConfig);
        MapCombineConfig mapCombineConfig = new MapCombineConfig();
        mapCombineConfig.setMapping(new Mapping());
        mapCombineConfig.setCombine(new Combine());
        config.setMapCombineConfig(mapCombineConfig);
        ParallelConfig parallelConfig = new ParallelConfig();
        config.setParallelConfig(parallelConfig);
        PlanConfig planConfig = new PlanConfig();
        planConfig.setIterationConfig(new IterationConfig());
        config.setPlanConfig(planConfig);
        ScriptConfig scriptConfig = new ScriptConfig();
        config.setScriptConfig(scriptConfig);
        FunctionConfig functionConfig = new FunctionConfig();
        config.setFunctionConfig(functionConfig);
        config.init();
        Assert.assertTrue(config.hasFunction());
        Assert.assertEquals(resourceConfig, config.getResourceConfig());
        Assert.assertEquals("NOTIFIER", config.getIterationConfig().getNotifier().getRefection());
        Assert.assertEquals("NOTIFIER", config.getIterationConfig().getNotifier().getProcessor());
        Assert.assertEquals("NOTIFIER", config.getMapCombineConfig().getMapping().getNotifier());
        Assert.assertEquals("NOTIFIER", config.getMapCombineConfig().getCombine().getNotifier());
        Assert.assertEquals("NOTIFIER", config.getParallelConfig().getNotifier());
        Assert.assertEquals("NOTIFIER", config.getPlanConfig().getNotifier().getException());
        Assert.assertEquals("NOTIFIER", config.getPlanConfig().getNotifier().getSummary());
        Assert.assertEquals("NOTIFIER", config.getPlanConfig().getNotifier().getPlan());
        Assert.assertEquals("NOTIFIER", config.getPlanConfig().getIterationConfig().getNotifier().getProcessor());
        Assert.assertEquals("NOTIFIER", config.getPlanConfig().getIterationConfig().getNotifier().getRefection());
        Assert.assertEquals("NOTIFIER", config.getScriptConfig().getNotifier());
        Assert.assertEquals("NOTIFIER", config.getPubSubConfig().getNotifier());
        Assert.assertEquals("NOTIFIER", config.getLlmFunCall().getTakeover().getNotifier());
    }

    @Test
    public void testChainSelectConfig1() {
        WorkflowConfig workflowConfig = new WorkflowConfig();
        Assert.assertFalse(workflowConfig.hasSelector());
        ChainSelectConfig chainSelectConfig = new ChainSelectConfig();
        workflowConfig.setChainSelectConfig(chainSelectConfig);
        Assert.assertFalse(workflowConfig.hasSelector());
        chainSelectConfig.setDynamic("Dynamic");
        Assert.assertTrue(workflowConfig.hasSelector());
    }

    @Test
    public void testChainSelectConfig2() {
        WorkflowConfig workflowConfig = new WorkflowConfig();
        Assert.assertFalse(workflowConfig.hasSelector());
        ChainSelectConfig chainSelectConfig = new ChainSelectConfig();
        workflowConfig.setChainSelectConfig(chainSelectConfig);
        Assert.assertFalse(workflowConfig.hasSelector());
        chainSelectConfig.setName("NAME");
        Assert.assertTrue(workflowConfig.hasSelector());
    }

    @Test
    public void testUnBoxed() {
        WorkflowConfig workflowConfig = new WorkflowConfig();
        Assert.assertEquals(WorkflowConfig.UNBOXED, workflowConfig.getUnboxed());
        workflowConfig.setUnboxed("_query_");
        Assert.assertEquals("_query_", workflowConfig.getUnboxed());
    }

    @Test
    public void testUnBoxed2() throws Exception {
        WorkflowConfig workflowConfig1 = new WorkflowConfig();
        Assert.assertEquals(WorkflowConfig.UNBOXED, workflowConfig1.getUnboxed());
        workflowConfig1.setUnboxed("_unbox");
        WorkflowConfig workflowConfig2 = new WorkflowConfig();
        Assert.assertEquals("_unbox", workflowConfig1.getUnboxed());
        workflowConfig2.setUnboxed(null);
        workflowConfig2.merge(workflowConfig1);
        Assert.assertEquals("_unbox", workflowConfig2.getUnboxed());
    }
    @Test
    public void testMergeNull() throws Exception {
        WorkflowConfig config = new WorkflowConfig();
        config.setAssistant("A");
        config.merge(null);
        Assert.assertEquals("A", config.getAssistant());
    }

    @Test
    public void testMergeSubConfigs() throws Exception {
        WorkflowConfig config1 = new WorkflowConfig();
        WorkflowConfig config2 = new WorkflowConfig();
        ResourceConfig resourceConfig = new ResourceConfig();
        resourceConfig.setTimeout(1000);
        config2.setResourceConfig(resourceConfig);
        ChainSelectConfig selector = new ChainSelectConfig();
        selector.setDynamic("D");
        config2.setChainSelectConfig(selector);
        
        config1.merge(config2);
        Assert.assertEquals(Integer.valueOf(1000), config1.getResourceConfig().getTimeout());
        Assert.assertEquals("D", config1.getChainSelectConfig().getDynamic());
    }

    @Test
    public void testInitMcpNull() {
        WorkflowConfig config = new WorkflowConfig();
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setMcpCall(new LLMMcpCall());
        config.setLlmConfig(llmConfig);
        config.setMcpConfig(null);
        config.init();
        Assert.assertNull(config.getLlmConfig().getMcpCall().getRewriter());
    }

    @Test
    public void testGetNotifier() {
        WorkflowConfig config = new WorkflowConfig();
        Assert.assertEquals("DEFAULT", config.getNotifier("DEFAULT"));
        config.setNotifier("CUSTOM");
        Assert.assertEquals("CUSTOM", config.getNotifier("DEFAULT"));
    }

    @Test
    public void testGetAssistantDefault() {
        WorkflowConfig config = new WorkflowConfig();
        Assert.assertEquals(ai.open.right.workflow.flow.assistant.DefaultAssistant.WORKFLOW_NAME, config.getAssistant());
        config.setAssistant("CUSTOM");
        Assert.assertEquals("CUSTOM", config.getAssistant());
    }
}
