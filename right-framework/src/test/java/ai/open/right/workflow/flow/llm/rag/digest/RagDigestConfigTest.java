package ai.open.right.workflow.flow.llm.rag.digest;

import org.junit.Assert;
import org.junit.Test;

public class RagDigestConfigTest {

    @Test
    public void testScene() {
        RagDigestConfig ragDigestConfig = new RagDigestConfig();
        ragDigestConfig.setScene("HELLO");
        Assert.assertEquals("HELLO", ragDigestConfig.getScene());
    }
}
