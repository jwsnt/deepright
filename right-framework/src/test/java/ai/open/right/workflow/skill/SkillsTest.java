package ai.open.right.workflow.skill;

import com.google.common.collect.ImmutableMap;
import org.junit.Assert;
import org.junit.Test;

public class SkillsTest {

    @Test
    public void copy_returnsNewInstanceWithSameSkillsAndUsage() throws Exception {
        SkillMetadata meta = SkillMetadata.builder().name("N").description("D").build();
        Skills original = Skills.builder()
                .skills(ImmutableMap.of("k", meta))
                .usage("usage-text")
                .build();
        Skills copied = original.copy();
        Assert.assertNotSame("copy() 应返回新实例", original, copied);
        Assert.assertEquals(original.getSkills(), copied.getSkills());
        Assert.assertEquals(original.getUsage(), copied.getUsage());
        Assert.assertEquals("usage-text", copied.getUsage());
        Assert.assertEquals(1, copied.getSkills().size());
        Assert.assertEquals(meta, copied.getSkills().get("k"));
    }
}
