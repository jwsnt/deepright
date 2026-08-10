package ai.open.right.workflow.skill;

import org.junit.Assert;
import org.junit.Test;

public class SkillMetadataTest {

    @Test
    public void test() {
        SkillMetadata skillMetadata = SkillMetadata.builder()
                .description("A")
                .path("B")
                .name("C")
                .build();
        Assert.assertEquals("A", skillMetadata.getDescription());
        Assert.assertEquals("B", skillMetadata.getPath());
        Assert.assertEquals("C", skillMetadata.getName());
    }
}
