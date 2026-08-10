package ai.open.right.workflow.config.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.config.Prompt;
import ai.open.right.workflow.config.PromptSearch;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.LLMDynamic;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.junit.Assert;
import org.junit.Test;

public class DyPromptServiceTest {

    @Test(expected = WorkflowException.class)
    public void testGetWithException() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        DyPromptService dyPromptService = new DyPromptService();
        dyPromptService.setNotifierService(notifierManager);
        dyPromptService.setTimeout(1000);
        PromptSearch search = new PromptSearch();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMQuery llmQuery = LLMQuery.build(workflowTask);
        LLMConfig llmConfig = new LLMConfig();
        LLMDynamic llmDynamic = new LLMDynamic();
        llmDynamic.setDynamic("NEXT");
        llmConfig.setDynamic(llmDynamic);
        search.setWorkTask(llmQuery);
        search.setLlmConfig(llmConfig);
        Prompt prompt = dyPromptService.get(search);
        Assert.assertTrue(prompt.getContent().length() > 20);
    }

    @Test
    public void testGetWithExceptionAndStopOnFailed() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        DyPromptService dyPromptService = new DyPromptService();
        dyPromptService.setNotifierService(notifierManager);
        dyPromptService.setTimeout(1000);
        PromptSearch search = new PromptSearch();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMQuery llmQuery = LLMQuery.build(workflowTask);
        LLMConfig llmConfig = new LLMConfig();
        LLMDynamic llmDynamic = new LLMDynamic();
        llmDynamic.setStopOnFailed(false);
        llmDynamic.setDynamic("NEXT");
        llmConfig.setDynamic(llmDynamic);
        search.setWorkTask(llmQuery);
        search.setLlmConfig(llmConfig);
        Prompt prompt = dyPromptService.get(search);
        Assert.assertEquals("UNKNOWN", prompt.getContent());
    }

    @Test
    public void testGet() throws Exception {
        DyPromptService dyPromptService = new DyPromptService();
        dyPromptService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("HELLO"));
        dyPromptService.setTimeout(1000);
        PromptSearch search = new PromptSearch();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMQuery llmQuery = LLMQuery.build(workflowTask);
        LLMConfig llmConfig = new LLMConfig();
        LLMDynamic llmDynamic = new LLMDynamic();
        llmDynamic.setStopOnFailed(false);
        llmDynamic.setDynamic("NEXT");
        llmConfig.setDynamic(llmDynamic);
        search.setWorkTask(llmQuery);
        search.setLlmConfig(llmConfig);
        Prompt prompt = dyPromptService.get(search);
        Assert.assertEquals("HELLO", prompt.getContent());
    }
}
