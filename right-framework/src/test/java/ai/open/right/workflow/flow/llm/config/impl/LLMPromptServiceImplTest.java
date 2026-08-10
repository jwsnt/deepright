package ai.open.right.workflow.flow.llm.config.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.config.Prompt;
import ai.open.right.workflow.config.PromptSearch;
import ai.open.right.workflow.config.PromptService;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.LLMDynamic;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.*;

public class LLMPromptServiceImplTest {

    @Test
    public void testSortConfigDescendingBySort() throws Exception {
        LLMPromptServiceImpl service = new LLMPromptServiceImpl();
        LLMConfig llmConfig = new LLMConfig();
        RagConfig low = new RagConfig();
        low.setKey("low");
        low.setSort((byte) 1);
        RagConfig mid = new RagConfig();
        mid.setKey("mid");
        mid.setSort((byte) 2);
        RagConfig high = new RagConfig();
        high.setKey("high");
        high.setSort((byte) 10);
        llmConfig.setRagConfig(Arrays.asList(mid, high, low));
        RagData ragData = RagData.builder().config(llmConfig).build();
        List<RagConfig> sorted = service.sortConfig(llmConfig, ragData);
        Assert.assertEquals(3, sorted.size());
        Assert.assertEquals("high", sorted.get(0).getKey());
        Assert.assertEquals("mid", sorted.get(1).getKey());
        Assert.assertEquals("low", sorted.get(2).getKey());
    }

    @Test
    public void testSortConfigEmpty() throws Exception {
        LLMPromptServiceImpl service = new LLMPromptServiceImpl();
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setRagConfig(Collections.emptyList());
        RagData ragData = RagData.builder().config(llmConfig).build();
        List<RagConfig> sorted = service.sortConfig(llmConfig, ragData);
        Assert.assertTrue(sorted.isEmpty());
    }

    @Test
    public void testSortConfigSingle() throws Exception {
        LLMPromptServiceImpl service = new LLMPromptServiceImpl();
        LLMConfig llmConfig = new LLMConfig();
        RagConfig only = new RagConfig();
        only.setKey("only");
        only.setSort((byte) 7);
        llmConfig.setRagConfig(Collections.singletonList(only));
        RagData ragData = RagData.builder().config(llmConfig).build();
        List<RagConfig> sorted = service.sortConfig(llmConfig, ragData);
        Assert.assertEquals(1, sorted.size());
        Assert.assertSame(only, sorted.get(0));
    }

    /**
     * sort 相同则保持原列表相对顺序（稳定排序）
     */
    @Test
    public void testSortConfigStableWhenSortEqual() throws Exception {
        LLMPromptServiceImpl service = new LLMPromptServiceImpl();
        LLMConfig llmConfig = new LLMConfig();
        RagConfig first = new RagConfig();
        first.setKey("first");
        first.setSort((byte) 5);
        RagConfig second = new RagConfig();
        second.setKey("second");
        second.setSort((byte) 5);
        llmConfig.setRagConfig(Arrays.asList(first, second));
        RagData ragData = RagData.builder().config(llmConfig).build();
        List<RagConfig> sorted = service.sortConfig(llmConfig, ragData);
        Assert.assertEquals("first", sorted.get(0).getKey());
        Assert.assertEquals("second", sorted.get(1).getKey());
    }

    @Test
    public void testRagEach() throws Exception {
        LLMPromptServiceImpl llmPromptService = new LLMPromptServiceImpl();
        RagFuture future = EasyMock.createMock(RagFuture.class);
        future.run();
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(future);
        llmPromptService.ragEach(Arrays.asList(future));
        EasyMock.verify(future);
    }

    @Test
    public void testRagEachWithErrorAndNotStopOnFailed() throws Exception {
        LLMPromptServiceImpl llmPromptService = new LLMPromptServiceImpl();
        RagFuture future = EasyMock.createMock(RagFuture.class);
        future.run();
        EasyMock.expectLastCall().andThrow(new RuntimeException()).anyTimes();
        RagConfig config = new RagConfig();
        config.setStopOnFailed(false);
        EasyMock.expect(future.config()).andReturn(config).anyTimes();
        EasyMock.replay(future);
        llmPromptService.ragEach(Arrays.asList(future));
        EasyMock.verify(future);
    }

    @Test(expected = WorkflowException.class)
    public void testRagEachWithErrorAndStopOnFailed() throws Exception {
        LLMPromptServiceImpl llmPromptService = new LLMPromptServiceImpl();
        RagFuture future = EasyMock.createMock(RagFuture.class);
        future.run();
        EasyMock.expectLastCall().andThrow(new WorkflowException()).anyTimes();
        RagConfig config = new RagConfig();
        config.setStopOnFailed(true);
        EasyMock.expect(future.config()).andReturn(config).anyTimes();
        EasyMock.replay(future);
        llmPromptService.ragEach(Arrays.asList(future));
        EasyMock.verify(future);
    }

    @Test
    public void testRagDoingWithoutRagConfig() throws Exception {
        LLMPromptServiceImpl llmPromptService = new LLMPromptServiceImpl();
        Assert.assertEquals(llmPromptService.ragDoing(new OpenAiRequest(), new LLMConfig(), ObjectBuilder.buildLLMQuery(), "Hello"), "Hello");
    }

    @Test
    public void testRagDoingWithRagGroup() throws Exception {
        RagService ragService = EasyMock.createMock(RagService.class);
        LLMConfig llmConfig = new LLMConfig();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setKey("rag_a");
        llmConfig.setRagConfig(Arrays.asList(ragConfig));
        RagData ragData = RagData.builder()
                .config(llmConfig)
                .prompt("Hello")
                .build();
        RagFuture future = EasyMock.createMock(RagFuture.class);
        EasyMock.expect(ragService.rag(ragConfig, ragData)).andReturn(future).anyTimes();
        EasyMock.replay(ragService, future);
        Map<String, RagService> ragServices = new HashMap<String, RagService>();
        ragServices.put("rag_a", ragService);
        LLMPromptServiceImpl llmPromptService = new LLMPromptServiceImpl();
        llmPromptService.setRagService(ragServices);
        List<RagFuture> futures = new ArrayList<RagFuture>();
        llmPromptService.ragGroup(llmConfig, futures, ragData);
        Assert.assertEquals(futures.get(0), future);
        EasyMock.verify(ragService);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testRagDoingWithRagGroupWithOutService1() throws Exception {
        RagService ragService = EasyMock.createMock(RagService.class);
        LLMConfig llmConfig = new LLMConfig();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setStopOnFailed(true);
        ragConfig.setKey("rag_b");
        llmConfig.setRagConfig(Arrays.asList(ragConfig));
        RagData ragData = RagData.builder()
                .config(llmConfig)
                .prompt("Hello")
                .build();
        RagFuture future = EasyMock.createMock(RagFuture.class);
        EasyMock.expect(ragService.rag(ragConfig, ragData)).andReturn(future).anyTimes();
        EasyMock.replay(ragService, future);
        Map<String, RagService> ragServices = new HashMap<String, RagService>();
        ragServices.put("rag_a", ragService);
        LLMPromptServiceImpl llmPromptService = new LLMPromptServiceImpl();
        llmPromptService.setRagService(ragServices);
        List<RagFuture> futures = new ArrayList<RagFuture>();
        try {
            llmPromptService.ragGroup(llmConfig, futures, ragData);
        } catch (Exception e) {
            EasyMock.verify(ragService, future);
            throw e;
        }
        Assert.fail();
    }

    @Test
    public void testRagDoingWithRagGroupWithOutService2() throws Exception {
        RagService ragService = EasyMock.createMock(RagService.class);
        LLMConfig llmConfig = new LLMConfig();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setStopOnFailed(false);
        ragConfig.setKey("rag_b");
        llmConfig.setRagConfig(Arrays.asList(ragConfig));
        RagData ragData = RagData.builder()
                .config(llmConfig)
                .prompt("Hello")
                .build();
        RagFuture future = EasyMock.createMock(RagFuture.class);
        EasyMock.expect(ragService.rag(ragConfig, ragData)).andReturn(future).anyTimes();
        EasyMock.replay(ragService, future);
        Map<String, RagService> ragServices = new HashMap<String, RagService>();
        ragServices.put("rag_a", ragService);
        LLMPromptServiceImpl llmPromptService = new LLMPromptServiceImpl();
        llmPromptService.setRagService(ragServices);
        List<RagFuture> futures = new ArrayList<RagFuture>();
        llmPromptService.ragGroup(llmConfig, futures, ragData);
        EasyMock.verify(ragService, future);
    }

    @Test
    public void testRagDoing() throws Exception {
        LLMConfig llmConfig = new LLMConfig();
        RagConfig ragConfig = new RagConfig();
        llmConfig.setRagConfig(Arrays.asList(ragConfig));
        LLMPromptServiceImpl llmPromptService = new LLMPromptServiceImpl() {
            protected void ragGroup(LLMConfig llmConfig, List<RagFuture> ragFuture, RagData ragData) throws Exception {

            }

            protected void ragEach(List<RagFuture> ragFuture) throws Exception {

            }
        };
        llmPromptService.ragDoing(new OpenAiRequest(), llmConfig, ObjectBuilder.buildLLMQuery(), "Hello");
    }

    @Test
    public void testPrompt() throws Exception {
        PromptService promptService = EasyMock.createMock(PromptService.class);
        Prompt prompt = EasyMock.createMock(Prompt.class);
        EasyMock.expect(prompt.getContent()).andReturn("Hello").anyTimes();
        EasyMock.expect(promptService.search(EasyMock.anyObject(PromptSearch.class))).andReturn(prompt).anyTimes();
        EasyMock.replay(promptService, prompt);
        LLMConfig llmConfig = new LLMConfig();
        RagConfig ragConfig = new RagConfig();
        llmConfig.setRagConfig(Arrays.asList(ragConfig));
        LLMPromptServiceImpl llmPromptService = new LLMPromptServiceImpl() {
            protected String ragDoing(LLMConfig llmConfig, LLMQuery llmQuery, String prompt) throws Exception {
                return prompt;
            }

            protected void ragGroup(LLMConfig llmConfig, List<RagFuture> ragFuture, RagData ragData) throws Exception {

            }

            protected void ragEach(List<RagFuture> ragFuture) throws Exception {

            }
        };
        llmPromptService.setPromptService(promptService);
        Assert.assertEquals("Hello", llmPromptService.prompt(new OpenAiRequest(), llmConfig, ObjectBuilder.buildLLMQuery()));
        EasyMock.verify(promptService, prompt);
    }

    @Test
    public void testInit() throws Exception {
        PromptService promptService = EasyMock.createMock(PromptService.class);
        EasyMock.replay(promptService);
        Map<String, RagService> ragServiceMap = new HashMap<>();
        LLMPromptServiceImpl.InitConfig service = new LLMPromptServiceImpl.InitConfig();
        service.setPromptService(promptService);
        service.setRagService(ragServiceMap);
        LLMPromptServiceImpl empty = (LLMPromptServiceImpl) service.llmPromptService();
        Assert.assertEquals(promptService, empty.getPromptService());
        Assert.assertEquals(ragServiceMap, empty.getRagService());
        EasyMock.verify(promptService);
    }

    @Test
    public void testPromptWithOutSystemPrompt() throws Exception {
        PromptService promptService = EasyMock.createMock(PromptService.class);
        Prompt prompt = EasyMock.createMock(Prompt.class);
        EasyMock.expect(prompt.getContent()).andReturn("").anyTimes();
        EasyMock.expect(promptService.search(EasyMock.anyObject(PromptSearch.class))).andReturn(prompt).anyTimes();
        EasyMock.replay(promptService, prompt);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setDynamic(new LLMDynamic());
        RagConfig ragConfig = new RagConfig();
        llmConfig.setRagConfig(Arrays.asList(ragConfig));
        LLMPromptServiceImpl llmPromptService = new LLMPromptServiceImpl() {
            protected String ragDoing(LLMConfig llmConfig, LLMQuery llmQuery, String prompt) throws Exception {
                return prompt;
            }

            protected void ragGroup(LLMConfig llmConfig, List<RagFuture> ragFuture, RagData ragData) throws Exception {

            }

            protected void ragEach(List<RagFuture> ragFuture) throws Exception {

            }
        };
        llmPromptService.setPromptService(promptService);
        Assert.assertEquals("", llmPromptService.prompt(new OpenAiRequest(), llmConfig, ObjectBuilder.buildLLMQuery()));
        EasyMock.verify(promptService, prompt);
    }

    @Test(expected = WorkflowException.class)
    public void testPromptWithOutSystemPromptAndException() throws Exception {
        PromptService promptService = EasyMock.createMock(PromptService.class);
        Prompt prompt = EasyMock.createMock(Prompt.class);
        EasyMock.expect(prompt.getContent()).andReturn("").anyTimes();
        EasyMock.expect(promptService.search(EasyMock.anyObject(PromptSearch.class))).andReturn(prompt).anyTimes();
        EasyMock.replay(promptService, prompt);
        LLMConfig llmConfig = new LLMConfig();
        RagConfig ragConfig = new RagConfig();
        llmConfig.setRagConfig(Arrays.asList(ragConfig));
        LLMPromptServiceImpl llmPromptService = new LLMPromptServiceImpl() {
            protected String ragDoing(LLMConfig llmConfig, LLMQuery llmQuery, String prompt) throws Exception {
                return prompt;
            }

            protected void ragGroup(LLMConfig llmConfig, List<RagFuture> ragFuture, RagData ragData) throws Exception {

            }

            protected void ragEach(List<RagFuture> ragFuture) throws Exception {

            }
        };
        llmPromptService.setPromptService(promptService);
        try {
            llmPromptService.prompt(new OpenAiRequest(), llmConfig, ObjectBuilder.buildLLMQuery());
        } finally {
            EasyMock.verify(promptService, prompt);
        }
    }
}
