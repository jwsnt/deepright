package ai.open.right.workflow.flow.llm.provider.xiaomi;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.provider.ProviderRouter;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiStream;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import ai.open.right.workflow.flow.llm.store.Dimension;
import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.flow.llm.token.TokenStatistic;
import ai.open.right.workflow.flow.llm.token.TokenData;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.media.MediaContext;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.test.util.ReflectionTestUtils;

import java.util.Collections;
import java.util.List;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.*;

class XiaomiQueryServiceTest {

    private XiaomiQueryService xiaomiQueryService;
    private XiaomiRequestService xiaomiRequestService;
    private XiaomiRouter xiaomiRouter;
    private SignalStream signalStream;
    private OpenAiRequest request;

    @BeforeEach
    void setUp() {
        xiaomiRequestService = new XiaomiRequestService();
        xiaomiRouter = new XiaomiRouter();
        signalStream = SignalStream.EMPTY;
        request = new OpenAiRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));

        NotifierService notifierService = new NotifierService() {
            public void notify(String n, Segment s, RedirectContext rc, NotifierWriteBack nwb, List<MediaContext> mc) {}
            public void notify(String n, Segment s, RedirectContext rc, NotifierWriteBack nwb) {}
            public void notify(Segment s, RedirectContext rc, NotifierWriteBack nwb, List<MediaContext> mc) {}
            public void notify(Segment s, RedirectContext rc, NotifierWriteBack nwb) {}
            public void notify(Segment s, NotifierWriteBack nwb, List<MediaContext> mc) {}
            public void notify(Segment s, NotifierWriteBack nwb) {}
        };

        HistoryStore historyStore = new HistoryStore() {
            public void store(Dimension d, List<String> repos, String q, String ans, String reasoning, Integer exp, Integer nums, Long now) {}
            public void store(Dimension d, List<String> repos, String q, String ans, Integer exp, Integer nums, Long now) {}
            public void store(Dimension d, List<String> repos, List<HistoryPair> pairs, Integer exp, Integer nums) {}
            public void store(Dimension d, List<String> repos, HistoryPair pair, Integer exp, Integer nums) {}
            public List<History> restore(Dimension d, String scene, Integer nums, Boolean desc, Long now, Long offset) { return Collections.emptyList(); }
            public List<History> restore(Dimension d, String scene, Integer nums, Boolean desc, Long now) { return Collections.emptyList(); }
            public List<History> restore(Dimension d, String scene, Integer nums, Boolean desc) { return Collections.emptyList(); }
            public List<History> restore(Dimension d, String scene, Integer nums, Long now) { return Collections.emptyList(); }
            public List<History> restore(Dimension d, String scene, Integer nums) { return Collections.emptyList(); }
            public void clear(Dimension d, List<String> repos, Boolean desc, Long now) {}
            public void clear(Dimension d, List<String> repos, Long now) {}
        };

        NamesService namesService = new NamesService() {
            public String encode(String prefix, String client, String name) { return prefix + client + "_" + name; }
            public String[] decode(String name) { return new String[]{"", ""}; }
            public Boolean isPrefixWorkflow(String name) { return false; }
            public Boolean isPrefixResource(String name) { return false; }
            public Boolean isPrefixPrompt(String name) { return false; }
            public Boolean isPrefixTools(String name) { return false; }
            public Boolean isPrefix(String name) { return false; }
        };

        TokenStatistic tokenStatistic = new TokenStatistic() {
            public Set<String> models() { return Collections.emptySet(); }
            public void stat(ProviderRequest pr, TokenData td) {}
            public List<TokenData> readAll(Dimension d, List<String> model) { return Collections.emptyList(); }
            public List<TokenData> readAll(Dimension d) { return Collections.emptyList(); }
            public TokenData read(Dimension d, String model) { return null; }
            public TokenData read(Dimension d) { return null; }
        };

        xiaomiQueryService = new XiaomiQueryService();
        ReflectionTestUtils.setField(xiaomiQueryService, "xiaomiRequestService", xiaomiRequestService);
        ReflectionTestUtils.setField(xiaomiQueryService, "xiaomiRouter", xiaomiRouter);
        ReflectionTestUtils.setField(xiaomiQueryService, "notifierService", notifierService);
        ReflectionTestUtils.setField(xiaomiQueryService, "historyStore", historyStore);
        ReflectionTestUtils.setField(xiaomiQueryService, "namesService", namesService);
        ReflectionTestUtils.setField(xiaomiQueryService, "tokenStatistic", tokenStatistic);
    }

    @Test
    @DisplayName("request: 应返回XiaomiRequestService实例")
    void testRequest_ShouldReturnXiaomiRequestService() {
        ProviderRequestService<OpenAiRequest> result = xiaomiQueryService.request();
        assertNotNull(result);
        assertSame(xiaomiRequestService, result);
    }

    @Test
    @DisplayName("request: 多次调用应返回同一实例")
    void testRequest_MultipleCallsShouldReturnSameInstance() {
        ProviderRequestService<OpenAiRequest> result1 = xiaomiQueryService.request();
        ProviderRequestService<OpenAiRequest> result2 = xiaomiQueryService.request();
        assertSame(result1, result2);
    }

    @Test
    @DisplayName("router: 应返回XiaomiRouter实例")
    void testRouter_ShouldReturnXiaomiRouter() {
        ProviderRouter<OpenAiRequest> result = xiaomiQueryService.router();
        assertNotNull(result);
        assertSame(xiaomiRouter, result);
    }

    @Test
    @DisplayName("router: 多次调用应返回同一实例")
    void testRouter_MultipleCallsShouldReturnSameInstance() {
        ProviderRouter<OpenAiRequest> result1 = xiaomiQueryService.router();
        ProviderRouter<OpenAiRequest> result2 = xiaomiQueryService.router();
        assertSame(result1, result2);
    }

    @Test
    @DisplayName("stream: 应创建XiaomiStream实例")
    void testStream_ShouldCreateXiaomiStream() throws Exception {
        OpenAiStream result = xiaomiQueryService.stream(signalStream, request);
        assertNotNull(result);
        assertTrue(result instanceof XiaomiStream);
    }

    @Test
    @DisplayName("stream: 应正确传递参数到XiaomiStream")
    void testStream_ShouldPassParametersCorrectly() throws Exception {
        OpenAiStream result = xiaomiQueryService.stream(signalStream, request);
        assertNotNull(result);
        assertTrue(result instanceof XiaomiStream);
    }

    @Test
    @DisplayName("xiaomiRequestService getter/setter测试")
    void testXiaomiRequestServiceGetterSetter() {
        XiaomiRequestService newService = new XiaomiRequestService();
        xiaomiQueryService.setXiaomiRequestService(newService);
        assertSame(newService, xiaomiQueryService.getXiaomiRequestService());
    }

    @Test
    @DisplayName("xiaomiRouter getter/setter测试")
    void testXiaomiRouterGetterSetter() {
        XiaomiRouter newRouter = new XiaomiRouter();
        xiaomiQueryService.setXiaomiRouter(newRouter);
        assertSame(newRouter, xiaomiQueryService.getXiaomiRouter());
    }

    @Test
    @DisplayName("InitConfig: xiaomiQueryService方法应创建正确实例")
    void testInitConfig_ShouldCreateXiaomiQueryService() throws Exception {
        XiaomiQueryService.InitConfig initConfig = new XiaomiQueryService.InitConfig();
        XiaomiRequestService reqService = new XiaomiRequestService();
        XiaomiRouter router = new XiaomiRouter();
        ReflectionTestUtils.setField(initConfig, "xiaomiRequestService", reqService);
        ReflectionTestUtils.setField(initConfig, "xiaomiRouter", router);
        XiaomiQueryService result = initConfig.xiaomiQueryService();
        assertNotNull(result);
        assertTrue(result instanceof XiaomiQueryService);
    }

    @Test
    @DisplayName("InitConfig: xiaomiQueryService方法应正确复制属性")
    void testInitConfig_ShouldCopyPropertiesCorrectly() throws Exception {
        XiaomiQueryService.InitConfig initConfig = new XiaomiQueryService.InitConfig();
        XiaomiRequestService reqService = new XiaomiRequestService();
        reqService.setModel("test-model");
        reqService.setToken("test-token");
        XiaomiRouter router = new XiaomiRouter();
        router.setUrl("https://test.api.com");
        ReflectionTestUtils.setField(initConfig, "xiaomiRequestService", reqService);
        ReflectionTestUtils.setField(initConfig, "xiaomiRouter", router);
        XiaomiQueryService result = initConfig.xiaomiQueryService();
        assertNotNull(result);
        assertNotNull(result.getXiaomiRequestService());
        assertNotNull(result.getXiaomiRouter());
        assertEquals("test-model", result.getXiaomiRequestService().getModel());
        assertEquals("test-token", result.getXiaomiRequestService().getToken());
        assertEquals("https://test.api.com", result.getXiaomiRouter().getUrl());
    }

    @Test
    @DisplayName("request: 内部依赖为null时应返回null")
    void testRequest_WhenDependencyNull_ShouldReturnNull() {
        ReflectionTestUtils.setField(xiaomiQueryService, "xiaomiRequestService", null);
        ProviderRequestService<OpenAiRequest> result = xiaomiQueryService.request();
        assertNull(result);
    }

    @Test
    @DisplayName("router: 内部依赖为null时应返回null")
    void testRouter_WhenDependencyNull_ShouldReturnNull() {
        ReflectionTestUtils.setField(xiaomiQueryService, "xiaomiRouter", null);
        ProviderRouter<OpenAiRequest> result = xiaomiQueryService.router();
        assertNull(result);
    }
}
