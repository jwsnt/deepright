package ai.open.right.workflow.flow.llm.rag.future;

import ai.open.right.workflow.flow.llm.rag.RagConfig;
import org.junit.Assert;
import org.junit.Test;

public class RagAtOnceTest {

    @Test
    public void testGetConfig() {
        RagConfig config = new RagConfig();
        RagAtOnce rag = new RagAtOnce(config);
        Assert.assertEquals(config, rag.config());
    }
}
