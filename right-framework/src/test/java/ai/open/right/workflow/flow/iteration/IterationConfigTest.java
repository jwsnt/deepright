package ai.open.right.workflow.flow.iteration;

import ai.open.right.workflow.flow.llm.config.LLMConfig;
import org.junit.Assert;
import org.junit.Test;

public class IterationConfigTest {

    @Test
    public void testSetGet() {
        IterationConfig iterationConfig = new IterationConfig();
        IterationNotifier iterationNotifier = new IterationNotifier();
        Assert.assertFalse(iterationConfig.getContainHistories());
        iterationConfig.setNotifier(iterationNotifier);
        iterationConfig.setFunCallTrack(true);
        iterationConfig.setTimes(10);
        iterationConfig.setTimeout(10);
        iterationConfig.setContainHistories(true);
        Assert.assertTrue(iterationConfig.getContainHistories());
        Assert.assertTrue(iterationConfig.getFunCallTrack());
        Assert.assertEquals(iterationNotifier, iterationConfig.getNotifier());
        Assert.assertEquals(Integer.valueOf(10), iterationConfig.getTimeout());
        Assert.assertEquals(iterationConfig.getTimes(), Integer.valueOf(10));
        LLMConfig llmConfig = new LLMConfig();
        iterationConfig.init(llmConfig);
        Assert.assertEquals(llmConfig, iterationConfig.getLlmConfig());
    }

    @Test
    public void testSetGetProcessor() {
        IterationConfig iterationConfig = new IterationConfig();
        Assert.assertNull(iterationConfig.getProcessor());
        Assert.assertFalse(iterationConfig.hasProcessor());
        iterationConfig.setRefection("Refection");
        Assert.assertNull(iterationConfig.getProcessor());
        Assert.assertFalse(iterationConfig.hasProcessor());
        iterationConfig.setProcessor("Processor");
        Assert.assertEquals("Processor", iterationConfig.getProcessor());
        Assert.assertTrue(iterationConfig.hasProcessor());
    }

    @Test
    public void testSetGetRefection() {
        IterationConfig iterationConfig = new IterationConfig();
        Assert.assertNull(iterationConfig.getRefection());
        Assert.assertFalse(iterationConfig.hasRefection());
        iterationConfig.setProcessor("Processor");
        Assert.assertNull(iterationConfig.getRefection());
        Assert.assertFalse(iterationConfig.hasRefection());
        iterationConfig.setRefection("Refection");
        Assert.assertEquals("Refection", iterationConfig.getRefection());
        Assert.assertTrue(iterationConfig.hasRefection());
    }

    @Test
    public void testGetTimeout() {
        IterationConfig iterationConfig = new IterationConfig();
        Assert.assertEquals(iterationConfig.getTimeout(1000), Integer.valueOf(1000));
        iterationConfig.setTimeout(500);
        Assert.assertEquals(iterationConfig.getTimeout(1000), Integer.valueOf(500));
    }

    @Test
    public void testInit() {
        IterationConfig config = new IterationConfig();
        Assert.assertEquals("NOTIFIER1", config.init("NOTIFIER1").getNotifier().getProcessor());
        Assert.assertEquals("NOTIFIER1", config.init("NOTIFIER1").getNotifier().getRefection());
        Assert.assertEquals("NOTIFIER1", config.init("NOTIFIER2").getNotifier().getProcessor());
        Assert.assertEquals("NOTIFIER1", config.init("NOTIFIER2").getNotifier().getRefection());
    }

    @Test
    public void testInit2() {
        IterationConfig config = new IterationConfig();
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setHistories(100);
        Assert.assertEquals(llmConfig, config.init(llmConfig).getLlmConfig());
    }

    @Test
    public void testMerge() throws Exception {
        IterationConfig source = new IterationConfig();
        IterationConfig target = new IterationConfig();
        IterationNotifier targetNotifier = new IterationNotifier();
        targetNotifier.setProcessor("targetProcessor");
        targetNotifier.setRefection("targetRefection");
        target.setNotifier(targetNotifier);
        target.setFunCallTrack(true);
        target.setRefection("targetRefectionStr");
        target.setCondition("targetCondition");
        target.setProcessor("targetProcessorStr");
        target.setTimeout(200);
        target.setTimes(5);
        target.setContainHistories(true);
        IterationConfig merged1 = source.merge(target);
        Assert.assertTrue(merged1.getContainHistories());
        Assert.assertEquals(targetNotifier.getProcessor(), merged1.getNotifier().getProcessor());
        Assert.assertEquals(targetNotifier.getRefection(), merged1.getNotifier().getRefection());
        Assert.assertTrue(merged1.getFunCallTrack());
        Assert.assertEquals("targetRefectionStr", merged1.getRefection());
        Assert.assertEquals("targetCondition", merged1.getCondition());
        Assert.assertEquals("targetProcessorStr", merged1.getProcessor());
        Assert.assertEquals(Integer.valueOf(200), merged1.getTimeout());
        Assert.assertEquals(Integer.valueOf(5), merged1.getTimes());
        IterationConfig source2 = new IterationConfig();
        IterationNotifier sourceNotifier = new IterationNotifier();
        sourceNotifier.setProcessor("sourceProcessor");
        sourceNotifier.setRefection("sourceRefection");
        source2.setNotifier(sourceNotifier);
        source2.setFunCallTrack(false);
        source2.setRefection("sourceRefectionStr");
        source2.setCondition("sourceCondition");
        source2.setProcessor("sourceProcessorStr");
        source2.setTimeout(100);
        source2.setTimes(3);
        IterationConfig target2 = new IterationConfig();
        IterationNotifier targetNotifier2 = new IterationNotifier();
        targetNotifier2.setProcessor("targetProcessor2");
        targetNotifier2.setRefection("targetRefection2");
        target2.setNotifier(targetNotifier2);
        target2.setFunCallTrack(true);
        target2.setRefection("targetRefectionStr2");
        target2.setCondition("targetCondition2");
        target2.setProcessor("targetProcessorStr2");
        target2.setTimeout(300);
        target2.setTimes(6);
        IterationConfig merged2 = source2.merge(target2);
        Assert.assertEquals(sourceNotifier.getProcessor(), merged2.getNotifier().getProcessor());
        Assert.assertEquals(sourceNotifier.getRefection(), merged2.getNotifier().getRefection());
        Assert.assertFalse(merged2.getFunCallTrack());
        Assert.assertEquals("sourceRefectionStr", merged2.getRefection());
        Assert.assertEquals("sourceCondition", merged2.getCondition());
        Assert.assertEquals("sourceProcessorStr", merged2.getProcessor());
        Assert.assertEquals(Integer.valueOf(100), merged2.getTimeout());
        Assert.assertEquals(Integer.valueOf(3), merged2.getTimes());
        IterationConfig source3 = new IterationConfig();
        source3.setFunCallTrack(true);
        source3.setTimes(10);
        IterationConfig merged3 = source3.merge(null);
        Assert.assertTrue(merged3.getFunCallTrack());
        Assert.assertEquals(Integer.valueOf(10), merged3.getTimes());
        IterationConfig source4 = new IterationConfig();
        source4.setFunCallTrack(null);
        source4.setRefection(null);
        source4.setTimeout(null);
        IterationConfig target4 = new IterationConfig();
        target4.setFunCallTrack(true);
        target4.setRefection("targetRefection4");
        target4.setTimeout(400);
        IterationConfig merged4 = source4.merge(target4);
        Assert.assertTrue(merged4.getFunCallTrack());
        Assert.assertEquals("targetRefection4", merged4.getRefection());
        Assert.assertEquals(Integer.valueOf(400), merged4.getTimeout());
    }
}
