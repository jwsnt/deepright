package ai.open.right.workflow.flow.llm.provider.deepseek;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.provider.ProviderData;
import ai.open.right.workflow.flow.llm.provider.ProviderUtils;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiReader;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.io.BufferedReader;
import java.io.ByteArrayInputStream;
import java.io.InputStreamReader;
import java.util.HashMap;
import java.util.concurrent.Executors;
import java.util.concurrent.atomic.AtomicBoolean;

import ai.open.right.workflow.flow.llm.provider.ProviderReaderConfig;
public class DeepSeekReaderTest {

    @Test
    public void test() throws Exception {
        String content = IOUtils.toString(new BufferedReader(new InputStreamReader(new ByteArrayInputStream("Hello".getBytes()))));
        OpenAiRequest req = EasyMock.createMock(OpenAiRequest.class);
        req.appendResponse(content);
        EasyMock.expectLastCall().anyTimes();
        req.autoDump();
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(req.hasAutoDump()).andReturn(false).anyTimes();
        EasyMock.expect(req.getProviderData()).andReturn(new ProviderData()).anyTimes();
        EasyMock.expect(req.hasFunCall()).andReturn(false).anyTimes();
        EasyMock.expect(req.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(req.getStream()).andReturn(true).anyTimes();
        LLMCallback cal = EasyMock.createMock(LLMCallback.class);
        cal.callback(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(req, cal);
        AtomicBoolean finished = new AtomicBoolean(false);
        StringBuilder builder = new StringBuilder();
        OpenAiReader reader = new OpenAiReader(ProviderReaderConfig.<OpenAiRequest>builder()
                .request(req)
                .llmCallback(cal)
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(1024)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {

            @Override
            protected void completed(String message) throws Exception {
                builder.append(message);
                finished.set(true);
                super.completed(message);
            }
        };
        reader.consuming(Executors.newFixedThreadPool(1));
        ProviderUtils.invokeOnContentReceived(reader, ProviderUtils.buildDecoder(reader, content), null);
        ProviderUtils.buildResult(reader);
        while (!finished.get()) {
        }
        Assert.assertEquals("Hello", builder.toString());
        EasyMock.verify(req, cal);
    }

    @Test
    public void testWithDone() throws Exception {
        OpenAiRequest req = EasyMock.createMock(OpenAiRequest.class);
        req.appendResponse("data: [DONE]");
        EasyMock.expectLastCall().anyTimes();
        req.autoDump();
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(req.getProviderData()).andReturn(new ProviderData()).anyTimes();
        EasyMock.expect(req.hasFunCall()).andReturn(false).anyTimes();
        EasyMock.expect(req.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(req.getStream()).andReturn(true).anyTimes();
        LLMCallback cal = EasyMock.createMock(LLMCallback.class);
        cal.callback(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(req, cal);
        String content = IOUtils.toString(new BufferedReader(new InputStreamReader(new ByteArrayInputStream("data: [DONE]".getBytes()))));
        AtomicBoolean finished = new AtomicBoolean(false);
        StringBuilder builder = new StringBuilder();
        OpenAiReader reader = new OpenAiReader(ProviderReaderConfig.<OpenAiRequest>builder()
                .request(req)
                .llmCallback(cal)
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(1024)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {
            @Override
            protected void completed(String message) throws Exception {
                super.completed(message);
                builder.append(message);
                finished.set(true);
            }
        };
        reader.consuming(Executors.newFixedThreadPool(1));
        reader.consumeContent(ProviderUtils.buildDecoder(reader, content), null);
        ProviderUtils.buildResult(reader);
        while (!finished.get()) {
        }
        Assert.assertEquals("data: [DONE]", builder.toString());
        EasyMock.verify(req, cal);
    }

    @Test
    public void testWithOnce() throws Exception {
        String content = IOUtils.toString(new BufferedReader(new InputStreamReader(new ByteArrayInputStream("data: [DONE]".getBytes()))));
        OpenAiRequest req = EasyMock.createMock(OpenAiRequest.class);
        req.autoDump();
        EasyMock.expectLastCall().anyTimes();
        req.appendResponse(content);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(req.getProviderData()).andReturn(new ProviderData()).anyTimes();
        EasyMock.expect(req.hasFunCall()).andReturn(false).anyTimes();
        EasyMock.expect(req.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(req.getStream()).andReturn(false).anyTimes();
        LLMCallback cal = EasyMock.createMock(LLMCallback.class);
        cal.callback(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(req, cal);
        AtomicBoolean finished = new AtomicBoolean(false);
        StringBuilder builder = new StringBuilder();
        OpenAiReader reader = new OpenAiReader(ProviderReaderConfig.<OpenAiRequest>builder()
                .request(req)
                .llmCallback(cal)
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(1024)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {
            @Override
            protected void completed(String message) throws Exception {
                super.completed(message);
                builder.append(message);
                finished.set(true);
            }
        };
        reader.consuming(Executors.newFixedThreadPool(1));
        reader.consumeContent(ProviderUtils.buildDecoder(reader, content), null);
        ProviderUtils.buildResult(reader);
        while (!finished.get()) {
        }
        Assert.assertEquals("data: [DONE]", builder.toString());
        EasyMock.verify(req, cal);
    }
}
