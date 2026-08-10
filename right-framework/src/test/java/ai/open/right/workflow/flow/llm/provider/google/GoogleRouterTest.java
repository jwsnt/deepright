package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.ObjectBuilder;
import ai.open.right.config.HttpClientConfig;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.LLMFunCall;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallData;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallResponse;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import org.apache.http.client.methods.HttpPost;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

public class GoogleRouterTest {

    @Test
    public void testReConfig() throws Exception {
        GoogleRouter router = new GoogleRouter() {
            @Override
            protected GoogleReader reader(GoogleRequest request, LLMConfig llmConfig, LLMCallback llmCallback) {
                return null;
            }

            @Override
            protected String url(GoogleRequest request, LLMConfig llmConfig, String t) {
                return "http://x";
            }
        };
        GoogleRequest req = new GoogleRequest();
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.setToken("TOKEN");
        req.setStream(false);
        HttpClientConfig httpClientConfig = new HttpClientConfig();
        httpClientConfig.setConnect4once(1);
        httpClientConfig.setSocket4once(2);
        httpClientConfig.setSocket4stream(3);
        httpClientConfig.setRouter(4);
        httpClientConfig.setTotal(5);
        httpClientConfig.setRequest4once(6);
        router.setHttpClientConfig(httpClientConfig);
        router.setTimeoutRate(2.0D);
        router.setTimeout(1);
        router.setBuffer(2);
        router.setQueue(3);
        HttpPost post = new HttpPost("http://x");
        router.reConfig(req, new LLMConfig(), post);
        Assertions.assertEquals("TOKEN", post.getFirstHeader("Authorization").getValue());
    }

    @Test
    public void testGoogleMessageConstruction() throws Exception {
        GoogleRequest req = new GoogleRequest();
        req.setPrompt("System Prompt");
        req.setToken("TOKEN");
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.getMessage().setQuery("User Query");

        // Add History
        List<History> histories = new ArrayList<>();
        long currentCreated = req.getMessage().getCreated();
        History h1 = new History();
        h1.setUser();
        h1.setContent("H1");
        h1.setCreated(currentCreated + 100000L);
        histories.add(h1);
        History h2 = new History();
        h2.setAssistant();
        h2.setContent("H2");
        h2.setCreated(currentCreated + 100000L);
        histories.add(h2);
        req.getMessage().addHistories(histories);

        // Add FunCall Data
        ProviderFunCallData funCallData = new ProviderFunCallData();
        ProviderFunCallRequest funReq = new ProviderFunCallRequest() {
            @Override
            public Long getCreated() {
                return currentCreated + 100001L;
            }
        };
        funReq.setName("func1");
        funReq.setArgs(Map.of("arg1", "val1"));
        funReq.setRefer(Map.of("thoughtSignature", "SIG1"));
        funCallData.getRequests().add(funReq);

        ProviderFunCallResponse funRes = new ProviderFunCallResponse() {
            @Override
            public Long getCreated() {
                return currentCreated + 100002L;
            }
        };
        funRes.setName("func1");
        funRes.setResponse("result1");
        funCallData.getResponses().add(funRes);
        req.setFunCallData(funCallData);

        // Add FunCall Config
        LLMFunCall tool = new LLMFunCall();
        tool.setName("func1");
        tool.setDescription("desc1");
        tool.setProperties(Map.of("arg1", Map.of("type", "string")));
        tool.setRequired(List.of("arg1"));
        req.setFunCalls(List.of(tool));

        GoogleRouter.GoogleMessage msg = new GoogleRouter.GoogleMessage(req);

        Assertions.assertEquals("System Prompt", msg.getInstruction().getPart().getText());
        // 增加非User的空Query
        Assertions.assertEquals(5 + 1, msg.getContents().size()); // H1, H2, User Query, funReq, funRes

        Assertions.assertEquals("", msg.getContents().get(0).getParts().get(0).getText());
        Assertions.assertEquals("user", msg.getContents().get(0).getRole());

        // Verify contents
        Assertions.assertEquals("User Query", msg.getContents().get(1).getParts().get(0).getText());
        Assertions.assertEquals("user", msg.getContents().get(1).getRole());

        Assertions.assertEquals("H1", msg.getContents().get(2).getParts().get(0).getText());
        Assertions.assertEquals("user", msg.getContents().get(2).getRole());

        Assertions.assertEquals("H2", msg.getContents().get(3).getParts().get(0).getText());

        Assertions.assertEquals("func1", msg.getContents().get(4).getParts().get(0).getFunctionCall().get("name"));
        Assertions.assertEquals("SIG1", msg.getContents().get(4).getParts().get(0).getThoughtSignature());

        Assertions.assertEquals("func1", msg.getContents().get(5).getParts().get(0).getFunctionResponse().get("name"));

        Assertions.assertEquals(1, msg.getTools().length);
        Assertions.assertEquals("func1", msg.getTools()[0].getFunctions().get(0).get("name"));
    }

    @Test
    public void testGoogleMessageWithMedia() throws Exception {
        GoogleRequest req = new GoogleRequest();
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.getMessage().setQuery("Query with image");
        req.setMimeType("inline:image/png");

        MediaContext media = new MediaContext();
        media.setData("ABC");
        req.setMediaContext(List.of(media));

        GoogleRouter.GoogleMessage msg = new GoogleRouter.GoogleMessage(req);
        Assertions.assertEquals(1, msg.getContents().size());
        Assertions.assertEquals(2, msg.getContents().get(0).getParts().size()); // Text + Image
        Assertions.assertEquals("Query with image", msg.getContents().get(0).getParts().get(0).getText());
        Assertions.assertNotNull(msg.getContents().get(0).getParts().get(1).getInline());
        Assertions.assertEquals("image/png", msg.getContents().get(0).getParts().get(1).getInline().getMimeType());
        Assertions.assertEquals("ABC", msg.getContents().get(0).getParts().get(1).getInline().getData());
    }

    @Test
    public void testGoogleMessageWithFileMedia() throws Exception {
        GoogleRequest req = new GoogleRequest();
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.getMessage().setQuery("Query with file");
        req.setMimeType("image/png"); // Not inline

        MediaContext media = new MediaContext();
        media.setData("gs://bucket/file.png");
        req.setMediaContext(List.of(media));

        GoogleRouter.GoogleMessage msg = new GoogleRouter.GoogleMessage(req);
        Assertions.assertEquals(1, msg.getContents().size());
        Assertions.assertEquals(2, msg.getContents().get(0).getParts().size());
        Assertions.assertNotNull(msg.getContents().get(0).getParts().get(1).getFile());
        Assertions.assertEquals("image/png", msg.getContents().get(0).getParts().get(1).getFile().getMimeType());
        Assertions.assertEquals("gs://bucket/file.png", msg.getContents().get(0).getParts().get(1).getFile().getUri());
    }
}
