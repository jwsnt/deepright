package ai.open.right.workflow.flow.llm.rag;

import ai.open.right.workflow.flow.config.TimeoutConfig;
import ai.open.right.workflow.flow.llm.rag.meta.RagMetaConfig;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import org.junit.Assert;
import org.junit.Test;

import java.util.List;
import java.util.Map;

public class RagConfigTest {

    @Test
    public void test() {
        TimeoutConfig timeoutConfig = new TimeoutConfig();
        timeoutConfig.setTimeout4Condition(100);
        timeoutConfig.setTimeout4Corrector(200);
        timeoutConfig.setTimeout4Service(300);
        timeoutConfig.setTimeout4Llm(400);
        timeoutConfig.setTimeout(500);
        RagConfig ragConfig = new RagConfig();
        ragConfig.setTimeout(timeoutConfig);
        ragConfig.setReplace("HELLO");
        Assert.assertEquals(Integer.valueOf(100), ragConfig.getTimeout4Condition(1000));
        Assert.assertEquals(Integer.valueOf(300), ragConfig.getTimeout4Service(3000));
        Assert.assertEquals(Integer.valueOf(400), ragConfig.getTimeout4Llm(4000));
        Assert.assertEquals(Integer.valueOf(500), ragConfig.getTimeout(5000));
        Assert.assertTrue(ragConfig.hasReplace());
    }

    @Test
    public void testEnv() {
        RagConfig ragConfig = new RagConfig();
        ragConfig.setEnvironment(List.of("os.name", "COMMAND_MODE"));
        Map<String, String> env = ragConfig.buildEnvironment();
        Assert.assertEquals("Mac OS X", env.get("os.name"));
        Assert.assertEquals("unix2003", env.get("COMMAND_MODE"));
    }
    @Test
    public void testGetTimeoutNull() {
        RagConfig config = new RagConfig();
        config.setTimeout(null);
        Assert.assertEquals(Integer.valueOf(100), config.getTimeout(100));
    }

    @Test
    public void testIsModeCase() {
        RagConfig config = new RagConfig();
        config.setMode(" JSON ");
        Assert.assertTrue(config.isMode("JSON"));
    }

    @Test
    public void testHasRagMetaWhenNull() {
        RagConfig config = new RagConfig();
        config.setRagMetaConfig(null);
        Assert.assertFalse(config.hasRagMeta());
    }

    @Test
    public void testHasRagMetaWhenSet() {
        RagConfig config = new RagConfig();
        config.setRagMetaConfig(new RagMetaConfig());
        Assert.assertTrue(config.hasRagMeta());
    }

    @Test
    public void testInitWithLlmConfig() {
        RagConfig config = new RagConfig();
        LLMConfig llmConfig = new LLMConfig();
        RagConfig result = config.init(llmConfig);
        Assert.assertSame(config, result);
        Assert.assertSame(llmConfig, config.getLlmConfig());
    }

    @Test
    public void testGetRagMetaConfig() {
        RagConfig config = new RagConfig();
        RagMetaConfig metaConfig = new RagMetaConfig();
        config.setRagMetaConfig(metaConfig);
        Assert.assertSame(metaConfig, config.getRagMetaConfig());
    }
}
