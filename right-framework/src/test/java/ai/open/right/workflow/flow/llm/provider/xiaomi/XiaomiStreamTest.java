package ai.open.right.workflow.flow.llm.provider.xiaomi;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import org.easymock.EasyMock;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

@DisplayName("XiaomiStream 单元测试")
class XiaomiStreamTest {

    private XiaomiStream xiaomiStream;

    @BeforeEach
    void setUp() throws Exception {
        xiaomiStream = new XiaomiStream(buildProviderStreamConfig());
    }

    @Test
    @DisplayName("stream: source以': PROCESSING'开头时应返回false")
    void testStream_SourceStartsWithProcessing_ShouldReturnFalse() throws Exception {
        assertFalse(xiaomiStream.stream(": PROCESSING some data"));
    }

    @Test
    @DisplayName("stream: source以': processing'小写开头时应返回false")
    void testStream_SourceStartsWithProcessingLowerCase_ShouldReturnFalse() throws Exception {
        assertFalse(xiaomiStream.stream(": processing status update"));
    }

    @Test
    @DisplayName("stream: source以': Processing'混合大小写开头时应返回false")
    void testStream_SourceStartsWithProcessingMixedCase_ShouldReturnFalse() throws Exception {
        assertFalse(xiaomiStream.stream(": Processing mixed case"));
    }

    @Test
    @DisplayName("stream: source仅为': PROCESSING'时应返回false")
    void testStream_SourceStartsWithProcessingNoSpace_ShouldReturnFalse() throws Exception {
        assertFalse(xiaomiStream.stream(": PROCESSING"));
    }

    @Test
    @DisplayName("stream: source以': PROCESSING'开头但后面有特殊字符时应返回false")
    void testStream_SourceStartsWithProcessingThenSpecialChars_ShouldReturnFalse() throws Exception {
        assertFalse(xiaomiStream.stream(": PROCESSING\n\r\t"));
    }

    @Test
    @DisplayName("stream: 合法OpenAI流式数据且未完成时应返回false")
    void testStream_SourceNormalData_ShouldReturnFalse() throws Exception {
        String source = "data: {\"id\":\"test\",\"choices\":[{\"delta\":{\"content\":\"hi\"},\"finish_reason\":null}]}";

        assertFalse(xiaomiStream.stream(source));
    }

    @Test
    @DisplayName("stream: source以其他前缀开头时应抛出IllegalArgumentException")
    void testStream_SourceWithOtherPrefix_ShouldThrowException() {
        IllegalArgumentException exception = assertThrows(
                IllegalArgumentException.class,
                () -> xiaomiStream.stream("event: message")
        );
        assertEquals("Invalid data", exception.getMessage());
    }

    @Test
    @DisplayName("stream: source为空字符串时应抛出IllegalArgumentException")
    void testStream_SourceEmpty_ShouldThrowException() {
        IllegalArgumentException exception = assertThrows(
                IllegalArgumentException.class,
                () -> xiaomiStream.stream("")
        );
        assertEquals("Invalid data", exception.getMessage());
    }

    @Test
    @DisplayName("stream: source为null时应抛出IllegalArgumentException")
    void testStream_SourceNull_ShouldThrowException() {
        IllegalArgumentException exception = assertThrows(
                IllegalArgumentException.class,
                () -> xiaomiStream.stream(null)
        );
        assertEquals("Invalid data", exception.getMessage());
    }

    @Test
    @DisplayName("stream: source包含PROCESSING但不在开头时应抛出IllegalArgumentException")
    void testStream_SourceContainsProcessingNotAtStart_ShouldThrowException() {
        IllegalArgumentException exception = assertThrows(
                IllegalArgumentException.class,
                () -> xiaomiStream.stream("some data : PROCESSING in middle")
        );
        assertEquals("Invalid data", exception.getMessage());
    }

    @Test
    @DisplayName("构造函数: 应正确初始化XiaomiStream")
    void testConstructor_ShouldInitializeCorrectly() throws Exception {
        XiaomiStream stream = new XiaomiStream(buildProviderStreamConfig());

        assertNotNull(stream);
    }

    @Test
    @DisplayName("构造函数: 应继承OpenAiStream的所有功能")
    void testConstructor_ShouldInheritOpenAiStream() throws Exception {
        XiaomiStream stream = new XiaomiStream(buildProviderStreamConfig());

        assertTrue(stream instanceof ai.open.right.workflow.flow.llm.provider.openai.OpenAiStream);
    }

    private ProviderStreamConfig<OpenAiRequest> buildProviderStreamConfig() throws Exception {
        HistoryStore historyStore = ObjectBuilder.buildMockHistoryWithNothing();
        EasyMock.replay(historyStore);

        OpenAiRequest request = new OpenAiRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        request.setTokenFirst(1024);
        request.setTokenBuffer(1024);
        request.setStream(true);

        return ProviderStreamConfig.<OpenAiRequest>builder()
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .providerReason(ObjectBuilder.getProviderReason())
                .historyStore(historyStore)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build();
    }
}
