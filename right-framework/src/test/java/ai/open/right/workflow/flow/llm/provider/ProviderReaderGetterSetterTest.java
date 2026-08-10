package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.ObjectBuilder;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import org.junit.jupiter.api.Test;

import java.nio.ByteBuffer;
import java.util.HashMap;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertSame;

/**
 * 覆盖 {@link ProviderReader} 上 Lombok {@code @Getter} / {@code @Setter} 生成的方法。
 */
class ProviderReaderGetterSetterTest {

    private static ProviderReader<ProviderRequest> newReader() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.setStream(false);
        return new ProviderReader<>(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(m -> { })
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build());
    }

    @Test
    void getters_afterConstruction_referenceFieldsAndDefaults() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.setStream(false);
        ProviderReader<ProviderRequest> reader = new ProviderReader<>(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(m -> { })
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build());

        assertSame(req, reader.getRequest());
        assertNotNull(reader.getProviderReaderCallback());
        assertNotNull(reader.getEventListenerService());
        assertNotNull(reader.getMessageQueue());
        assertNotNull(reader.getByteBuffer());
        assertEquals(1024, reader.getByteBuffer().capacity());
        assertNull(reader.getExpMessage());
        assertEquals(0, reader.getMessageCnt());
        assertEquals(0, reader.getChunkIndex());
        assertEquals(ProtocolCode.C200, reader.getCode());
    }

    @Test
    void setters_roundTrip_mutableState() throws Exception {
        ProviderReader<ProviderRequest> reader = newReader();

        StringBuffer exp = new StringBuffer("err");
        reader.setExpMessage(exp);
        assertSame(exp, reader.getExpMessage());

        ByteBuffer buf = ByteBuffer.allocate(64);
        reader.setByteBuffer(buf);
        assertSame(buf, reader.getByteBuffer());

        reader.setMessageCnt(11);
        assertEquals(11, reader.getMessageCnt());

        reader.setChunkIndex(22);
        assertEquals(22, reader.getChunkIndex());

        reader.setCode(503);
        assertEquals(503, reader.getCode());
    }

    @Test
    void staticConstants_extensionAndDone() {
        assertNotNull(ProviderReader.EXTENSION);
        assertEquals(0, ProviderReader.EXTENSION.size());
        assertEquals("data: [DONE]", ProviderReader.DONE);
    }
}
