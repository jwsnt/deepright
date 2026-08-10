package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.ObjectBuilder;
import ai.open.right.resouce.ResourceService;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.impl.LLMPromptServiceImpl;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestRewriter;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.provider.ProviderToken;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import com.google.common.collect.ImmutableMap;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.springframework.util.StringUtils;

import java.net.URI;
import java.util.Collections;
import java.util.HashMap;
import java.util.Map;

public class VertexRequestServiceTest {

    @Test
    public void token_returnsCurrentToken() throws Exception {
        VertexRequestService s = new VertexRequestService();
        Assert.assertNull(s.token());
        s.setToken("MY_TOKEN");
        Assertions.assertEquals("MY_TOKEN", s.token());
        s.setToken("OTHER");
        Assertions.assertEquals("OTHER", s.token());
    }

    @Test
    public void testBuild() throws Exception {
        VertexRequestService s = new VertexRequestService();
        s.setResourceService(ObjectBuilder.buildResourceService());
        Assert.assertNotNull(s.getResourceService());
        Assertions.assertEquals(s.build().getClass(), GoogleRequest.class);
    }

    @Test
    public void testInitAndRefresh() throws Exception {
        VertexRequestService s = new VertexRequestService() {

            @Override
            public void refresh() throws Exception {
                throw new RuntimeException();
            }
        };
        s.init();
    }

    @Test
    public void testConfig() throws Exception {
        LLMPromptServiceImpl prompt = EasyMock.createMock(LLMPromptServiceImpl.class);
        EasyMock.expect(prompt.prompt(EasyMock.anyObject(ProviderRequest.class), EasyMock.anyObject(), EasyMock.anyObject())).andReturn("Hello").anyTimes();
        EasyMock.replay(prompt);
        VertexRequestService s = new VertexRequestService();
        s.setModel("model");
        s.setProviderToken(new ProviderToken());
        s.setProviderRequestRewriter(new ProviderRequestRewriter.BaseRequestRewriter());
        s.setToken("Token");
        s.setLlmPromptService(prompt);
        LLMConfig config = new LLMConfig();
        GoogleRequest req = s.config(config, ObjectBuilder.buildLLMQuery());
        req.setModel("model");
        Assertions.assertNotNull(req);
        EasyMock.verify(prompt);
    }

    @Test
    public void testRequestWithOutSafe() throws Exception {
        GoogleRequest req = new GoogleRequest();
        req.setModel("model");
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        req.setMessage(message);
        req.setPrompt("Prompt");
        req.setStream(true);
        req.setHistories(10);
        req.setTokenFirst(100);
        req.setTokenBuffer(100);
        req.setContainHistories(true);
        LLMPromptServiceImpl pservice = EasyMock.createMock(LLMPromptServiceImpl.class);
        EasyMock.expect(pservice.prompt(EasyMock.anyObject(ProviderRequest.class), EasyMock.anyObject(), EasyMock.anyObject())).andReturn("Hello").anyTimes();
        EasyMock.replay(pservice);
        VertexRequestService pro = new VertexRequestService();
        pro.setProviderToken(new ProviderToken());
        pro.setToken("test-token");
        LLMConfig config = new LLMConfig();
        Map<String, Object> add = new HashMap<String, Object>();
        add.put(GeminiRequestService.KEY_THINKING_CONFIG, ImmutableMap.of("A", "B"));
        add.put(GeminiRequestService.KEY_RESPONSE_SCHEMA, Collections.singletonMap("Hello", "World"));
        add.put(VertexRequestService.KEY_MAX_OUTPUT_TOKENS, "10");
        add.put(VertexRequestService.KEY_FREQUENCY_PENALTY, "2.0");
        add.put(VertexRequestService.KEY_PRESENCE_PENALTY, "1.0");
        add.put(VertexRequestService.KEY_TEMPERATURE, "0.3");
        add.put(VertexRequestService.KEY_TOP_P, "10");
        add.put(VertexRequestService.KEY_TOP_K, "5");
        add.put(VertexRequestService.KEY_TOOL_CONFIG, Collections.singletonMap("World", "Hello"));
        add.put(ProviderRequestService.KEY_TOKEN, "AA");
        add.put(VertexRequestService.KEY_SEED, 100);
        config.setAdditional(add);
        config.setContainHistories(true);
        config.setTokenBuffer(20);
        config.setTokenFirst(10);
        config.setHistories(20);
        config.setStream(true);
        pro.setLlmPromptService(pservice);
        pro.setModel("model");
        pro.request(req, config, message);
        Assertions.assertEquals("B", req.getThinkingConfig().get("A"));
        Assertions.assertEquals(Integer.valueOf(100), req.getSeed());
        Assertions.assertEquals(req.getStream(), true);
        Assertions.assertEquals(req.getContainHistories(), true);
        Assertions.assertEquals(req.getHistories(), Integer.valueOf(20));
        Assertions.assertEquals(req.getTokenFirst(), Integer.valueOf(10));
        Assertions.assertEquals(req.getTokenBuffer(), Integer.valueOf(20));
        Assertions.assertEquals(req.getPresencePenalty(), Double.valueOf(1.0));
        Assertions.assertEquals(req.getFrequencyPenalty(), Double.valueOf(2.0));
        Assertions.assertEquals("Bearer AA", req.getToken());
        EasyMock.verify(pservice);
    }

    @Test
    public void testInit() throws Exception {
        VertexRequestService vertexRequestService = new VertexRequestService();
        VertexTokenExchange exchange = new VertexTokenExchange() {
            @Override
            public String exchange() throws Exception {
                return "EXCHANGED";
            }
        };
        vertexRequestService.setVertexTokenExchange(exchange);
        vertexRequestService.setPolicy("HELLO");
        vertexRequestService.init();
        Assertions.assertEquals("HELLO", vertexRequestService.getSafeSettings().get(0).get("threshold"));
        Assertions.assertEquals("EXCHANGED", vertexRequestService.getToken());
    }

    @Test
    public void testTokenExchange() throws Exception {
        VertexTokenExchange tokenExchange = new VertexTokenExchange() {
            @Override
            public String exchange() throws Exception {
                return "OK";
            }
        };
        VertexRequestService vertexRequestService = new VertexRequestService();
        vertexRequestService.setVertexTokenExchange(tokenExchange);
        vertexRequestService.refresh();
        Assertions.assertEquals("OK", vertexRequestService.getToken());
    }

    @Test
    public void testRefreshWithTokenUri() throws Exception {
        VertexRequestService service = new VertexRequestService();
        service.setTokenUri(""); // Empty
        service.setVertexTokenExchange(new VertexTokenExchange() {
            @Override
            public String exchange() throws Exception {
                return "EXCHANGE";
            }
        });
        service.refresh();
        Assertions.assertEquals("EXCHANGE", service.getToken());
    }

    @Test
    public void testInitConfig() throws Exception {
        VertexRequestService.InitConfig initConfig = new VertexRequestService.InitConfig();
        initConfig.setPolicy("BLOCK_ONLY_HIGH");
        initConfig.setTokenUri("uri://test");
        initConfig.setSeconds(100);
        initConfig.setModel("HELLO_VERTEX");
        VertexRequestService service = initConfig.vertexRequestService();
        Assertions.assertEquals(Integer.valueOf(100), service.getSeconds());
        Assertions.assertEquals("BLOCK_ONLY_HIGH", service.getPolicy());
        Assertions.assertEquals("uri://test", service.getTokenUri());
        Assertions.assertEquals("HELLO_VERTEX", service.getModel(null));
    }

    @Test
    public void testInitToke() throws Exception {
        String file = System.getenv("PROVIDER_VERTEX_TOKEN_URI");
        if (StringUtils.hasText(file)) {
            VertexRequestService.InitConfig initConfig = new VertexRequestService.InitConfig();
            initConfig.setTokenUri(file);
            initConfig.setSeconds(100);
            ResourceService resourceService = EasyMock.createMock(ResourceService.class);
            EasyMock.expect(resourceService.url(file)).andReturn(URI.create(file).toURL()).anyTimes();
            EasyMock.replay(resourceService);
            VertexRequestService service = initConfig.vertexRequestService();
            service.setResourceService(resourceService);
            service.init();
            Assert.assertNotNull(service.getToken());
            EasyMock.verify(resourceService);
        }
    }
}
