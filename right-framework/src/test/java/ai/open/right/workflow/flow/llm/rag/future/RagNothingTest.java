package ai.open.right.workflow.flow.llm.rag.future;

import org.junit.Assert;
import org.junit.Test;

public class RagNothingTest {

    @Test
    public void test() throws Exception {
        RagFuture nothing = RagFuture.NOTHING;
        Assert.assertNull(nothing.config());
        nothing.run();
    }
}
