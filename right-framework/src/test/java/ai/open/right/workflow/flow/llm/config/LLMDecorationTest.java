package ai.open.right.workflow.flow.llm.config;

import org.junit.Assert;
import org.junit.Test;

public class LLMDecorationTest {

    @Test
    public void test() {
        LLMDecoration decoration = new LLMDecoration();
        decoration.setPrefix("PREFIX");
        decoration.setSuffix("SUFFIX");
        Assert.assertEquals("PREFIX", decoration.getPrefix());
        Assert.assertEquals("SUFFIX", decoration.getSuffix());
    }

    @Test
    public void testMerge() throws Exception {
        LLMDecoration decoration1 = new LLMDecoration();
        decoration1.setPrefix("P1");
        decoration1.setSuffix("S1");
        LLMDecoration result1 = decoration1.merge(null);
        Assert.assertEquals("P1", result1.getPrefix());
        Assert.assertEquals("S1", result1.getSuffix());
        LLMDecoration decoration2 = new LLMDecoration();
        LLMDecoration other2 = new LLMDecoration();
        other2.setPrefix("P2");
        other2.setSuffix("S2");
        LLMDecoration result2 = decoration2.merge(other2);
        Assert.assertEquals("P2", result2.getPrefix());
        Assert.assertEquals("S2", result2.getSuffix());
        LLMDecoration decoration3 = new LLMDecoration();
        decoration3.setPrefix("P3");
        decoration3.setSuffix("S3");
        LLMDecoration other3 = new LLMDecoration();
        LLMDecoration result3 = decoration3.merge(other3);
        Assert.assertEquals("P3", result3.getPrefix());
        Assert.assertEquals("S3", result3.getSuffix());
        LLMDecoration decoration4 = new LLMDecoration();
        decoration4.setPrefix("P4");
        LLMDecoration other4 = new LLMDecoration();
        other4.setSuffix("S4");
        LLMDecoration result4 = decoration4.merge(other4);
        Assert.assertEquals("P4", result4.getPrefix());
        Assert.assertEquals("S4", result4.getSuffix());
        LLMDecoration decoration5 = new LLMDecoration();
        decoration5.setPrefix("P5");
        decoration5.setSuffix("S5");
        LLMDecoration other5 = new LLMDecoration();
        other5.setPrefix("P5_OTHER");
        other5.setSuffix("S5_OTHER");
        LLMDecoration result5 = decoration5.merge(other5);
        Assert.assertEquals("P5", result5.getPrefix());
        Assert.assertEquals("S5", result5.getSuffix());
    }
}
