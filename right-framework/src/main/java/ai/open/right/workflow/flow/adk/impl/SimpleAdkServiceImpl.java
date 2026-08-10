////////////////////////////////////////////////////////////////
///<dependency>
///     <groupId>com.google.adk</groupId>
///     <artifactId>google-adk</artifactId>
///     <version>0.2.0</version>
///     <scope>compile</scope>
///</dependency>
////////////////////////////////////////////////////////////////
package ai.open.right.workflow.flow.adk.impl;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.config.Prompt;
import ai.open.right.workflow.config.PromptSearch;
import ai.open.right.workflow.config.PromptService;
import ai.open.right.workflow.config.impl.PromptServiceImpl;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.adk.AdkService;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import com.google.adk.agents.LlmAgent;
import com.google.adk.agents.RunConfig;
import com.google.adk.runner.InMemoryRunner;
import com.google.adk.runner.Runner;
import com.google.genai.types.Content;
import com.google.genai.types.Part;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;
import org.springframework.util.StringUtils;

import java.util.List;
import java.util.Map;

@Setter
@Getter
@Slf4j
public class SimpleAdkServiceImpl implements AdkService {

    protected PromptService promptService;

    protected LlmAgent agent;

    protected Runner runner;

    protected String model;

    protected String name;

    @PostConstruct
    public void init() throws Exception {
        this.agent = LlmAgent.builder()
                .model(this.model)
                .tools()
                .name(this.name).build();
        this.runner = new InMemoryRunner(this.agent);
    }

    @Override
    public String execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        PromptSearch promptSearch = this.buildPromptSearch(workflowConfig, workTask);
        Prompt prompt = this.promptService.search(promptSearch);
        String session = this.buildSession(workflowConfig, workTask);
        String user = this.buildUser(workflowConfig, workTask);
        if (log.isInfoEnabled()) {
            log.info("ADK user={},session={}", user, session);
        }
        this.runLLM(user, session);
        return this.buildResponse(workflowConfig, workTask, prompt, user, session);
    }

    protected String blockingForEach(WorkflowConfig workflowConfig, WorkflowTask workTask, String user, String session, Content promptContent) throws Exception {
        // 无法测试
        final StringBuffer response = new StringBuffer();
        this.runner.runAsync(user, session, promptContent, this.buildRunConfig(workflowConfig, workTask))
                .blockingForEach(event -> {
                    response.append(this.buildTextResponse(event.toJson()));
                });
        return response.toString();
    }

    protected String buildResponse(WorkflowConfig workflowConfig, WorkflowTask workTask, Prompt prompt, String user, String session) throws Exception {
        Content promptContent = this.buildContentPrompt(prompt);
        String response = this.blockingForEach(workflowConfig, workTask, user, session, promptContent);
        if (log.isInfoEnabled()) {
            log.info("ADK original response={}", response);
        }
        return response;
    }

    protected PromptSearch buildPromptSearch(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        return PromptSearch.builder()
                .llmConfig(workflowConfig.getLlmConfig())
                .prompt(workTask.getWorkflow())
                .biz(workTask.getBiz())
                .workTask(workTask)
                .build();
    }

    protected RunConfig buildRunConfig(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        return RunConfig.builder()
                // Stream
                .setStreamingMode(workflowConfig.getLlmConfig().getStream() ? RunConfig.StreamingMode.SSE : RunConfig.StreamingMode.NONE)
                .build();
    }

    protected String buildSession(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        return workTask.getConversation() + System.currentTimeMillis();
    }

    protected String buildUser(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        return workTask.getDevice() + System.currentTimeMillis();
    }

    protected Content buildContentPrompt(Prompt prompt) throws Exception {
        return Content.fromParts(Part.fromText(prompt.getContent()));
    }

    protected void runLLM(String user, String session) throws Exception {
        // 无法测试
        this.runner.sessionService().createSession(this.runner.appName(), user, null, session).blockingGet();
    }

    // 构建非流式请求的响应
    protected String buildTextResponse(String source) throws Exception {
        StringBuffer contentBuffer = new StringBuffer();
        Map body = JsonUtils.read(source, Map.class);
        Assert.notEmpty(body, "Body can not be empty");
        Map content = Map.class.cast(body.get("content"));
        Assert.notEmpty(content, "Vertex content can not be empty");
        String role = String.class.cast(content.get("role"));
        if ("model".equalsIgnoreCase(role)) {
            List<Map<String, String>> parts = List.class.cast(content.get("parts"));
            Assert.notEmpty(parts, "Parts can not be empty");
            for (Map<String, String> part : parts) {
                String text = part.get("text");
                if (StringUtils.hasText(text)) {
                    contentBuffer.append(text);
                }
            }
        }
        return contentBuffer.toString();
    }
    @Configuration
    @Setter
    @Getter
    @ConditionalOnProperty(name = "adk.enable", havingValue = "true", matchIfMissing = false)
    public static class InitConfig {

        @Autowired
        @Qualifier(PromptServiceImpl.NAME)
        protected PromptService promptService;

        @Value("${vertex.model:gemini-2.0-flash-001}")
        protected String model;

        @Value("${spring.application.name:}")
        protected String name;

        @Bean
        @ConditionalOnMissingBean(AdkService.class)
        public AdkService adkService() throws Exception {
            Assert.hasText(System.getenv("GOOGLE_APPLICATION_CREDENTIALS"), "System env `GOOGLE_APPLICATION_CREDENTIALS` can not be empty");
            Assert.hasText(System.getenv("GOOGLE_GENAI_USE_VERTEXAI"), "System env `GOOGLE_GENAI_USE_VERTEXAI` can not be empty");
            Assert.hasText(System.getenv("GOOGLE_CLOUD_PROJECT"), "System env `GOOGLE_CLOUD_PROJECT` can not be empty");
            SimpleAdkServiceImpl adkService = new SimpleAdkServiceImpl();
            BeanUtils.copyProperties(this, adkService);
            log.info("SimpleAdkServiceImpl inited: model={},name={}", adkService.getModel(), adkService.getName());
            return adkService;
        }
    }
}
