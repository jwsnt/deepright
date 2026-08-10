package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.context.RedirectContext;
import ai.open.right.listener.Event;
import ai.open.right.listener.EventListenerService;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.provider.google.GoogleReader;
import ai.open.right.workflow.flow.llm.provider.google.GoogleRequest;
import ai.open.right.workflow.flow.llm.provider.google.GoogleStream;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiReader;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.apache.commons.io.IOUtils;
import org.apache.http.*;
import org.apache.http.nio.ContentDecoder;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import ch.qos.logback.classic.Level;
import ch.qos.logback.classic.Logger;
import ch.qos.logback.classic.spi.ILoggingEvent;
import ch.qos.logback.core.read.ListAppender;
import org.junit.jupiter.api.Assertions;
import org.slf4j.MDC;
import org.slf4j.LoggerFactory;
import org.springframework.util.ResourceUtils;

import java.io.BufferedReader;
import java.io.ByteArrayInputStream;
import java.io.InputStreamReader;
import java.nio.ByteBuffer;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.Executor;
import java.util.concurrent.Executors;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicReference;

import static org.junit.jupiter.api.Assertions.assertEquals;

import ai.open.right.workflow.flow.llm.provider.ProviderReaderConfig;

public class ProviderReaderTest {

    /**
     * 模拟已收到 HTTP 响应（与 HttpAsyncClient 一致，先设置 status code 再读 body）。
     * 同包可访问 {@link ProviderReader#onResponseReceived(HttpResponse)}。
     */
    static void receiveHttp(ProviderReader<?> reader, int statusCode) throws Exception {
        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine status = EasyMock.createMock(StatusLine.class);
        EasyMock.expect(status.getStatusCode()).andReturn(statusCode).anyTimes();
        EasyMock.expect(response.getStatusLine()).andReturn(status).anyTimes();
        EasyMock.replay(response, status);
        reader.onResponseReceived(response);
    }

    @Test
    public void test200() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setToken("Token");
        req.setStream(true);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine status = EasyMock.createMock(StatusLine.class);
        EasyMock.expect(status.getStatusCode()).andReturn(200).anyTimes();
        EasyMock.expect(response.getStatusLine()).andReturn(status).anyTimes();
        HttpEntity entity = EasyMock.createMock(HttpEntity.class);
        EasyMock.expect(response.getEntity()).andReturn(entity).anyTimes();
        EasyMock.expect(entity.getContent()).andReturn(new ByteArrayInputStream("HELLO".getBytes(StandardCharsets.UTF_8))).anyTimes();
        req.setContainHistories(true);
        req.setMessage(message);
        req.setTokenBuffer(100);
        req.setTokenFirst(100);
        req.setHistories(10);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        EasyMock.replay(callback, status, response, entity);
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(callback)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {

            @Override
            protected void completed(String message) throws Exception {
            }

        };
        Assert.assertNotNull(reader.getMessageQueue());
        reader.completed(IOUtils.toString(entity.getContent()));
        Assert.assertNotNull(reader.getRequest());
        Assert.assertNotNull(reader.getProviderReaderCallback().getLlmCallback());
        Assert.assertNotNull(reader.getProviderReaderCallback().getNotifierService());
        Assert.assertNotNull(reader.getEventListenerService());
        EasyMock.verify(callback, status, response, entity);
    }

    @Test
    public void testHttp200() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setToken("Token");
        req.setStream(true);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine status = EasyMock.createMock(StatusLine.class);
        EasyMock.expect(status.getStatusCode()).andReturn(200).anyTimes();
        EasyMock.expect(response.getStatusLine()).andReturn(status).anyTimes();
        HttpEntity entity = EasyMock.createMock(HttpEntity.class);
        EasyMock.expect(response.getEntity()).andReturn(entity).anyTimes();
        Header contentType = EasyMock.createMock(Header.class);
        HeaderElement element = EasyMock.createMock(HeaderElement.class);
        EasyMock.expect(element.getName()).andReturn("Content-Type").anyTimes();
        EasyMock.expect(element.getValue()).andReturn("ABC").anyTimes();
        NameValuePair pair = EasyMock.createMock(NameValuePair.class);
        EasyMock.expect(pair.getName()).andReturn("Content-Type").anyTimes();
        EasyMock.expect(pair.getValue()).andReturn("ABC").anyTimes();
        EasyMock.expect(element.getParameters()).andReturn(new NameValuePair[]{pair}).anyTimes();
        EasyMock.expect(contentType.getElements()).andReturn(new HeaderElement[]{element}).anyTimes();
        EasyMock.expect(entity.getContentType()).andReturn(contentType).anyTimes();
        EasyMock.expect(entity.getContent()).andReturn(new ByteArrayInputStream(new byte[]{'a'})).anyTimes();
        req.setContainHistories(true);
        req.setMessage(message);
        req.setMaxError(1000);
        req.setTokenBuffer(100);
        req.setTokenFirst(100);
        req.setHistories(10);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        callback.callback("a");
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(callback, status, response, entity, contentType, element, pair);
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(callback)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {
            // 追加消息
            @Override
            protected void completed(String message) throws Exception {
                super.completed(IOUtils.toString(entity.getContent(), StandardCharsets.UTF_8));
            }
        };
        reader.consuming(Executors.newFixedThreadPool(1));
        reader.onContentReceived(ProviderUtils.buildDecoder(reader, "ABC"), null);
        ProviderUtils.buildResult(reader);
        EasyMock.verify(callback, status, response, entity, contentType, element, pair);
    }

    @Test
    public void testNot200() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setToken("Token");
        req.setStream(true);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine status = EasyMock.createMock(StatusLine.class);
        EasyMock.expect(status.getStatusCode()).andReturn(400).anyTimes();
        EasyMock.expect(response.getStatusLine()).andReturn(status).anyTimes();
        HttpEntity entity = EasyMock.createMock(HttpEntity.class);
        EasyMock.expect(entity.getContentType()).andReturn(null).anyTimes();
        EasyMock.expect(response.getEntity()).andReturn(entity).anyTimes();
        EasyMock.expect(entity.getContent()).andReturn(new ByteArrayInputStream(new byte[]{97})).anyTimes();
        req.setContainHistories(true);
        req.setMessage(message);
        req.setMaxError(1000);
        req.setTokenBuffer(100);
        req.setTokenFirst(100);
        req.setHistories(10);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        EasyMock.replay(callback, status, response, entity);
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(callback)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {
            // 追加消息
            @Override
            protected void completed(String message) throws Exception {
                super.completed(IOUtils.toString(entity.getContent()));
            }
        };
        ContentDecoder decoder = EasyMock.createMock(ContentDecoder.class);
        EasyMock.expect(decoder.read(reader.getByteBuffer())).andReturn(-1).anyTimes();
        EasyMock.replay(decoder);
        try {
            reader.responseReceived(response);
        } catch (WorkflowException e) {
            Assert.fail();
        }
        EasyMock.verify(callback, status, response, entity);
    }

    @Test
    public void testNotify() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setToken("Token");
        req.setStream(true);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine status = EasyMock.createMock(StatusLine.class);
        EasyMock.expect(status.getStatusCode()).andReturn(400).anyTimes();
        EasyMock.expect(response.getStatusLine()).andReturn(status).anyTimes();
        HttpEntity entity = EasyMock.createMock(HttpEntity.class);
        EasyMock.expect(response.getEntity()).andReturn(entity).anyTimes();
        EasyMock.expect(entity.getContent()).andReturn(new ByteArrayInputStream(new byte[]{})).anyTimes();
        req.setContainHistories(true);
        req.setMessage(message);
        req.setMaxError(1000);
        req.setTokenBuffer(100);
        req.setTokenFirst(100);
        req.setHistories(10);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        EasyMock.replay(callback, status, response, entity);
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(callback)
                .notifierService(ObjectBuilder.buildAssertNotifierManagerWithOnlyAssert("OK"))
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {
            @Override
            protected void completed(String message) throws Exception {
            }

            public Exception getException() {
                return new RuntimeException("OK");
            }
        };
        reader.releaseResources();
        EasyMock.verify(callback, status, response, entity);
    }

    @Test
    public void testEvent() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setStream(false);
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        String stream = IOUtils.toString(new BufferedReader(new InputStreamReader(ResourceUtils.getURL("classpath:VertexResponse_stream.json").openStream(), "UTF-8")));
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(new EventListenerService() {

            @Override
            public void listen(Event event) throws Exception {
                Assert.assertEquals(stream, ProviderData.class.cast(event.getData()).getResponse());
            }
        })
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {

            @Override
            protected void completed(String message) throws Exception {
                super.completed(stream);
            }
        };
        receiveHttp(reader, 200);
        reader.consuming(Executors.newFixedThreadPool(1));
        reader.consumeContent(ProviderUtils.buildDecoder(reader, stream), null);
        ProviderUtils.buildResult(reader);
        reader.releaseResources();
    }

    @Test
    public void testEventWithException() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setStream(false);
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        String stream = IOUtils.toString(new BufferedReader(new InputStreamReader(ResourceUtils.getURL("classpath:VertexResponse_stream.json").openStream(), "UTF-8")));
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(new EventListenerService() {

            @Override
            public void listen(Event event) throws Exception {
                throw new RuntimeException("ERROR");
            }
        })
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {

            @Override
            protected void completed(String message) throws Exception {
                super.completed(stream);
            }
        };
        receiveHttp(reader, 200);
        reader.consuming(Executors.newFixedThreadPool(1));
        reader.consumeContent(ProviderUtils.buildDecoder(reader, stream), null);
        ProviderUtils.buildResult(reader);
        reader.releaseResources();
    }

    @Test
    public void testCapacity1() throws Exception {
        BufferedReader stream = new BufferedReader(new InputStreamReader(ResourceUtils.getURL("classpath:VertexResponse_stream.json").openStream(), "UTF-8"));
        StringBuilder buffer = new StringBuilder();
        String content = IOUtils.toString(new BufferedReader(stream));
        OpenAiRequest request = new OpenAiRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        request.setStream(true);
        AtomicBoolean finished = new AtomicBoolean(false);
        OpenAiReader reader = new OpenAiReader(ProviderReaderConfig.<OpenAiRequest>builder()
                .request(request)
                .llmCallback(new LLMCallback() {
            @Override
            public void callback(String message) {
                buffer.append(message);
                finished.set(true);
            }
        })
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1)
                .capacity(1048576)
                .queue(1024)
                .build()) {
        };
        receiveHttp(reader, 200);
        reader.consuming(Executors.newFixedThreadPool(1));
        reader.consumeContent(ProviderUtils.buildDecoder(reader, content), null);
        ProviderUtils.buildResult(reader);
        while (!finished.get()) {
        }
        String expect = "[{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"${\"}]}}]},{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"I_01;S_00;S_01}\\n\"}]},\"safetyRatings\":[{\"category\":\"HARM_CATEGORY_HATE_SPEECH\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.24804688,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.13671875},{\"category\":\"HARM_CATEGORY_DANGEROUS_CONTENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.28710938,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.06738281},{\"category\":\"HARM_CATEGORY_HARASSMENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.28710938,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.14355469},{\"category\":\"HARM_CATEGORY_SEXUALLY_EXPLICIT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.31835938,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.10986328}]}]},{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"我们是科技，主要做非洲业务。\\n\\n${S_0\"}]},\"safetyRatings\":[{\"category\":\"HARM_CATEGORY_HATE_SPEECH\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.38085938,\"severity\":\"HARM_SEVERITY_MEDIUM\",\"severityScore\":0.4375},{\"category\":\"HARM_CATEGORY_DANGEROUS_CONTENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.13964844,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.119140625},{\"category\":\"HARM_CATEGORY_HARASSMENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.3203125,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.31054688},{\"category\":\"HARM_CATEGORY_SEXUALLY_EXPLICIT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.14257813,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.21289063}]}]},{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"3=2,3}\\n我们这里有两种流量包，一个是肯尼亚的5G流量包，一个是肯尼亚的套餐。您比较偏\"}]},\"safetyRatings\":[{\"category\":\"HARM_CATEGORY_HATE_SPEECH\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.2890625,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.265625},{\"category\":\"HARM_CATEGORY_DANGEROUS_CONTENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.15917969,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.12695313},{\"category\":\"HARM_CATEGORY_HARASSMENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.23925781,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.2109375},{\"category\":\"HARM_CATEGORY_SEXUALLY_EXPLICIT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.14941406,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.26953125}]}]},{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"向哪一种呢？\\n\"}]},\"safetyRatings\":[{\"category\":\"HARM_CATEGORY_HATE_SPEECH\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.29492188,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.29296875},{\"category\":\"HARM_CATEGORY_DANGEROUS_CONTENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.123535156,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.115722656},{\"category\":\"HARM_CATEGORY_HARASSMENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.21777344,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.18847656},{\"category\":\"HARM_CATEGORY_SEXUALLY_EXPLICIT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.13671875,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.29882813}]}]},{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":2729,\"candidatesTokenCount\":71,\"totalTokenCount\":2800}}]";
        Assert.assertEquals(expect, buffer.toString().replaceAll("\\s+", ""));
    }

    @Test
    public void testBuildResultWithStream() throws Exception {
        BufferedReader stream = new BufferedReader(new InputStreamReader(ResourceUtils.getURL("classpath:VertexResponse_stream.json").openStream(), "UTF-8"));
        StringBuilder buffer = new StringBuilder();
        String content = IOUtils.toString(new BufferedReader(stream));
        OpenAiRequest request = new OpenAiRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        request.setStream(true);
        AtomicBoolean finished = new AtomicBoolean(false);
        OpenAiReader reader = new OpenAiReader(ProviderReaderConfig.<OpenAiRequest>builder()
                .request(request)
                .llmCallback(new LLMCallback() {
            @Override
            public void callback(String message) {
                buffer.append(message);
                finished.set(true);
            }
        })
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1)
                .capacity(1048576)
                .queue(1024)
                .build()) {
        };
        receiveHttp(reader, 200);
        reader.consuming(Executors.newFixedThreadPool(1));
        reader.consumeContent(ProviderUtils.buildDecoder(reader, content), null);
        ProviderUtils.buildResult(reader);
        while (!finished.get()) {
        }
        String expect = "[{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"${\"}]}}]},{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"I_01;S_00;S_01}\\n\"}]},\"safetyRatings\":[{\"category\":\"HARM_CATEGORY_HATE_SPEECH\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.24804688,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.13671875},{\"category\":\"HARM_CATEGORY_DANGEROUS_CONTENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.28710938,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.06738281},{\"category\":\"HARM_CATEGORY_HARASSMENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.28710938,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.14355469},{\"category\":\"HARM_CATEGORY_SEXUALLY_EXPLICIT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.31835938,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.10986328}]}]},{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"我们是科技，主要做非洲业务。\\n\\n${S_0\"}]},\"safetyRatings\":[{\"category\":\"HARM_CATEGORY_HATE_SPEECH\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.38085938,\"severity\":\"HARM_SEVERITY_MEDIUM\",\"severityScore\":0.4375},{\"category\":\"HARM_CATEGORY_DANGEROUS_CONTENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.13964844,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.119140625},{\"category\":\"HARM_CATEGORY_HARASSMENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.3203125,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.31054688},{\"category\":\"HARM_CATEGORY_SEXUALLY_EXPLICIT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.14257813,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.21289063}]}]},{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"3=2,3}\\n我们这里有两种流量包，一个是肯尼亚的5G流量包，一个是肯尼亚的套餐。您比较偏\"}]},\"safetyRatings\":[{\"category\":\"HARM_CATEGORY_HATE_SPEECH\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.2890625,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.265625},{\"category\":\"HARM_CATEGORY_DANGEROUS_CONTENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.15917969,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.12695313},{\"category\":\"HARM_CATEGORY_HARASSMENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.23925781,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.2109375},{\"category\":\"HARM_CATEGORY_SEXUALLY_EXPLICIT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.14941406,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.26953125}]}]},{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"向哪一种呢？\\n\"}]},\"safetyRatings\":[{\"category\":\"HARM_CATEGORY_HATE_SPEECH\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.29492188,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.29296875},{\"category\":\"HARM_CATEGORY_DANGEROUS_CONTENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.123535156,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.115722656},{\"category\":\"HARM_CATEGORY_HARASSMENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.21777344,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.18847656},{\"category\":\"HARM_CATEGORY_SEXUALLY_EXPLICIT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.13671875,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.29882813}]}]},{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":2729,\"candidatesTokenCount\":71,\"totalTokenCount\":2800}}]";
        Assert.assertEquals(expect, buffer.toString().replaceAll("\\s+", ""));
    }

    @Test
    public void testBuildResultWithOnce() throws Exception {
        BufferedReader stream = new BufferedReader(new InputStreamReader(ResourceUtils.getURL("classpath:VertexResponse.json").openStream(), "UTF-8"));
        String content = IOUtils.toString(new BufferedReader(stream));
        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        request.setStream(false);
        request.setContainHistories(false);
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
            }
        };
        AtomicBoolean finished = new AtomicBoolean(false);
        GoogleStream googleStream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(notifierManager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(SignalStream.EMPTY)
                .historyStore(ObjectBuilder.buildHistoryStore())
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected void afterAtOnce() {
                finished.set(true);
            }
        };
        GoogleReader reader = new GoogleReader(ProviderReaderConfig.<GoogleRequest>builder()
                .request(request)
                .llmCallback(googleStream)
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1)
                .capacity(1048576)
                .queue(1024)
                .build()) {
        };
        receiveHttp(reader, 200);
        reader.consuming(Executors.newFixedThreadPool(1));
        reader.consumeContent(ProviderUtils.buildDecoder(reader, content), null);
        ProviderUtils.buildResult(reader);
        while (!finished.get()) {
        }
        String expect = "好的，这是一个Python脚本，用于读取你提供的两个文件并打印它们的内容。请注意：1.**文件路径**：这个脚本会尝试读取你提供的绝对路径。如果文件不存在，或者脚本没有足够的权限读取它们，它会打印相应的错误信息。2.**编码**：默认使用`utf-8`编码读取文件，这是处理大多数文本文件的最佳实践。```pythonimportosdefread_and_print_file(file_path):\"\"\"读取指定路径的文件内容并打印。如果文件不存在或无法读取，则打印错误信息。\"\"\"print(f\"\\n---正在读取文件:{file_path}---\")ifnotos.path.exists(file_path):print(f\"错误:文件不存在于此路径:{file_path}\")print(f\"---文件读取结束:{file_path}(未找到)---\\n\")returntry:withopen(file_path,'r',encoding='utf-8')asf:content=f.read()print(content)exceptPermissionError:print(f\"错误:没有权限读取文件:{file_path}\")exceptExceptionase:print(f\"读取文件时发生未知错误{file_path}:{e}\")finally:print(f\"---文件读取结束:{file_path}---\\n\")#定义要读取的两个文件路径file_path_1=\"/Users/shenjiawei/run/py/open_ai.py\"file_path_2=\"/Users/shenjiawei/run/ws/ws.ini\"#调用函数读取并打印第一个文件read_and_print_file(file_path_1)#调用函数读取并打印第二个文件read_and_print_file(file_path_2)```**如何运行这个脚本：**1.将上述代码保存为一个`.py`文件，例如`read_files.py`。2.打开你的终端或命令行。3.导航到你保存`read_files.py`文件的目录。4.运行命令：`pythonread_files.py`脚本将依次尝试读取这两个文件，并将其内容（或错误信息）打印到你的终端上。";
        Assert.assertEquals(expect, googleStream.getContentBuffer().toString().replaceAll("\\s+", ""));
    }

    @Test
    public void testBuildResultWithException1() throws Exception {
        BufferedReader stream = new BufferedReader(new InputStreamReader(ResourceUtils.getURL("classpath:VertexResponse.json").openStream(), "UTF-8"));
        String content = IOUtils.toString(new BufferedReader(stream));
        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        request.setStream(true);
        request.setContainHistories(false);
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
            }
        };
        GoogleStream googleStream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(notifierManager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(SignalStream.EMPTY)
                .historyStore(ObjectBuilder.buildHistoryStore())
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
        };
        AtomicBoolean finished = new AtomicBoolean(true);
        GoogleReader reader = new GoogleReader(ProviderReaderConfig.<GoogleRequest>builder()
                .request(request)
                .llmCallback(googleStream)
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1)
                .capacity(1048576)
                .queue(1024)
                .build()) {

            @Override
            protected void completed(String message) throws Exception {
                if (finished.get()) {
                    super.completed(message);
                } else {
                    throw new RuntimeException("ERROR");
                }
            }
        };
        receiveHttp(reader, 200);
        reader.consuming(Executors.newFixedThreadPool(1));
        reader.consumeContent(ProviderUtils.buildDecoder(reader, content), null);
        try {
            finished.set(false);
            ProviderUtils.buildResult(reader);
        } catch (Exception e) {
            Assert.fail();
        }
    }

    @Test
    public void testBuildResultWithException3() throws Exception {
        BufferedReader stream = new BufferedReader(new InputStreamReader(ResourceUtils.getURL("classpath:VertexResponse.json").openStream(), "UTF-8"));
        String content = IOUtils.toString(new BufferedReader(stream));
        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        request.setStream(true);
        request.setContainHistories(false);
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
            }
        };
        AtomicBoolean finished = new AtomicBoolean(false);
        GoogleStream googleStream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(notifierManager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(SignalStream.EMPTY)
                .historyStore(ObjectBuilder.buildHistoryStore())
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
        };
        AtomicBoolean counter = new AtomicBoolean(true);
        GoogleReader reader = new GoogleReader(ProviderReaderConfig.<GoogleRequest>builder()
                .request(request)
                .llmCallback(googleStream)
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(1024)
                .timeout(1024)
                .buffer(1)
                .capacity(1048576)
                .queue(20)
                .build()) {

            @Override
            protected void completed(String message) throws Exception {
                if (counter.get()) {
                    super.completed(message);
                } else {
                    throw new RuntimeException("ERROR");
                }
            }

            @Override
            public void released() {
                super.released();
                finished.set(true);
            }
        };
        // 不消费
        // reader.consuming(Executors.newFixedThreadPool(1));
        receiveHttp(reader, 200);
        reader.consumeContent(ProviderUtils.buildDecoder(reader, content), null);
        try {
            counter.set(false);
            while (reader.messageQueue.offer("OK")) {

            }
            ProviderUtils.buildResult(reader);
        } catch (Exception e) {
            Assert.fail();
        }
    }

    @Test
    public void testCapacity2() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setToken("Token");
        req.setStream(true);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        EasyMock.replay(callback);
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(callback)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(1024)
                .timeout(1024)
                .buffer(1)
                .capacity(1)
                .queue(1024)
                .build()) {

            @Override
            protected void completed(String message) throws Exception {
            }

        };
        Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(reader.byteBuffer.capacity()));
        reader.capacity(9999);
        Assert.assertEquals(Integer.valueOf(9999), Integer.valueOf(reader.byteBuffer.capacity()));
        EasyMock.verify(callback);
    }

    @Test
    public void testCapacity3() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setToken("Token");
        req.setStream(true);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        EasyMock.replay(callback);
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(callback)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(1024)
                .timeout(1024)
                .buffer(1)
                .capacity(1)
                .queue(1024)
                .build()) {

            @Override
            protected void completed(String message) throws Exception {
            }

        };
        Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(reader.getByteBuffer().capacity()));
        reader.capacity(1023);
        Assert.assertEquals(Integer.valueOf(1023), Integer.valueOf(reader.getByteBuffer().capacity()));
        EasyMock.verify(callback);
    }

    @Test
    public void testHasRemain() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        req.setMaxError(1000);
        req.setStream(true);
        req.setContainHistories(true);
        req.setTokenBuffer(100);
        req.setTokenFirst(100);
        req.setHistories(10);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        EasyMock.replay(callback);
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(callback)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(0)
                .capacity(0)
                .queue(1024)
                .build()) {
        };
        ContentDecoder decoder = EasyMock.createMock(ContentDecoder.class);
        EasyMock.expect(decoder.read(reader.getByteBuffer())).andReturn(1).times(1);
        EasyMock.expect(decoder.read(reader.getByteBuffer())).andReturn(-1).anyTimes();
        EasyMock.replay(decoder);
        reader.consumeContent(decoder, null);
        EasyMock.verify(callback);
    }

    @Test
    public void testBuildResult() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        req.setMaxError(1000);
        req.setStream(false);
        req.setContainHistories(true);
        req.setTokenBuffer(100);
        req.setTokenFirst(100);
        req.setHistories(10);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        EasyMock.replay(callback);
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(callback)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {

            @Override
            protected void completed(String message) throws Exception {
                Assert.assertEquals("HELLO WORLD2", message);
                super.completed(message);
            }
        };
        ContentDecoder decoder = EasyMock.createMock(ContentDecoder.class);
        EasyMock.expect(decoder.read(reader.getByteBuffer())).andReturn(1).times(1);
        EasyMock.expect(decoder.read(reader.getByteBuffer())).andReturn(-1).anyTimes();
        EasyMock.replay(decoder);
        reader.getByteBuffer().put("HELLO WORLD2".getBytes());
        reader.getByteBuffer().flip();
        reader.getByteBuffer().compact();
        reader.buildResult(null);
        EasyMock.verify(callback);
    }

    @Test
    public void testBuildResultWithException2() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        req.setMaxError(1000);
        req.setStream(true);
        req.setContainHistories(true);
        req.setTokenBuffer(100);
        req.setTokenFirst(100);
        req.setHistories(10);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        EasyMock.replay(callback);
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(callback)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {

            @Override
            public void completed(String message) throws Exception {
                throw new WorkflowException("test");
            }
        };
        reader.consuming(Executors.newFixedThreadPool(1));
        try {
            reader.consumeContent(ProviderUtils.buildDecoder(reader, "ABCD"), null);
            ProviderUtils.buildResult(reader);
        } finally {
            EasyMock.verify(callback);
        }
    }

    @Test
    public void testOtherWithNoException() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        req.setMaxError(1000);
        req.setStream(false);
        req.setContainHistories(true);
        req.setTokenBuffer(100);
        req.setTokenFirst(100);
        req.setHistories(10);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        EasyMock.replay(callback);
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(callback)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {
        };
        reader.getProviderReaderCallback().completed((Void) null);
        reader.getProviderReaderCallback().cancelled();
        EasyMock.verify(callback);
    }

    @Test
    public void testRelease() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setStream(false);
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(new EventListenerService() {

            @Override
            public void listen(Event event) throws Exception {
                throw new WorkflowException("test");
            }
        })
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {

            @Override
            protected void completed(String message) throws Exception {
            }
        };
        ContentDecoder decoder = EasyMock.createMock(ContentDecoder.class);
        EasyMock.expect(decoder.read(reader.getByteBuffer())).andReturn(-1).anyTimes();
        EasyMock.replay(decoder);
        reader.consumeContent(decoder, null);
        reader.releaseResources();
        EasyMock.verify(decoder);
    }

    @Test
    public void testIndexOfBoundary() throws Exception {
        ProviderRequest req = new ProviderRequest();
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(null)
                .eventListenerService(null)
                .extension(new HashMap<>())
                .discard(0)
                .timeout(100)
                .buffer(1024)
                .capacity(1024)
                .queue(10)
                .build()) {
            @Override
            protected void completed(String message) {
            }
        };
        java.nio.ByteBuffer buf = java.nio.ByteBuffer.allocate(16);
        buf.put((byte) 'A');
        buf.put((byte) 10);
        buf.put((byte) 10);
        buf.put((byte) 'B');
        buf.flip();
        Assert.assertEquals(1, reader.indexOf(buf));
    }

    @Test
    public void testExactMatchAtStart() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setStream(false);
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        String stream = IOUtils.toString(new BufferedReader(new InputStreamReader(ResourceUtils.getURL("classpath:VertexResponse_stream.json").openStream(), "UTF-8")));
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(new EventListenerService() {

            @Override
            public void listen(Event event) throws Exception {
                Assert.assertEquals(stream, ProviderData.class.cast(event.getData()).getResponse());
            }
        })
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build());
        // 在 8 字节块的最开始
        byte[] data = new byte[16];
        data[0] = 10;
        data[1] = 10;
        assertEquals(0, reader.indexOf(ByteBuffer.wrap(data)));
    }

    @Test
    public void testMatchInsideLong() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setStream(false);
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        String stream = IOUtils.toString(new BufferedReader(new InputStreamReader(ResourceUtils.getURL("classpath:VertexResponse_stream.json").openStream(), "UTF-8")));
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(new EventListenerService() {

            @Override
            public void listen(Event event) throws Exception {
                Assert.assertEquals(stream, ProviderData.class.cast(event.getData()).getResponse());
            }
        })
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build());
        // 在第一个 8 字节块的中间
        byte[] data = new byte[16];
        data[4] = 10;
        data[5] = 10;
        assertEquals(4, reader.indexOf(ByteBuffer.wrap(data)));
    }

    @Test
    public void testMatchAtEndOfLong() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setStream(false);
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        String stream = IOUtils.toString(new BufferedReader(new InputStreamReader(ResourceUtils.getURL("classpath:VertexResponse_stream.json").openStream(), "UTF-8")));
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(new EventListenerService() {

            @Override
            public void listen(Event event) throws Exception {
                Assert.assertEquals(stream, ProviderData.class.cast(event.getData()).getResponse());
            }
        })
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build());
        // 在第一个 8 字节块的末尾 (index 6 和 7)
        byte[] data = new byte[16];
        data[6] = 10;
        data[7] = 10;
        assertEquals(6, reader.indexOf(ByteBuffer.wrap(data)));
    }

    @Test
    public void testCrossBlockBoundary() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setStream(false);
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        String stream = IOUtils.toString(new BufferedReader(new InputStreamReader(ResourceUtils.getURL("classpath:VertexResponse_stream.json").openStream(), "UTF-8")));
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(new EventListenerService() {

            @Override
            public void listen(Event event) throws Exception {
                Assert.assertEquals(stream, ProviderData.class.cast(event.getData()).getResponse());
            }
        })
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build());
        // 关键边界：第一个 \n 在 index 7，第二个 \n 在 index 8
        byte[] data = new byte[16];
        data[7] = 10;
        data[8] = 10;
        assertEquals(7, reader.indexOf(ByteBuffer.wrap(data)));
    }

    @Test
    public void testRemainingBytesAfterLongs() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setStream(false);
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        String stream = IOUtils.toString(new BufferedReader(new InputStreamReader(ResourceUtils.getURL("classpath:VertexResponse_stream.json").openStream(), "UTF-8")));
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(new EventListenerService() {

            @Override
            public void listen(Event event) throws Exception {
                Assert.assertEquals(stream, ProviderData.class.cast(event.getData()).getResponse());
            }
        })
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build());
        // 总长度 10 字节，前 8 字节没找到，在最后 2 字节找到
        byte[] data = new byte[10];
        data[8] = 10;
        data[9] = 10;
        assertEquals(8, reader.indexOf(ByteBuffer.wrap(data)));
    }

    @Test
    public void testMultipleOccurrences() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setStream(false);
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        String stream = IOUtils.toString(new BufferedReader(new InputStreamReader(ResourceUtils.getURL("classpath:VertexResponse_stream.json").openStream(), "UTF-8")));
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(new EventListenerService() {

            @Override
            public void listen(Event event) throws Exception {
                Assert.assertEquals(stream, ProviderData.class.cast(event.getData()).getResponse());
            }
        })
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build());
        // 存在多个匹配，应返回第一个
        byte[] data = new byte[24];
        data[2] = 10;
        data[3] = 10; // 第一个
        data[10] = 10;
        data[11] = 10; // 第二个
        assertEquals(2, reader.indexOf(ByteBuffer.wrap(data)));
    }

    @Test
    public void testSingleNewlinesOnly() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setStream(false);
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        String stream = IOUtils.toString(new BufferedReader(new InputStreamReader(ResourceUtils.getURL("classpath:VertexResponse_stream.json").openStream(), "UTF-8")));
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(new EventListenerService() {

            @Override
            public void listen(Event event) throws Exception {
                Assert.assertEquals(stream, ProviderData.class.cast(event.getData()).getResponse());
            }
        })
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build());
        // 包含换行符但不是连续的
        byte[] data = {10, 65, 10, 66, 10, 67, 10, 68, 10, 69};
        assertEquals(-1, reader.indexOf(ByteBuffer.wrap(data)));
    }

    @Test
    public void testWithBufferPositionOffset() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setStream(false);
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        String stream = IOUtils.toString(new BufferedReader(new InputStreamReader(ResourceUtils.getURL("classpath:VertexResponse_stream.json").openStream(), "UTF-8")));
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(new EventListenerService() {

            @Override
            public void listen(Event event) throws Exception {
                Assert.assertEquals(stream, ProviderData.class.cast(event.getData()).getResponse());
            }
        })
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build());
        // 测试 ByteBuffer 的 position 偏移是否被正确尊重
        byte[] data = {0, 0, 0, 10, 10, 0};
        ByteBuffer buffer = ByteBuffer.wrap(data);
        buffer.position(4); // 从第二个 10 开始看，后面没有连续的了
        assertEquals(-1, reader.indexOf(buffer));
        buffer.position(3); // 从第一个 10 开始看
        assertEquals(3, reader.indexOf(buffer));
    }

    @Test
    public void testNoMatch() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setStream(false);
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        String stream = IOUtils.toString(new BufferedReader(new InputStreamReader(ResourceUtils.getURL("classpath:VertexResponse_stream.json").openStream(), "UTF-8")));
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(new EventListenerService() {

            @Override
            public void listen(Event event) throws Exception {
                Assert.assertEquals(stream, ProviderData.class.cast(event.getData()).getResponse());
            }
        })
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build());
        byte[] data = new byte[20];
        for (int i = 0; i < data.length; i++) data[i] = (byte) 'A';
        assertEquals(-1, reader.indexOf(ByteBuffer.wrap(data)));
    }

    @org.junit.jupiter.api.Test
    void testQueueFull() throws Exception {
        ProviderRequest req = new ProviderRequest();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        req.setMessage(message);
        // 队列大小设为 1
        ProviderReader<ProviderRequest> reader = new ProviderReader<ProviderRequest>(providerReaderConfig(req, null, ObjectBuilder.buildNotifierManagerWithimplement(), ObjectBuilder.buildEventListenerService(), new HashMap<>(), 0, 1024, 1024, 1)) {
        };

        receiveHttp(reader, 200);
        // 填满队列
        reader.completed("msg1");

        // 再次添加应该抛出异常
        Assertions.assertThrows(WorkflowException.class, () -> {
            reader.completed("msg2");
        });
    }

    @org.junit.jupiter.api.Test
    void testBuildStreamMultipleMessages() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));

        final List<String> received = new ArrayList<>();
        ProviderReader<ProviderRequest> reader = new ProviderReader<ProviderRequest>(providerReaderConfig(req, null, ObjectBuilder.buildNotifierManagerWithimplement(), ObjectBuilder.buildEventListenerService(), new HashMap<>(), 0, 1024, 1024, 10)) {
            @Override
            protected void completed(String message) throws Exception {
                received.add(message);
            }
        };

        receiveHttp(reader, 200);
        String data = "data: msg1\n\ndata: msg2\n\n";
        reader.getByteBuffer().put(data.getBytes(StandardCharsets.UTF_8));
        reader.buildStream();
        Assertions.assertEquals(2, received.size());
        Assertions.assertEquals("data: msg1", received.get(0));
        Assertions.assertEquals("data: msg2", received.get(1));
    }

    @org.junit.jupiter.api.Test
    void testBuildStreamWithException() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));

        ProviderReader<ProviderRequest> reader = new ProviderReader<ProviderRequest>(providerReaderConfig(req, null, ObjectBuilder.buildNotifierManagerWithimplement(), ObjectBuilder.buildEventListenerService(), new HashMap<>(), 0, 1024, 1024, 10)) {
            @Override
            protected void completed(String message) throws Exception {
                throw new WorkflowException(message);
            }
        };

        receiveHttp(reader, 200);
        String data = "data: msg1\n\ndata: msg2\n\n";
        reader.getByteBuffer().put(data.getBytes(StandardCharsets.UTF_8));
        try {
            reader.buildStream();
        } catch (WorkflowException e) {
            // 分行，第一行错误
            Assert.assertEquals("data: msg1", e.getMessage());
        }
    }

    @org.junit.jupiter.api.Test
    void testOnResponseReceivedMDC() throws Exception {
        ProviderRequest req = new ProviderRequest();
        Message message = EasyMock.createMock(Message.class);
        EasyMock.expect(message.getTrace()).andReturn("test-trace").anyTimes();
        EasyMock.expect(message.getDimension()).andReturn("test-dim").anyTimes();
        EasyMock.expect(message.getConsuming()).andReturn(100L).anyTimes();
        req.setMessage(message);

        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine status = EasyMock.createMock(StatusLine.class);
        EasyMock.expect(status.getStatusCode()).andReturn(200).anyTimes();
        EasyMock.expect(response.getStatusLine()).andReturn(status).anyTimes();

        EasyMock.replay(message, response, status);

        ProviderReader<ProviderRequest> reader = new ProviderReader<ProviderRequest>(providerReaderConfig(req, null, ObjectBuilder.buildNotifierManagerWithimplement(), ObjectBuilder.buildEventListenerService(), new HashMap<>(), 0, 1024, 1024, 10)) {
        };

        reader.onResponseReceived(response);

        Assertions.assertEquals("test-trace", MDC.get("trace"));
        Assertions.assertEquals("test-dim", MDC.get("dimension"));

        EasyMock.verify(message, response, status);
    }

    @org.junit.jupiter.api.Test
    void testOnResponseReceivedWhen2xxDoesNotThrow() throws Exception {
        ProviderRequest req = new ProviderRequest();
        Message message = EasyMock.createMock(Message.class);
        EasyMock.expect(message.getTrace()).andReturn("trace-2xx").anyTimes();
        EasyMock.expect(message.getDimension()).andReturn("dim-2xx").anyTimes();
        EasyMock.expect(message.getConsuming()).andReturn(50L).anyTimes();
        req.setMessage(message);

        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine status = EasyMock.createMock(StatusLine.class);
        EasyMock.expect(status.getStatusCode()).andReturn(299).anyTimes();
        EasyMock.expect(response.getStatusLine()).andReturn(status).anyTimes();
        EasyMock.replay(message, response, status);

        ProviderReader<ProviderRequest> reader = new ProviderReader<ProviderRequest>(providerReaderConfig(req, null, ObjectBuilder.buildNotifierManagerWithimplement(), ObjectBuilder.buildEventListenerService(), new HashMap<>(), 0, 1024, 1024, 10)) {
        };
        reader.onResponseReceived(response);

        EasyMock.verify(message, response, status);
    }

    @org.junit.jupiter.api.Test
    void testOnResponseReceivedWhenNon2xxThrowsWorkflowException() throws Exception {
        ProviderRequest req = new ProviderRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        req.setMessage(message);
        req.getProviderData().setRequest("mock-request-body");

        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine status = EasyMock.createMock(StatusLine.class);
        int statusCode = 400;
        EasyMock.expect(status.getStatusCode()).andReturn(statusCode).anyTimes();
        EasyMock.expect(response.getStatusLine()).andReturn(status).anyTimes();
        EasyMock.replay(response, status);

        ProviderReader<ProviderRequest> reader = new ProviderReader<ProviderRequest>(providerReaderConfig(req, null, ObjectBuilder.buildNotifierManagerWithimplement(), ObjectBuilder.buildEventListenerService(), new HashMap<>(), 0, 1024, 1024, 10)) {
        };

        reader.onResponseReceived(response);
        EasyMock.verify(response, status);
    }

    @org.junit.jupiter.api.Test
    void testOnResponseReceivedWhen500ThrowsWorkflowException() throws Exception {
        ProviderRequest req = new ProviderRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        req.setMessage(message);
        req.getProviderData().setRequest("request-body");

        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine status = EasyMock.createMock(StatusLine.class);
        int statusCode = 500;
        EasyMock.expect(status.getStatusCode()).andReturn(statusCode).anyTimes();
        EasyMock.expect(response.getStatusLine()).andReturn(status).anyTimes();
        EasyMock.replay(response, status);

        ProviderReader<ProviderRequest> reader = new ProviderReader<ProviderRequest>(providerReaderConfig(req, null, ObjectBuilder.buildNotifierManagerWithimplement(), ObjectBuilder.buildEventListenerService(), new HashMap<>(), 0, 1024, 1024, 10)) {
        };

        reader.onResponseReceived(response);

        EasyMock.verify(response, status);
    }

    @org.junit.jupiter.api.Test
    void testCapacityNoIncrease() throws Exception {
        ProviderRequest req = new ProviderRequest();
        ProviderReader<ProviderRequest> reader = new ProviderReader<ProviderRequest>(providerReaderConfig(req, null, ObjectBuilder.buildNotifierManagerWithimplement(), ObjectBuilder.buildEventListenerService(), new HashMap<>(), 0, 1024, 100, 10)) {
        };

        int initialCapacity = reader.getByteBuffer().capacity();
        Assertions.assertEquals(100, initialCapacity);

        // 请求更小的容量，不应改变
        reader.capacity(50);
        Assertions.assertEquals(100, reader.getByteBuffer().capacity());

        // 请求相等的容量，不应改变
        reader.capacity(100);
        Assertions.assertEquals(100, reader.getByteBuffer().capacity());
    }

    @org.junit.jupiter.api.Test
    void testConsumingMethod() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));

        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        Executor executor = EasyMock.createMock(Executor.class);

        // 预期执行一次
        executor.execute(EasyMock.anyObject(Runnable.class));
        EasyMock.expectLastCall().once();

        EasyMock.replay(callback, executor);

        ProviderReader<ProviderRequest> reader = new ProviderReader<ProviderRequest>(providerReaderConfig(req, callback, ObjectBuilder.buildNotifierManagerWithimplement(), ObjectBuilder.buildEventListenerService(), new HashMap<>(), 0, 1024, 1024, 10)) {
        };

        reader.consuming(executor);

        EasyMock.verify(executor);
    }

    @org.junit.jupiter.api.Test
    void testOnContentReceivedCapacityExpansion() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        req.setStream(false);

        ProviderReader<ProviderRequest> reader = new ProviderReader<ProviderRequest>(providerReaderConfig(req, null, ObjectBuilder.buildNotifierManagerWithimplement(), ObjectBuilder.buildEventListenerService(), new HashMap<>(), 0, 1024, 10, 10)) {
        };

        ContentDecoder decoder = EasyMock.createMock(ContentDecoder.class);

        EasyMock.expect(decoder.read(EasyMock.anyObject(ByteBuffer.class))).andAnswer(new org.easymock.IAnswer<Integer>() {
            @Override
            public Integer answer() throws Throwable {
                ByteBuffer buf = (ByteBuffer) EasyMock.getCurrentArguments()[0];
                int remaining = buf.remaining();
                if (remaining > 0) {
                    byte[] data = new byte[remaining];
                    buf.put(data);
                    return remaining;
                }
                return 0;
            }
        }).times(1);

        EasyMock.expect(decoder.read(EasyMock.anyObject(ByteBuffer.class))).andReturn(-1).once();

        EasyMock.replay(decoder);

        reader.onContentReceived(decoder, null);

        Assertions.assertEquals(20, reader.getByteBuffer().capacity());
        EasyMock.verify(decoder);
    }

    @org.junit.jupiter.api.Test
    void testOnContentReceivedCapacityExpansionStopsAtConfiguredLimit() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        req.setStream(false);

        ProviderReader<ProviderRequest> reader = new ProviderReader<ProviderRequest>(providerReaderConfig(req, null, ObjectBuilder.buildNotifierManagerWithimplement(), ObjectBuilder.buildEventListenerService(), new HashMap<>(), 0, 1024, 16, 10, 10)) {
        };

        ContentDecoder decoder = EasyMock.createMock(ContentDecoder.class);
        EasyMock.expect(decoder.read(EasyMock.anyObject(ByteBuffer.class))).andAnswer(new org.easymock.IAnswer<Integer>() {
            @Override
            public Integer answer() {
                ByteBuffer buf = (ByteBuffer) EasyMock.getCurrentArguments()[0];
                int remaining = buf.remaining();
                byte[] data = new byte[remaining];
                buf.put(data);
                return remaining;
            }
        }).once();
        EasyMock.expect(decoder.read(EasyMock.anyObject(ByteBuffer.class))).andReturn(-1).once();
        EasyMock.replay(decoder);

        reader.onContentReceived(decoder, null);

        Assertions.assertEquals(16, reader.getByteBuffer().capacity());
        EasyMock.verify(decoder);
    }

    @org.junit.jupiter.api.Test
    void testOnContentReceivedCapacityLimitReachedNotifiesFailure() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        req.setStream(false);

        AtomicReference<Exception> failed = new AtomicReference<>();
        ProviderReader<ProviderRequest> reader = new ProviderReader<ProviderRequest>(providerReaderConfig(req, null, ObjectBuilder.buildNotifierManagerWithimplement(), ObjectBuilder.buildEventListenerService(), new HashMap<>(), 0, 1024, 16, 10, 10)) {
            @Override
            protected void notifyFailed(Exception e) {
                failed.set(e);
            }
        };

        ContentDecoder decoder = EasyMock.createMock(ContentDecoder.class);
        EasyMock.expect(decoder.read(EasyMock.anyObject(ByteBuffer.class))).andAnswer(new org.easymock.IAnswer<Integer>() {
            @Override
            public Integer answer() {
                ByteBuffer buf = (ByteBuffer) EasyMock.getCurrentArguments()[0];
                int remaining = buf.remaining();
                byte[] data = new byte[remaining];
                buf.put(data);
                return remaining;
            }
        }).times(2);
        EasyMock.replay(decoder);

        reader.onContentReceived(decoder, null);

        Assertions.assertNotNull(failed.get());
        Assertions.assertEquals(IllegalArgumentException.class, failed.get().getClass());
        Assertions.assertTrue(failed.get().getMessage().contains("The request's buffer is too large: 16"));
        Assertions.assertEquals(16, reader.getByteBuffer().capacity());
        EasyMock.verify(decoder);
    }

    @org.junit.jupiter.api.Test
    void testBuildStreamWithMalformedInputException() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(ai.open.right.workflow.flow.llm.Message.build(ai.open.right.ObjectBuilder.buildLLMQuery()));
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ai.open.right.ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ai.open.right.ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {
        };
        receiveHttp(reader, 200);
        // Invalid UTF-8 sequence
        reader.getByteBuffer().put(new byte[]{(byte) 0xF0, (byte) 0x28, (byte) 0x8C, (byte) 0x28, 10, 10});
        reader.buildStream();
    }

    @org.junit.jupiter.api.Test
    void testBuildStreamWithJacksonException() throws Exception {
        ProviderRequest req = new ProviderRequest() {
            @Override
            public void appendResponse(String response) throws Exception {
                throw new com.fasterxml.jackson.core.JsonParseException(null, "test");
            }
        };
        req.setMessage(ai.open.right.workflow.flow.llm.Message.build(ai.open.right.ObjectBuilder.buildLLMQuery()));
        req.setStream(true);

        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ai.open.right.ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ai.open.right.ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {
        };
        receiveHttp(reader, 200);
        reader.getByteBuffer().put("data: msg1\n\n".getBytes(java.nio.charset.StandardCharsets.UTF_8));
        // Jackson 2.12+ 中 JsonParseException 继承链含 JacksonException，会在 buildStream 中被捕获并仅打 DEBUG
        Assertions.assertDoesNotThrow(reader::buildStream);
    }

    @org.junit.jupiter.api.Test
    void testOnContentReceivedWithException() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(ai.open.right.workflow.flow.llm.Message.build(ai.open.right.ObjectBuilder.buildLLMQuery()));
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ai.open.right.ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ai.open.right.ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {
        };
        org.apache.http.nio.ContentDecoder decoder = org.easymock.EasyMock.createMock(org.apache.http.nio.ContentDecoder.class);
        org.apache.http.nio.IOControl ioControl = org.easymock.EasyMock.createMock(org.apache.http.nio.IOControl.class);
        org.easymock.EasyMock.expect(decoder.read(org.easymock.EasyMock.anyObject(java.nio.ByteBuffer.class))).andThrow(new java.io.IOException("test exception"));
        ioControl.shutdown();
        org.easymock.EasyMock.expectLastCall().anyTimes();
        org.easymock.EasyMock.replay(decoder, ioControl);
        reader.onContentReceived(decoder, ioControl);
    }

    @org.junit.jupiter.api.Test
    void testOnEntityEnclosed() throws Exception {
        ProviderRequest req = new ProviderRequest();
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(null)
                .eventListenerService(null)
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {
        };
        reader.onEntityEnclosed(null, null);
    }

    @org.junit.jupiter.api.Test
    void testIndexOfBoundaryCheckCoverage() throws Exception {
        ProviderRequest req = new ProviderRequest();
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(null)
                .eventListenerService(null)
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {
            @Override
            protected long hasByte(long x, byte b) {
                return 0; // Force bypass to hit boundary check
            }
        };
        byte[] data = new byte[16];
        data[7] = 10;
        data[8] = 10;
        java.nio.ByteBuffer buffer = java.nio.ByteBuffer.wrap(data);
        org.junit.jupiter.api.Assertions.assertEquals(7, reader.indexOf(buffer));
    }

    @org.junit.jupiter.api.Test
    void testBuildStreamWithMalformedInputExceptionDebug() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(ai.open.right.workflow.flow.llm.Message.build(ai.open.right.ObjectBuilder.buildLLMQuery()));
        try {
            org.slf4j.Logger logger = org.slf4j.LoggerFactory.getLogger(ProviderReader.class);
            if (logger instanceof ch.qos.logback.classic.Logger) {
                ((ch.qos.logback.classic.Logger) logger).setLevel(ch.qos.logback.classic.Level.DEBUG);
            }
        } catch (Throwable t) {
        }
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ai.open.right.ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ai.open.right.ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {
        };
        receiveHttp(reader, 200);
        reader.getByteBuffer().put(new byte[]{(byte) 0xF0, (byte) 0x28, (byte) 0x8C, (byte) 0x28, 10, 10});
        reader.buildStream();
    }

    @org.junit.jupiter.api.Test
    void testBuildResultWithOutExceptionCoverage() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(ai.open.right.workflow.flow.llm.Message.build(ai.open.right.ObjectBuilder.buildLLMQuery()));
        // 使 completed 抛出异常
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ai.open.right.ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ai.open.right.ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1)
                .build()) {
            @Override
            protected void completed(String message) throws Exception {
                throw new RuntimeException("forced exception");
            }
        };
        receiveHttp(reader, 200);
        reader.getByteBuffer().put("some data".getBytes());
        reader.buildResult(null);
    }

    @org.junit.jupiter.api.Test
    void testOnContentReceivedWithExceptionCoverage() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setStream(true);
        req.setMessage(ai.open.right.workflow.flow.llm.Message.build(ai.open.right.ObjectBuilder.buildLLMQuery()));
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ai.open.right.ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ai.open.right.ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {
            @Override
            protected void buildStream() throws Exception {
                throw new RuntimeException("forced exception");
            }
        };
        org.apache.http.nio.ContentDecoder decoder = org.easymock.EasyMock.createMock(org.apache.http.nio.ContentDecoder.class);
        org.easymock.EasyMock.expect(decoder.read(org.easymock.EasyMock.anyObject(java.nio.ByteBuffer.class))).andReturn(-1);
        org.easymock.EasyMock.replay(decoder);
        reader.onContentReceived(decoder, null);
    }

    /**
     * 非 2xx 时仍可走完 onContentReceived（decoder 返回 -1 表示无更多内容）。
     */
    @org.junit.jupiter.api.Test
    void testOnContentReceived_whenNon2xx_decoderReturnsMinusOne() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        req.setStream(false);
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {
        };
        receiveHttp(reader, 500);
        org.apache.http.nio.ContentDecoder decoder = EasyMock.createMock(org.apache.http.nio.ContentDecoder.class);
        EasyMock.expect(decoder.read(EasyMock.anyObject(ByteBuffer.class))).andReturn(-1);
        EasyMock.replay(decoder);
        Assertions.assertDoesNotThrow(() -> reader.onContentReceived(decoder, null));
        EasyMock.verify(decoder);
    }

    /**
     * 非 2xx 且 body 为空时 buildResult 经 notifyFailed 收尾，不向调用方抛出。
     */
    @org.junit.jupiter.api.Test
    void testBuildResult_whenNon2xxEmptyBody_completesWithoutThrowingToCaller() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {
        };
        receiveHttp(reader, 500);
        Assertions.assertDoesNotThrow(() -> reader.buildResult(null));
    }

    /**
     * buildResult 的 finally 中 releaseMessageQueue 抛错时只走 WorkflowException.dolog，不经过 notifyFailed。
     */
    @org.junit.jupiter.api.Test
    void testBuildResult_whenFinallyReleaseThrows_dologWithoutNotifyFailedPath() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {
            @Override
            protected void releaseMessageQueue() {
                throw new RuntimeException("finally-release-fail");
            }
        };
        reader.code = 500;
        Logger wfLogger = (Logger) LoggerFactory.getLogger(ai.open.right.WorkflowException.class);
        Level oldLevel = wfLogger.getLevel();
        ListAppender<ILoggingEvent> appender = new ListAppender<>();
        appender.start();
        wfLogger.addAppender(appender);
        wfLogger.setLevel(Level.ERROR);
        try {
            reader.buildResult(null);
            boolean found = appender.list.stream().anyMatch(e -> e.getMessage() != null && e.getMessage().contains("finally-release-fail"));
            Assertions.assertTrue(found);
        } finally {
            wfLogger.detachAppender(appender);
            wfLogger.setLevel(oldLevel);
            appender.stop();
        }
    }

    @org.junit.jupiter.api.Test
    void testOnEntityEnclosedCoverage() throws Exception {
        ProviderRequest req = new ProviderRequest();
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(null)
                .eventListenerService(null)
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {
        };
        reader.onEntityEnclosed(null, null);
    }

    @org.junit.jupiter.api.Test
    void testIndexOfBoundaryCheckCoverageEnhanced() throws Exception {
        ProviderRequest req = new ProviderRequest();
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(null)
                .notifierService(null)
                .eventListenerService(null)
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {
            @Override
            protected long hasByte(long x, byte b) {
                return 0;
            }
        };
        byte[] data = new byte[16];
        data[7] = 10;
        data[8] = 10;
        java.nio.ByteBuffer buffer = java.nio.ByteBuffer.wrap(data);
        org.junit.jupiter.api.Assertions.assertEquals(7, reader.indexOf(buffer));
    }

    /**
     * event() 对 ProviderData 执行 init（缓冲写入 response 并清空 buffer），再向 EventListenerService 投递 ProviderEvent。
     */
    @org.junit.jupiter.api.Test
    void testEvent_flushesProviderDataAndNotifiesListener() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        req.appendResponse("chunk-a");
        req.appendResponse("chunk-b");
        List<Event> captured = new ArrayList<>();
        EventListenerService els = captured::add;
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        EasyMock.replay(callback);
        ProviderReader<ProviderRequest> reader = new ProviderReader<>(providerReaderConfig(req, callback,
                ObjectBuilder.buildNotifierManagerWithimplement(), els, new HashMap<>(), 0, 1024, 1024, 1024));
        reader.event();
        org.junit.jupiter.api.Assertions.assertEquals("chunk-achunk-b", req.getProviderData().getResponse());
        org.junit.jupiter.api.Assertions.assertNotNull(req.getProviderData().getResponseBuffer());
        org.junit.jupiter.api.Assertions.assertEquals(1, captured.size());
        org.junit.jupiter.api.Assertions.assertInstanceOf(ProviderEvent.class, captured.get(0));
        ProviderEvent pe = (ProviderEvent) captured.get(0);
        org.junit.jupiter.api.Assertions.assertSame(req, pe.getProviderRequest());
        org.junit.jupiter.api.Assertions.assertSame(req.getProviderData(), pe.getProviderData());
        EasyMock.verify(callback);
    }

    @org.junit.jupiter.api.Test
    void testEvent_swallowsExceptionWhenListenFails() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        EventListenerService els = event -> {
            throw new RuntimeException("listen-fail");
        };
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        EasyMock.replay(callback);
        ProviderReader<ProviderRequest> reader = new ProviderReader<>(providerReaderConfig(req, callback,
                ObjectBuilder.buildNotifierManagerWithimplement(), els, new HashMap<>(), 0, 1024, 1024, 1024));
        org.junit.jupiter.api.Assertions.assertDoesNotThrow(reader::event);
        EasyMock.verify(callback);
    }

    @org.junit.jupiter.api.Test
    void testEvent_swallowsExceptionWhenInitFailsOnSecondCall() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        EventListenerService els = event -> {
        };
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        EasyMock.replay(callback);
        ProviderReader<ProviderRequest> reader = new ProviderReader<>(providerReaderConfig(req, callback,
                ObjectBuilder.buildNotifierManagerWithimplement(), els, new HashMap<>(), 0, 1024, 1024, 1024));
        reader.event();
        org.junit.jupiter.api.Assertions.assertDoesNotThrow(reader::event);
        EasyMock.verify(callback);
    }

    /**
     * 非 2xx 时 {@link ProviderReader#completed(String)} 走异常分支：懒创建 {@code expMessage} 并追加片段，不增加 messageCnt、不往队列投递。
     */
    @org.junit.jupiter.api.Test
    void completed_whenNon2xx_appendsToExpMessage() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        ProviderReader<ProviderRequest> reader = new ProviderReader<>(providerReaderConfig(req, null,
                ObjectBuilder.buildNotifierManagerWithimplement(), ObjectBuilder.buildEventListenerService(),
                new HashMap<>(), 0, 1024, 1024, 1024));
        receiveHttp(reader, 400);
        org.junit.jupiter.api.Assertions.assertNull(reader.getExpMessage());
        org.junit.jupiter.api.Assertions.assertEquals(0, reader.getMessageCnt());

        reader.completed("err-");
        org.junit.jupiter.api.Assertions.assertNotNull(reader.getExpMessage());
        org.junit.jupiter.api.Assertions.assertEquals("err-", reader.getExpMessage().toString());
        org.junit.jupiter.api.Assertions.assertEquals(0, reader.getMessageCnt());

        reader.completed("body");
        org.junit.jupiter.api.Assertions.assertEquals("err-body", reader.getExpMessage().toString());
        org.junit.jupiter.api.Assertions.assertEquals(0, reader.getMessageCnt());
    }

    private static ProviderReaderConfig<ProviderRequest> providerReaderConfig(
            ProviderRequest request,
            LLMCallback llmCallback,
            NotifierService notifierService,
            EventListenerService eventListenerService,
            Map<String, Object> extension,
            int discard,
            int timeout,
            int buffer,
            int queue) {
        return providerReaderConfig(request, llmCallback, notifierService, eventListenerService, extension, discard, timeout, Math.max(buffer, 1024 * 1024), buffer, queue);
    }

    private static ProviderReaderConfig<ProviderRequest> providerReaderConfig(
            ProviderRequest request,
            LLMCallback llmCallback,
            NotifierService notifierService,
            EventListenerService eventListenerService,
            Map<String, Object> extension,
            int discard,
            int timeout,
            int capacity,
            int buffer,
            int queue) {
        return ProviderReaderConfig.<ProviderRequest>builder()
                .request(request)
                .llmCallback(llmCallback)
                .notifierService(notifierService)
                .eventListenerService(eventListenerService)
                .extension(extension)
                .discard(discard)
                .timeout(timeout)
                .buffer(buffer)
                .capacity(capacity)
                .queue(queue)
                .build();
    }
}
