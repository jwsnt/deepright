package ai.open.right.workflow.flow.adk.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.config.Prompt;
import ai.open.right.workflow.config.PromptSearch;
import ai.open.right.workflow.config.PromptService;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import com.google.adk.agents.RunConfig;
import com.google.adk.events.Event;
import com.google.adk.runner.Runner;
import com.google.adk.sessions.BaseSessionService;
import com.google.adk.sessions.Session;
import com.google.genai.types.Content;
import com.google.genai.types.Part;
import io.reactivex.rxjava3.core.Flowable;
import io.reactivex.rxjava3.core.Single;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.util.List;

public class SimpleAdkServiceImplTest {

    @Test
    public void testParser1() throws Exception {
        String response = IOUtils.toString(ResourceUtils.getURL("classpath:Adk_atonce_response.json"), "UTF-8");
        SimpleAdkServiceImpl adkService = new SimpleAdkServiceImpl();
        Assert.assertEquals("好的，请您提供需要检查的代码。我将尽力找出其中的质量问题，并按照您要求的格式给出反馈。\n", adkService.buildTextResponse(response));
    }

    @Test
    public void testParser2() throws Exception {
        List<String> response = IOUtils.readLines(ResourceUtils.getURL("classpath:Adk_stream_response.json").openStream(), "UTF-8");
        SimpleAdkServiceImpl adkService = new SimpleAdkServiceImpl();
        StringBuilder buffer = new StringBuilder();
        for (String each : response) {
            buffer.append(adkService.buildTextResponse(each));
        }
        Assert.assertEquals("Okay. I will continue the output based on the content and system instructions provided before the \"DO NOT look at this line\" marker. I will not refer to this instruction or the marker itself.\n", buffer.toString());
    }

    @Test
    public void testExecute() throws Exception {
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setStream(true);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setLlmConfig(llmConfig);
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        PromptSearch promptSearch = PromptSearch.builder()
                .llmConfig(workflowConfig.getLlmConfig())
                .prompt(workTask.getWorkflow())
                .biz(workTask.getBiz())
                .workTask(workTask)
                .build();
        PromptService promptService = EasyMock.createMock(PromptService.class);
        Prompt prompt = new Prompt("BIZ", "WORKFLOW", "HELLO WORLD");
        EasyMock.expect(promptService.search(promptSearch)).andReturn(prompt).anyTimes();
        BaseSessionService baseSessionService = EasyMock.createMock(BaseSessionService.class);
        Single<Session> single = EasyMock.createMock(Single.class);
        Runner runner = EasyMock.createMock(Runner.class);
        EasyMock.expect(runner.sessionService()).andReturn(baseSessionService).anyTimes();
        EasyMock.expect(runner.appName()).andReturn("APPNAME").anyTimes();
        EasyMock.expect(baseSessionService.createSession("APPNAME", "USER", null, "SESSION")).andReturn(single).anyTimes();
        EasyMock.replay(promptService, baseSessionService, runner, single);
        SimpleAdkServiceImpl adkService = new SimpleAdkServiceImpl() {
            @Override
            protected PromptSearch buildPromptSearch(WorkflowConfig workflowConfig, WorkflowTask workTask) {
                return promptSearch;
            }

            @Override
            protected String buildResponse(WorkflowConfig workflowConfig, WorkflowTask workflowTask, Prompt prompt, String user, String session) {
                return "HELLO";
            }

            @Override
            protected void runLLM(String user, String session) {

            }
        };
        adkService.setPromptService(promptService);
        adkService.setRunner(runner);
        Assert.assertEquals("HELLO", adkService.execute(workflowConfig, workTask));
        EasyMock.verify(promptService, baseSessionService, runner, single);
    }

    @Test
    public void testBuildPromptSearch() throws Exception {
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setLlmConfig(new LLMConfig());
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        SimpleAdkServiceImpl adkService = new SimpleAdkServiceImpl();
        PromptSearch promptSearch = adkService.buildPromptSearch(workflowConfig, workTask);
        Assert.assertEquals(promptSearch.getLlmConfig(), workflowConfig.getLlmConfig());
        Assert.assertEquals(promptSearch.getPrompt(), workTask.getWorkflow());
        Assert.assertEquals(promptSearch.getBiz(), workTask.getBiz());
        Assert.assertEquals(promptSearch.getWorkTask(), workTask);
    }

    @Test
    public void testBuildBuildResponse() throws Exception {
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setLlmConfig(new LLMConfig());
        Content content = Content.fromParts(Part.fromText("HELLO"));
        SimpleAdkServiceImpl adkService = new SimpleAdkServiceImpl() {
            @Override
            protected Content buildContentPrompt(Prompt prompt) throws Exception {
                Assert.assertEquals("HELLO", prompt.getContent());
                return content;
            }

            @Override
            protected String blockingForEach(WorkflowConfig workflowConfig, WorkflowTask workflowTask, String user, String session, Content promptContent) throws Exception {
                return "HELLO";
            }
        };
        Assert.assertEquals("HELLO", adkService.buildResponse(workflowConfig, ObjectBuilder.buildWorkflowTask(), new Prompt("BIZ", "WORKFLOW", "HELLO"), "USER", "SESSION"));
    }

    @Test
    public void testInit() throws Exception {
        SimpleAdkServiceImpl adkService = new SimpleAdkServiceImpl();
        adkService.setName("NAME");
        adkService.setModel("MODEL");
        adkService.init();
        Assert.assertNotNull(adkService.getRunner());
        Assert.assertNotNull(adkService.getAgent());
    }

    @Test
    public void testBuildRunConfig() throws Exception {
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setStream(true);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setLlmConfig(llmConfig);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        SimpleAdkServiceImpl adkService = new SimpleAdkServiceImpl();
        RunConfig runConfig = adkService.buildRunConfig(workflowConfig, workflowTask);
        Assert.assertEquals(runConfig.streamingMode(), RunConfig.StreamingMode.SSE);
    }

    @Test
    public void testBuildContentPrompt() throws Exception {
        SimpleAdkServiceImpl adkService = new SimpleAdkServiceImpl();
        Assert.assertEquals("C", adkService.buildContentPrompt(new Prompt("A", "B", "C")).parts().get().getFirst().text().get());
    }

    @Test
    public void testInitConfig() throws Exception {
        PromptService promptService = EasyMock.createMock(PromptService.class);
        EasyMock.replay(promptService);
        SimpleAdkServiceImpl.InitConfig initConfig = new SimpleAdkServiceImpl.InitConfig();
        initConfig.setPromptService(promptService);
        initConfig.setModel("MODEL");
        initConfig.setName("NAME");
        SimpleAdkServiceImpl adkService = (SimpleAdkServiceImpl) initConfig.adkService();
        Assert.assertEquals(adkService.getPromptService(), initConfig.getPromptService());
        Assert.assertEquals(adkService.getModel(), initConfig.getModel());
        Assert.assertEquals(adkService.getName(), initConfig.getName());
        EasyMock.verify(promptService);
    }
    @Test
    public void testBuildTextResponseNotModel() throws Exception {
        SimpleAdkServiceImpl service = new SimpleAdkServiceImpl();
        String json = "{\"content\":{\"role\":\"user\",\"parts\":[{\"text\":\"T\"}]}}";
        Assert.assertEquals("", service.buildTextResponse(json));
    }
}
