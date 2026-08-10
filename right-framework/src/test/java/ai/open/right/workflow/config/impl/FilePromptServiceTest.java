package ai.open.right.workflow.config.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.resouce.PlaceholderResolver;
import ai.open.right.workflow.config.Prompt;
import ai.open.right.workflow.config.PromptSearch;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class FilePromptServiceTest {

    @Test
    public void initTest() {
        PlaceholderResolver placeholderResolver = EasyMock.createMock(PlaceholderResolver.class);
        EasyMock.replay(placeholderResolver);
        FilePromptService.InitConfig initConfig = new FilePromptService.InitConfig();
        initConfig.setResourceService(ObjectBuilder.buildResourceService());
        initConfig.setPath("PATH");
        initConfig.setPlaceholderResolver(placeholderResolver);
        Assert.assertEquals(placeholderResolver, initConfig.getPlaceholderResolver());
        Assert.assertNotNull(initConfig.getResourceService());
        Assert.assertEquals("PATH", initConfig.getPath());
        EasyMock.verify(placeholderResolver);
    }

    @Test
    public void testInit1() throws Exception {
        FilePromptService filePromptService = new FilePromptService();
        filePromptService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        filePromptService.setResourceService(ObjectBuilder.buildResourceService());
        filePromptService.setPath("HelloWorld");
        filePromptService.init();
        Assert.assertNotNull(filePromptService.getResourceService());
        Assert.assertEquals("HelloWorld", filePromptService.getPath());
    }

    @Test
    public void testInit2() throws Exception {
        FilePromptService filePromptService = new FilePromptService();
        filePromptService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        filePromptService.setPath("HelloWorld/");
        filePromptService.init();
        Assert.assertEquals("HelloWorld", filePromptService.getPath());
    }

    @Test
    public void testGet() throws Exception {
        FilePromptService filePromptService = new FilePromptService();
        filePromptService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        filePromptService.setPath("classpath:config");
        filePromptService.setResourceService(ObjectBuilder.buildResourceService());
        PromptSearch search = new PromptSearch();
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQueryWithBiz("prompt");
        LLMQuery llmQuery = LLMQuery.build(workflowTask);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setPrompt("dynamicPrompt");
        search.setWorkTask(llmQuery);
        search.setLlmConfig(llmConfig);
        Prompt prompt = filePromptService.get(search);
        Assert.assertTrue(prompt.getContent().length() > 5);
    }

    @Test
    public void testIOException() throws Exception {
        FilePromptService filePromptService = new FilePromptService();
        filePromptService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        filePromptService.setPath("classpath:config");
        filePromptService.setResourceService(ObjectBuilder.buildResourceService());
        PromptSearch search = new PromptSearch();
        WorkflowTask workflowTask = ObjectBuilder.buildLLMQueryWithBiz("prompt_");
        LLMQuery llmQuery = LLMQuery.build(workflowTask);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setPrompt("dynamicPrompt");
        search.setWorkTask(llmQuery);
        search.setLlmConfig(llmConfig);
        Prompt prompt = filePromptService.get(search);
        Assert.assertEquals(workflowTask.getBiz(), prompt.getBiz());
        Assert.assertEquals("UNKNOWN", prompt.getWorkflow());
        Assert.assertEquals("", prompt.getContent());
    }
}
