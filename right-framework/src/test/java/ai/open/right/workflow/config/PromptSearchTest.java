package ai.open.right.workflow.config;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import org.junit.Assert;
import org.junit.Test;

public class PromptSearchTest {

    @Test
    public void testGetSet() {
        PromptSearch promptSearch = new PromptSearch();
        LLMConfig llmConfig = new LLMConfig();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        Assert.assertFalse(promptSearch.hasNotifier());
        promptSearch.setNotifier("NOTIFIER");
        promptSearch.setLlmConfig(llmConfig);
        promptSearch.setWorkTask(llmQuery);
        promptSearch.setBiz("BIZ");
        promptSearch.setPrompt("PROMPT");
        Assert.assertTrue(promptSearch.hasNotifier());
        Assert.assertEquals("NOTIFIER", promptSearch.getNotifier());
        Assert.assertEquals("BIZ", promptSearch.getBiz());
        Assert.assertEquals("PROMPT", promptSearch.getPrompt());
        Assert.assertEquals(llmConfig, promptSearch.getLlmConfig());
        Assert.assertEquals(llmQuery, promptSearch.getWorkTask());
        Assert.assertNotNull(promptSearch.toString());
    }

    @Test
    public void testInit() {
        LLMConfig llmConfig = new LLMConfig();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        PromptSearch promptSearch = new PromptSearch(llmQuery, llmConfig, "NOTIFIER", "PROMPT", "BIZ");
        Assert.assertTrue(promptSearch.hasNotifier());
        Assert.assertEquals("NOTIFIER", promptSearch.getNotifier());
        Assert.assertEquals("BIZ", promptSearch.getBiz());
        Assert.assertEquals("PROMPT", promptSearch.getPrompt());
        Assert.assertEquals(llmConfig, promptSearch.getLlmConfig());
        Assert.assertEquals(llmQuery, promptSearch.getWorkTask());
    }

    @Test
    public void testBuild() {
        LLMConfig llmConfig = new LLMConfig();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        PromptSearch promptSearch = PromptSearch.builder()
                .llmConfig(llmConfig).workTask(llmQuery).notifier("NOTIFIER").prompt("PROMPT").biz("BIZ")
                .build();
        Assert.assertTrue(promptSearch.hasNotifier());
        Assert.assertEquals("NOTIFIER", promptSearch.getNotifier());
        Assert.assertEquals("BIZ", promptSearch.getBiz());
        Assert.assertEquals("PROMPT", promptSearch.getPrompt());
        Assert.assertEquals(llmConfig, promptSearch.getLlmConfig());
        Assert.assertEquals(llmQuery, promptSearch.getWorkTask());
    }
}
