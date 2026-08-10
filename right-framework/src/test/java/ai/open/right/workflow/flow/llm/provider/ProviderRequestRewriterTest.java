package ai.open.right.workflow.flow.llm.provider;

import org.junit.Assert;
import org.junit.Test;

public class ProviderRequestRewriterTest {

    @Test
    public void testInit() throws Exception {
        ProviderRequestRewriter.InitConfig initConfig = new ProviderRequestRewriter.InitConfig();
        ProviderRequestRewriter.BaseRequestRewriter baseRequestRewriter = initConfig.requestRewriter();
        Assert.assertNotNull(baseRequestRewriter);
    }
}
