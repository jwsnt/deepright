package ai.open.right.workflow.flow.llm.provider.seedream;

import ai.open.right.ObjectBuilder;
import ai.open.right.listener.EventListenerService;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.notify.NotifierService;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Test;

import java.util.HashMap;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;

import ai.open.right.workflow.flow.llm.provider.ProviderReaderConfig;
/**
 * AnthropicReader 单元测试
 */
class SeedreamReaderTest {

    @Test
    void testConstructor() throws Exception {
        // 准备 Mock 对象和参数
        SeedreamRequest request = EasyMock.createMock(SeedreamRequest.class);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();

        NotifierService notifierService = ObjectBuilder.buildNotifierManagerWithimplement();
        EventListenerService eventListenerService = ObjectBuilder.buildEventListenerService();
        Integer timeout = 1024;
        Integer discard = 2048;
        Integer buffer = 2048;
        Integer queue = 100;

        EasyMock.replay(request, callback);

        // 执行构造函数
        SeedreamReader reader = new SeedreamReader(ProviderReaderConfig.<SeedreamRequest>builder()
                .request(request)
                .llmCallback(callback)
                .notifierService(notifierService)
                .eventListenerService(eventListenerService)
                .extension(new HashMap<>())
                .discard(discard)
                .timeout(timeout)
                .buffer(buffer)
                .capacity(buffer)
                .queue(queue)
                .build());

        // 验证初始化结果
        assertNotNull(reader, "Reader should not be null");
        assertEquals(request, reader.getRequest(), "Request should be correctly set");
        assertEquals(callback, reader.getProviderReaderCallback().getLlmCallback(), "LLMCallback should be correctly set");
        assertEquals(notifierService, reader.getProviderReaderCallback().getNotifierService(), "NotifierService should be correctly set");
        assertEquals(eventListenerService, reader.getEventListenerService(), "EventListenerService should be correctly set");
        assertEquals(buffer.intValue(), reader.getByteBuffer().capacity(), "ByteBuffer capacity should match buffer size");

        EasyMock.verify(request, callback);
    }
}

