package ai.open.right.workflow.config;

import org.junit.Assert;
import org.junit.Test;

public class TokenEntryTest {

    @Test
    public void test() {
        TokenEntry tokenEntry = TokenEntry.builder()
                .workflow("WORKFLOW")
                .biz("BIZ")
                .build();
        Assert.assertEquals("WORKFLOW", tokenEntry.getWorkflow());
        Assert.assertEquals("BIZ", tokenEntry.getBiz());
    }
}
