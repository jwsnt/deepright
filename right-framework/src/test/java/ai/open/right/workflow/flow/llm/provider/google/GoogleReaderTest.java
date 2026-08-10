package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.provider.ProviderData;
import ai.open.right.workflow.flow.llm.provider.ProviderUtils;
import com.fasterxml.jackson.core.JsonParseException;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.springframework.util.ResourceUtils;

import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.util.HashMap;
import java.util.concurrent.Executors;
import java.util.concurrent.atomic.AtomicBoolean;

import ai.open.right.workflow.flow.llm.provider.ProviderReaderConfig;
public class GoogleReaderTest {

    @Test
    public void testContent() throws Exception {
        BufferedReader stream = new BufferedReader(new InputStreamReader(ResourceUtils.getURL("classpath:VertexResponse_stream.json").openStream(), "UTF-8"));
        StringBuilder buffer = new StringBuilder();
        String content = IOUtils.toString(stream);
        GoogleRequest request = EasyMock.createMock(GoogleRequest.class);
        request.appendResponse(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getProviderData()).andReturn(new ProviderData()).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.hasFunCall()).andReturn(false).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.replay(request);
        AtomicBoolean finished = new AtomicBoolean(false);
        GoogleReader googleReader = new GoogleReader(ProviderReaderConfig.<GoogleRequest>builder()
                .request(request)
                .llmCallback(new LLMCallback() {
            @Override
            public void callback(String message) throws Exception {
                buffer.append(message);
                finished.set(true);
            }
        })
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(1024)
                .timeout(1024)
                .buffer(10024)
                .capacity(10024)
                .queue(1024)
                .build());
        googleReader.consuming(Executors.newSingleThreadExecutor());
        googleReader.setSkipFirst(false);
        googleReader.consumeContent(ProviderUtils.buildDecoder(googleReader, content), null);
        ProviderUtils.buildResult(googleReader);
        while (!finished.get()) {
            Thread.sleep(10);
        }
        String expect = "[{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"${\"}]}}]},{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"I_01;S_00;S_01}\\n\"}]},\"safetyRatings\":[{\"category\":\"HARM_CATEGORY_HATE_SPEECH\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.24804688,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.13671875},{\"category\":\"HARM_CATEGORY_DANGEROUS_CONTENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.28710938,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.06738281},{\"category\":\"HARM_CATEGORY_HARASSMENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.28710938,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.14355469},{\"category\":\"HARM_CATEGORY_SEXUALLY_EXPLICIT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.31835938,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.10986328}]}]},{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"我们是科技，主要做非洲业务。\\n\\n${S_0\"}]},\"safetyRatings\":[{\"category\":\"HARM_CATEGORY_HATE_SPEECH\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.38085938,\"severity\":\"HARM_SEVERITY_MEDIUM\",\"severityScore\":0.4375},{\"category\":\"HARM_CATEGORY_DANGEROUS_CONTENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.13964844,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.119140625},{\"category\":\"HARM_CATEGORY_HARASSMENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.3203125,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.31054688},{\"category\":\"HARM_CATEGORY_SEXUALLY_EXPLICIT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.14257813,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.21289063}]}]},{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"3=2,3}\\n我们这里有两种流量包，一个是肯尼亚的5G流量包，一个是肯尼亚的套餐。您比较偏\"}]},\"safetyRatings\":[{\"category\":\"HARM_CATEGORY_HATE_SPEECH\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.2890625,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.265625},{\"category\":\"HARM_CATEGORY_DANGEROUS_CONTENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.15917969,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.12695313},{\"category\":\"HARM_CATEGORY_HARASSMENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.23925781,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.2109375},{\"category\":\"HARM_CATEGORY_SEXUALLY_EXPLICIT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.14941406,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.26953125}]}]},{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"向哪一种呢？\\n\"}]},\"safetyRatings\":[{\"category\":\"HARM_CATEGORY_HATE_SPEECH\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.29492188,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.29296875},{\"category\":\"HARM_CATEGORY_DANGEROUS_CONTENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.123535156,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.115722656},{\"category\":\"HARM_CATEGORY_HARASSMENT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.21777344,\"severity\":\"HARM_SEVERITY_NEGLIGIBLE\",\"severityScore\":0.18847656},{\"category\":\"HARM_CATEGORY_SEXUALLY_EXPLICIT\",\"probability\":\"NEGLIGIBLE\",\"probabilityScore\":0.13671875,\"severity\":\"HARM_SEVERITY_LOW\",\"severityScore\":0.29882813}]}]},{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":2729,\"candidatesTokenCount\":71,\"totalTokenCount\":2800}}]";
        Assertions.assertEquals(expect.replaceAll("\\s+", ""), buffer.toString().replaceAll("\\s+", ""));
    }

    @Test
    public void testPrepareSkipFirst() throws Exception {
        GoogleRequest request = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.replay(request);
        GoogleReader reader = new GoogleReader(ProviderReaderConfig.<GoogleRequest>builder()
                .request(request)
                .llmCallback(null)
                .notifierService(null)
                .eventListenerService(null)
                .extension(new HashMap<>())
                .discard(1024)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build());
        reader.setSkipFirst(true);
        reader.getByteBuffer().put((byte) '[');
        reader.getByteBuffer().flip();
        reader.prepare();
        Assertions.assertEquals(1, reader.getByteBuffer().position());
        Assertions.assertFalse(reader.getSkipFirst());
    }

    @Test
    public void testCompletedWithInvalidJson() throws Exception {
        GoogleRequest request = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.replay(request);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        // callback should NOT be called
        EasyMock.replay(callback);
        GoogleReader reader = new GoogleReader(ProviderReaderConfig.<GoogleRequest>builder()
                .request(request)
                .llmCallback(callback)
                .notifierService(null)
                .eventListenerService(null)
                .extension(new HashMap<>())
                .discard(1024)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build());
        reader.completed("not a json");
        EasyMock.verify(callback);
    }

    @Test
    public void testCheck() throws Exception {
        String content = "[]";
        GoogleRequest request = EasyMock.createMock(GoogleRequest.class);
        request.setUrl(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getUrl()).andReturn("http://www.w3c.com").anyTimes();
        request.appendResponse(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        request.autoDump(EasyMock.anyObject(WorkflowException.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getProviderData()).andReturn(new ProviderData()).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.hasFunCall()).andReturn(false).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.replay(request);
        GoogleReader googleReader = new GoogleReader(ProviderReaderConfig.<GoogleRequest>builder()
                .request(request)
                .llmCallback(new LLMCallback() {
            @Override
            public void callback(String message) throws Exception {
            }
        })
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(1024)
                .timeout(1024)
                .buffer(10024)
                .capacity(10024)
                .queue(1024)
                .build());
        googleReader.consuming(Executors.newSingleThreadExecutor());
        googleReader.setSkipFirst(false);
        try {
            googleReader.consumeContent(ProviderUtils.buildDecoder(googleReader, content), null);
        } catch (WorkflowException e) {
            Assert.assertEquals(ProtocolCode.C914, e.getCode());
        } finally {
            EasyMock.verify(request);
        }
    }

    @Test
    public void testWithJsonException() throws Exception {
        String content = "[]";
        GoogleRequest request = EasyMock.createMock(GoogleRequest.class);
        request.appendResponse(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getProviderData()).andReturn(new ProviderData()).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.hasFunCall()).andReturn(false).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.replay(request);
        GoogleReader googleReader = new GoogleReader(ProviderReaderConfig.<GoogleRequest>builder()
                .request(request)
                .llmCallback(new LLMCallback() {
            @Override
            public void callback(String message) throws Exception {
            }
        })
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(1024)
                .timeout(1024)
                .buffer(10024)
                .capacity(10024)
                .queue(1024)
                .build()) {

            @Override
            protected void check(String message) throws Exception {
                throw new JsonParseException(message);
            }
        };
        googleReader.consuming(Executors.newSingleThreadExecutor());
        googleReader.setSkipFirst(false);
        // 不应该抛出异常
        googleReader.consumeContent(ProviderUtils.buildDecoder(googleReader, content), null);
    }
}
