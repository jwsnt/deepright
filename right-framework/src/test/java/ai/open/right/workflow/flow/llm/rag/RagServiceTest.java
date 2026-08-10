package ai.open.right.workflow.flow.llm.rag;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import org.junit.Assert;
import org.junit.Test;

public class RagServiceTest {

    @Test
    public void testUpdatePrompt() throws Exception {
        LLMConfig llmConfig = new LLMConfig();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        llmQuery.setQuery("HELLO ");
        RagConfig ragConfig = new RagConfig();
        RagData ragData = RagData.builder()
                .prompt("HELLO #KEY")
                .config(llmConfig)
                .query(llmQuery)
                .build();
        RagService.updatePrompt(ragConfig, ragData, "#KEY", "WORLD");
        Assert.assertEquals("HELLO WORLD", ragData.getPrompt());
        Assert.assertEquals("HELLO ", ragData.getQuery().getQuery());
    }

    @Test
    public void testUpdateQuery() throws Exception {
        LLMConfig llmConfig = new LLMConfig();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        llmQuery.setQuery("HELLO ");
        RagConfig ragConfig = new RagConfig();
        RagData ragData = RagData.builder()
                .prompt("HELLO #KEY")
                .config(llmConfig)
                .query(llmQuery)
                .build();
        RagService.updatePrompt(ragConfig, ragData, "", "WORLD");
        Assert.assertEquals("HELLO #KEY", ragData.getPrompt());
        Assert.assertEquals("HELLO WORLD", ragData.getQuery().getQuery());
    }

    @Test
    public void testUpdateOverride() throws Exception {
        LLMConfig llmConfig = new LLMConfig();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        llmQuery.setQuery("HELLO ");
        RagConfig ragConfig = new RagConfig();
        ragConfig.setOverride(true);
        RagData ragData = RagData.builder()
                .prompt("HELLO #KEY")
                .config(llmConfig)
                .query(llmQuery)
                .build();
        RagService.updatePrompt(ragConfig, ragData, "", "WORLD");
        Assert.assertEquals("HELLO #KEY", ragData.getPrompt());
        Assert.assertEquals("WORLD", ragData.getQuery().getQuery());
    }
}
