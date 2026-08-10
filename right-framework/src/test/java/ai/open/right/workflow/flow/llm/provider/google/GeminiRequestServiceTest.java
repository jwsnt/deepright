package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.impl.LLMPromptServiceImpl;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestRewriter;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.provider.ProviderToken;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class GeminiRequestServiceTest {

    @Test
    public void testInit() throws Exception {
        GeminiRequestService geminiRequestService = new GeminiRequestService();
        geminiRequestService.setPolicy("HELLO");
        geminiRequestService.setToken("TOKEN");
        geminiRequestService.init();
        Assertions.assertEquals("TOKEN", geminiRequestService.getToken());
        Assertions.assertEquals("HELLO", geminiRequestService.getSafeSettings().get(0).get("threshold"));
        Assertions.assertEquals("HELLO", geminiRequestService.getSafeSettings().get(1).get("threshold"));
        Assertions.assertEquals("HELLO", geminiRequestService.getSafeSettings().get(2).get("threshold"));
        Assertions.assertEquals("HELLO", geminiRequestService.getSafeSettings().get(3).get("threshold"));
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = GeminiRequestService.class.getConstructor((Class<?>[]) null).newInstance((Object[]) null);
        Assertions.assertEquals(object.hashCode(), object.hashCode());
        Assertions.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = GeminiRequestService.InitConfig.class.getConstructor((Class<?>[]) null).newInstance((Object[]) null);
        Assertions.assertEquals(object.hashCode(), object.hashCode());
        Assertions.assertEquals(object, object);
    }

    @Test
    public void testBuild() throws Exception {
        GeminiRequestService s = new GeminiRequestService();
        Assertions.assertEquals(s.build().getClass(), GoogleRequest.class);
    }

    @Test
    public void testConfig() throws Exception {
        LLMPromptServiceImpl prompt = EasyMock.createMock(LLMPromptServiceImpl.class);
        EasyMock.expect(prompt.prompt(EasyMock.anyObject(ProviderRequest.class), EasyMock.anyObject(), EasyMock.anyObject())).andReturn("Hello").anyTimes();
        EasyMock.replay(prompt);
        GeminiRequestService s = new GeminiRequestService();
        s.setProviderRequestRewriter(new ProviderRequestRewriter.BaseRequestRewriter());
        s.setModel("model");
        s.setToken("token");
        s.setLlmPromptService(prompt);
        s.setProviderToken(new ProviderToken());
        LLMConfig config = new LLMConfig();
        Map<String, Object> add = new HashMap<String, Object>();
        add.put(GeminiRequestService.KEY_RESPONSE_SCHEMA, Collections.singletonMap("Hello", "World"));
        add.put(GeminiRequestService.KEY_TOKEN, "TOKEN_A");
        config.setAdditional(add);
        GoogleRequest req = s.config(config, ObjectBuilder.buildLLMQuery());
        Assertions.assertNotNull(req);
        EasyMock.verify(prompt);
    }

    @Test
    public void testRequest() throws Exception {
        GoogleRequest req = new GoogleRequest();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        req.setMessage(message);
        req.setMimeType("image");
        req.setPrompt("Prompt");
        req.setStream(true);
        req.setHistories(10);
        req.setTokenFirst(100);
        req.setTokenBuffer(100);
        req.setContainHistories(true);
        LLMPromptServiceImpl pservice = EasyMock.createMock(LLMPromptServiceImpl.class);
        EasyMock.expect(pservice.prompt(EasyMock.anyObject(ProviderRequest.class), EasyMock.anyObject(), EasyMock.anyObject())).andReturn("Hello").anyTimes();
        EasyMock.replay(pservice);
        GeminiRequestService pro = new GeminiRequestService();
        pro.setModel("model");
        pro.setToken("token");
        LLMConfig config = new LLMConfig();
        Map<String, Object> add = new HashMap<String, Object>();
        add.put(GeminiRequestService.KEY_MEDIA_RESOLUTION, "low");
        add.put(GeminiRequestService.KEY_MIMETYPE, "image");
        add.put(GeminiRequestService.KEY_FREQUENCY_PENALTY, "2.0");
        add.put(GeminiRequestService.KEY_PRESENCE_PENALTY, "1.0");
        add.put(GeminiRequestService.KEY_MAX_OUTPUT_TOKENS, "10");
        add.put(GeminiRequestService.KEY_TEMPERATURE, "0.3");
        add.put(GeminiRequestService.KEY_TOP_P, "10");
        add.put(GeminiRequestService.KEY_TOP_K, "5");
        add.put(ProviderRequestService.KEY_TOKEN, "AA");
        add.put(GeminiRequestService.KEY_SEED, 100);
        add.put(GeminiRequestService.KEY_RESPONSE_MODALITIES, List.of("HELLO"));
        config.setAdditional(add);
        config.setContainHistories(true);
        config.setTokenBuffer(20);
        config.setTokenFirst(10);
        config.setHistories(20);
        config.setStream(true);
        pro.setProviderToken(new ProviderToken());
        pro.setLlmPromptService(pservice);
        pro.setModel("model");
        pro.request(req, config, message);
        Assertions.assertEquals("low", req.getMediaResolution());
        Assertions.assertEquals(true, req.getStream());
        Assertions.assertEquals(Integer.valueOf(100), req.getSeed());
        Assertions.assertEquals("HELLO", req.getResponseModalities().get(0));
        Assertions.assertEquals(req.getPresencePenalty(), Double.valueOf(1.0));
        Assertions.assertEquals(req.getFrequencyPenalty(), Double.valueOf(2.0));
        Assertions.assertEquals(true, req.getContainHistories());
        Assertions.assertEquals(req.getHistories(), Integer.valueOf(20));
        Assertions.assertEquals(req.getTokenFirst(), Integer.valueOf(10));
        Assertions.assertEquals(req.getTokenBuffer(), Integer.valueOf(20));
        EasyMock.verify(pservice);
    }

    @Test
    public void testInitConfigBeanCreation() throws Exception {
        GeminiRequestService.InitConfig initConfig = new GeminiRequestService.InitConfig();
        initConfig.setPolicy("BLOCK_ONLY_HIGH");
        initConfig.setToken("test-token");
        initConfig.setModel("HELLO_MODEL");
        initConfig.setFunCallTimeout(5000);
        GeminiRequestService service = initConfig.geminiRequestService();
        Assertions.assertEquals("BLOCK_ONLY_HIGH", service.getPolicy());
        Assertions.assertEquals("test-token", service.getToken());
        Assertions.assertEquals("HELLO_MODEL", service.getModel(null));
    }

    @Test
    public void testRequestWithDefaultToken() throws Exception {
        LLMPromptServiceImpl pservice = EasyMock.createMock(LLMPromptServiceImpl.class);
        EasyMock.expect(pservice.prompt(EasyMock.anyObject(ProviderRequest.class), EasyMock.anyObject(), EasyMock.anyObject())).andReturn("Hello").anyTimes();
        EasyMock.replay(pservice);
        GeminiRequestService service = new GeminiRequestService();
        service.setModel("model");
        service.setLlmPromptService(pservice);
        service.setToken("default-token");
        service.setProviderRequestRewriter(new ProviderRequestRewriter.BaseRequestRewriter());
        service.setProviderToken(new ProviderToken());
        GoogleRequest req = new GoogleRequest();
        LLMConfig config = new LLMConfig();
        service.request(req, config, ObjectBuilder.buildLLMQuery());
        Assertions.assertEquals("default-token", req.getToken());
    }

    @Test
    public void testRequestWithCustomToken() throws Exception {
        LLMPromptServiceImpl pservice = EasyMock.createMock(LLMPromptServiceImpl.class);
        EasyMock.expect(pservice.prompt(EasyMock.anyObject(ProviderRequest.class), EasyMock.anyObject(), EasyMock.anyObject())).andReturn("Hello").anyTimes();
        EasyMock.replay(pservice);
        GeminiRequestService service = new GeminiRequestService();
        service.setModel("model");
        service.setLlmPromptService(pservice);
        service.setToken("default-token");
        service.setProviderRequestRewriter(new ProviderRequestRewriter.BaseRequestRewriter());
        service.setProviderToken(new ProviderToken());
        GoogleRequest req = new GoogleRequest();
        LLMConfig config = new LLMConfig();
        config.getAdditional().put(ProviderRequestService.KEY_TOKEN, "custom-token");
        service.request(req, config, ObjectBuilder.buildLLMQuery());
        Assertions.assertEquals("custom-token", req.getToken());
    }
}
