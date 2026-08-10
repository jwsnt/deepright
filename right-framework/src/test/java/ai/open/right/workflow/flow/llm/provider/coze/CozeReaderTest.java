package ai.open.right.workflow.flow.llm.provider.coze;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.provider.ProviderData;
import ai.open.right.workflow.flow.llm.provider.ProviderUtils;
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
public class CozeReaderTest {

    @Test
    public void test() throws Exception {
        String content = IOUtils.toString(new BufferedReader(new InputStreamReader(new ByteArrayInputStream("Hello".getBytes()))));
        CozeRequest req = EasyMock.createMock(CozeRequest.class);
        req.appendResponse(content);
        EasyMock.expectLastCall().anyTimes();
        LLMCallback cal = EasyMock.createMock(LLMCallback.class);
        cal.callback(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(req.getProviderData()).andReturn(new ProviderData()).anyTimes();
        EasyMock.expect(req.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(req.hasFunCall()).andReturn(false).anyTimes();
        EasyMock.expect(req.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.replay(req, cal);
        AtomicBoolean finished = new AtomicBoolean(false);
        StringBuilder builder = new StringBuilder();
        CozeReader reader = new CozeReader(ProviderReaderConfig.<CozeRequest>builder()
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
            // 追加消息
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
        Assert.assertEquals("Hello", builder.toString());
        EasyMock.verify(req, cal);
    }
}
