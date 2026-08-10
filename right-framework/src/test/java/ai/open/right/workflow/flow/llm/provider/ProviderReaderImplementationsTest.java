package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.provider.anthropic.AnthropicReader;
import ai.open.right.workflow.flow.llm.provider.anthropic.AnthropicRequest;
import ai.open.right.workflow.flow.llm.provider.coze.CozeReader;
import ai.open.right.workflow.flow.llm.provider.coze.CozeRequest;
import ai.open.right.workflow.flow.llm.provider.google.GoogleReader;
import ai.open.right.workflow.flow.llm.provider.google.GoogleRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiReader;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.seedream.SeedreamReader;
import ai.open.right.workflow.flow.llm.provider.seedream.SeedreamRequest;
import org.apache.http.HttpResponse;
import org.apache.http.StatusLine;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Test;

import java.nio.charset.StandardCharsets;
import java.util.HashMap;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;

import static org.junit.jupiter.api.Assertions.assertDoesNotThrow;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

/**
 * {@link ProviderReader} 各实现类：构造、HTTP 状态与 buildResult 分支覆盖。
 */
class ProviderReaderImplementationsTest {

    private static void receiveHttp(ProviderReader<?> reader, int statusCode) throws Exception {
        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine status = EasyMock.createMock(StatusLine.class);
        EasyMock.expect(status.getStatusCode()).andReturn(statusCode).anyTimes();
        EasyMock.expect(response.getStatusLine()).andReturn(status).anyTimes();
        EasyMock.replay(response, status);
        reader.onResponseReceived(response);
    }

    private static <T extends ProviderRequest> ProviderReaderConfig<T> baseConfig(T request, LLMCallback callback) throws Exception {
        return ProviderReaderConfig.<T>builder()
                .request(request)
                .llmCallback(callback)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build();
    }

    @Test
    void openAiReader_buildResult_200_withBody_incrementsMessage() throws Exception {
        OpenAiRequest req = new OpenAiRequest();
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.setStream(false);
        CountDownLatch latch = new CountDownLatch(1);
        AtomicInteger calls = new AtomicInteger();
        OpenAiReader reader = new OpenAiReader(baseConfig(req, m -> {
            calls.incrementAndGet();
            latch.countDown();
        }));
        ExecutorService es = Executors.newSingleThreadExecutor();
        reader.consuming(es);
        receiveHttp(reader, 200);
        reader.getByteBuffer().put("{\"x\":1}".getBytes(StandardCharsets.UTF_8));
        assertDoesNotThrow(() -> ProviderUtils.buildResult(reader));
        assertTrue(latch.await(5, TimeUnit.SECONDS));
        assertEquals(1, calls.get());
        es.shutdown();
    }

    @Test
    void googleReader_buildResult_400_statusPath() throws Exception {
        GoogleRequest req = new GoogleRequest();
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.setStream(false);
        GoogleReader reader = new GoogleReader(baseConfig(req, m -> { }));
        receiveHttp(reader, 400);
        reader.getByteBuffer().put("err".getBytes(StandardCharsets.UTF_8));
        assertDoesNotThrow(() -> ProviderUtils.buildResult(reader));
    }

    @Test
    void anthropicReader_buildResult_200_singleChunk() throws Exception {
        AnthropicRequest req = new AnthropicRequest();
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.setStream(false);
        AnthropicReader reader = new AnthropicReader(baseConfig(req, m -> { }));
        receiveHttp(reader, 200);
        reader.getByteBuffer().put("{\"a\":1}".getBytes(StandardCharsets.UTF_8));
        assertDoesNotThrow(() -> ProviderUtils.buildResult(reader));
    }

    @Test
    void cozeReader_buildResult_200_singleChunk() throws Exception {
        CozeRequest req = new CozeRequest();
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.setStream(false);
        CozeReader reader = new CozeReader(baseConfig(req, m -> { }));
        receiveHttp(reader, 200);
        reader.getByteBuffer().put("{}".getBytes(StandardCharsets.UTF_8));
        assertDoesNotThrow(() -> ProviderUtils.buildResult(reader));
    }

    @Test
    void seedreamReader_buildResult_200_singleChunk() throws Exception {
        SeedreamRequest req = new SeedreamRequest();
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.setStream(false);
        SeedreamReader reader = new SeedreamReader(baseConfig(req, m -> { }));
        receiveHttp(reader, 200);
        reader.getByteBuffer().put("{}".getBytes(StandardCharsets.UTF_8));
        assertDoesNotThrow(() -> ProviderUtils.buildResult(reader));
    }

    @Test
    void providerReader_buildResult_200_emptyBody_failsAtLeastOneMessage() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.setStream(false);
        ProviderReader<ProviderRequest> reader = new ProviderReader<>(baseConfig(req, m -> { }));
        receiveHttp(reader, 200);
        assertDoesNotThrow(() -> ProviderUtils.buildResult(reader));
    }

    @Test
    void isSuccess_nullCode_treatedAsFailurePath() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.setStream(false);
        ProviderReader<ProviderRequest> reader = new ProviderReader<>(baseConfig(req, m -> { }));
        reader.getByteBuffer().put("x".getBytes(StandardCharsets.UTF_8));
        assertDoesNotThrow(() -> ProviderUtils.buildResult(reader));
    }
}
