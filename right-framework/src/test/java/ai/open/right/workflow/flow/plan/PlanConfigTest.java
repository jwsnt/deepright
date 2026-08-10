package ai.open.right.workflow.flow.plan;

import ai.open.right.workflow.flow.iteration.IterationConfig;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import org.junit.Assert;
import org.junit.Test;

public class PlanConfigTest {

    @Test
    public void testGetSet() {
        PlanConfig planConfig = new PlanConfig();
        PlanNotifier planNotifier = new PlanNotifier();
        planConfig.setNotifier(planNotifier);
        Assert.assertFalse(planConfig.hasPlan());
        Assert.assertFalse(planConfig.hasException());
        Assert.assertFalse(planConfig.hasSummary());
        Assert.assertEquals(planNotifier, planConfig.getNotifier());
        Assert.assertEquals(Integer.valueOf(1000), planConfig.getTimeout4Llm(1000));
        planConfig.setPlan("PLAN");
        planConfig.setSummary("SUMMARY");
        planConfig.setTimeout4Llm(10000);
        planConfig.setException("EXCEPTION");
        Assert.assertEquals("PLAN", planConfig.getPlan());
        Assert.assertEquals(Integer.valueOf(10000), planConfig.getTimeout4Llm());
        Assert.assertEquals(Integer.valueOf(10000), planConfig.getTimeout4Llm(1000));
        Assert.assertEquals("EXCEPTION", planConfig.getException());
        Assert.assertEquals("SUMMARY", planConfig.getSummary());
        Assert.assertTrue(planConfig.hasPlan());
        Assert.assertTrue(planConfig.hasException());
        Assert.assertTrue(planConfig.hasSummary());
    }

    @Test
    public void testInit() {
        PlanConfig planConfig = new PlanConfig();
        planConfig.setIterationConfig(new IterationConfig());
        LLMConfig llmConfig = new LLMConfig();
        Assert.assertEquals("NOTIFIER1", planConfig.init("NOTIFIER1").getNotifier().getException());
        Assert.assertEquals("NOTIFIER1", planConfig.init("NOTIFIER1").getNotifier().getSummary());
        Assert.assertEquals("NOTIFIER1", planConfig.init("NOTIFIER1").getIterationConfig().getNotifier().getRefection());
        Assert.assertEquals("NOTIFIER1", planConfig.init("NOTIFIER1").getIterationConfig().getNotifier().getProcessor());
        Assert.assertEquals("NOTIFIER1", planConfig.init("NOTIFIER1").getNotifier().getPlan());
        Assert.assertEquals("NOTIFIER1", planConfig.init("NOTIFIER2").getNotifier().getException());
        Assert.assertEquals("NOTIFIER1", planConfig.init("NOTIFIER2").getNotifier().getSummary());
        Assert.assertEquals("NOTIFIER1", planConfig.init("NOTIFIER2").getNotifier().getPlan());
        Assert.assertTrue(planConfig.toString().length() > 50);
        planConfig.init(llmConfig);
        Assert.assertEquals(llmConfig, planConfig.getIterationConfig().getLlmConfig());
        Assert.assertEquals(llmConfig, planConfig.getLlmConfig());
    }

    @Test
    public void testMerge() throws Exception {
        PlanConfig base = new PlanConfig();
        PlanConfig override = new PlanConfig();
        base.setTimeout4Llm(100);
        override.setTimeout4Llm(200);
        base.setException("base-exception");
        override.setException("override-exception");
        base.setSummary("base-summary");
        override.setSummary("override-summary");
        base.setPlan("base-plan");
        override.setPlan("override-plan");
        PlanNotifier baseNotifier = new PlanNotifier();
        baseNotifier.setException("base-notifier-exception");
        base.setNotifier(baseNotifier);
        PlanNotifier overrideNotifier = new PlanNotifier();
        overrideNotifier.setSummary("override-notifier-summary");
        override.setNotifier(overrideNotifier);
        PlanConfig merged = base.merge(override);
        Assert.assertEquals(Integer.valueOf(100), merged.getTimeout4Llm());
        Assert.assertEquals("base-exception", merged.getException());
        Assert.assertEquals("base-summary", merged.getSummary());
        Assert.assertEquals("base-plan", merged.getPlan());
        Assert.assertEquals("base-notifier-exception", merged.getNotifier().getException());
        Assert.assertEquals("override-notifier-summary", merged.getNotifier().getSummary());
        PlanConfig base2 = new PlanConfig();
        PlanConfig override2 = new PlanConfig();
        override2.setTimeout4Llm(300);
        override2.setException("override2-exception");
        override2.setSummary("override2-summary");
        override2.setPlan("override2-plan");
        PlanNotifier overrideNotifier2 = new PlanNotifier();
        overrideNotifier2.setException("override2-notifier-exception");
        overrideNotifier2.setSummary("override2-notifier-summary");
        override2.setNotifier(overrideNotifier2);
        PlanConfig merged2 = base2.merge(override2);
        Assert.assertEquals(Integer.valueOf(300), merged2.getTimeout4Llm());
        Assert.assertEquals("override2-exception", merged2.getException());
        Assert.assertEquals("override2-summary", merged2.getSummary());
        Assert.assertEquals("override2-plan", merged2.getPlan());
        Assert.assertEquals("override2-notifier-exception", merged2.getNotifier().getException());
        Assert.assertEquals("override2-notifier-summary", merged2.getNotifier().getSummary());
        PlanConfig base3 = new PlanConfig();
        PlanConfig merged3 = base3.merge(null);
        PlanConfig base4 = new PlanConfig();
        PlanConfig override4 = new PlanConfig();
        base4.setNotifier(new PlanNotifier());
        override4.setNotifier(new PlanNotifier().init("test-notifier"));
        PlanConfig merged4 = base4.merge(override4);
        Assert.assertEquals("test-notifier", merged4.getNotifier().getException());
        Assert.assertEquals("test-notifier", merged4.getNotifier().getSummary());
        Assert.assertEquals("test-notifier", merged4.getNotifier().getPlan());
    }
    @Test
    public void testMergeNull() throws Exception {
        PlanConfig config = new PlanConfig();
        config.setPlan("P");
        Assert.assertEquals("P", config.merge(null).getPlan());
    }

    @Test
    public void testGetContainHistoriesNull() {
        PlanConfig config = new PlanConfig();
        config.setContainHistories(null);
        Assert.assertFalse(config.getContainHistories());
    }
}
