package ai.open.right.workflow.config;

import ai.open.right.workflow.config.impl.PromptServiceImpl;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class PromptManagerTest {

    @Test
    public void testGet() throws Exception {
        PromptService promptService = EasyMock.createMock(PromptService.class);
        PromptSearch promptSearch = new PromptSearch();
        LLMConfig llmConfig = new LLMConfig();
        promptSearch.setLlmConfig(llmConfig);
        Prompt prompt = new Prompt("BIZ", "WORKFLOW", "CONTENT");
        EasyMock.expect(promptService.get(promptSearch)).andReturn(prompt).anyTimes();
        Map<String, PromptService> promptServices = new HashMap<String, PromptService>();
        promptServices.put("CS", promptService);
        EasyMock.replay(promptService);
        PromptServiceImpl promptManager = new PromptServiceImpl();
        promptManager.setPromptService(promptServices);
        promptManager.setInstance("CS");
        Assert.assertEquals(prompt, promptManager.get(promptSearch));
        EasyMock.verify(promptService);
    }

    @Test
    public void testSearch() throws Exception {
        PromptService promptService = EasyMock.createMock(PromptService.class);
        PromptSearch promptSearch = new PromptSearch();
        LLMConfig llmConfig = new LLMConfig();
        promptSearch.setLlmConfig(llmConfig);
        Prompt prompt = new Prompt("BIZ", "WORKFLOW", "CONTENT");
        EasyMock.expect(promptService.search(promptSearch)).andReturn(prompt).anyTimes();
        Map<String, PromptService> promptServices = new HashMap<String, PromptService>();
        promptServices.put("CS", promptService);
        EasyMock.replay(promptService);
        PromptServiceImpl promptManager = new PromptServiceImpl();
        promptManager.setPromptService(promptServices);
        promptManager.setInstance("CS");
        Assert.assertEquals(prompt, promptManager.search(promptSearch));
        EasyMock.verify(promptService);
    }
}
