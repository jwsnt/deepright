package ai.open.right.workflow.flow.llm.provider.xiaomi;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

@DisplayName("XiaomiRouter 单元测试")
class XiaomiRouterTest {

    private XiaomiRouter xiaomiRouter;

    private static final String TEST_URL = "https://api.xiaomimimo.com/v1";
    private static final String REQUEST_URL = "https://request.api.com/v1";
    private static final String METADATA_URL = "https://metadata.api.com/v1";

    @BeforeEach
    void setUp() {
        xiaomiRouter = new XiaomiRouter();
        xiaomiRouter.setUrl(TEST_URL);
    }

    // ========================================
    // 正常路径测试
    // ========================================

    @Test
    @DisplayName("URL优先级1: metadata中__url存在时，应返回metadata中的url")
    void testUrl_MetadataUrlExists_ShouldReturnMetadataUrl() throws Exception {
        // Given
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("__url", METADATA_URL);
        OpenAiRequest request = buildRequest(metadata, REQUEST_URL);

        // When
        String result = xiaomiRouter.url(request, null, "test");

        // Then
        assertEquals(METADATA_URL, result);
    }

    @Test
    @DisplayName("URL优先级2: metadata中__url为空字符串时，应抛出IllegalArgumentException")
    void testUrl_MetadataUrlEmpty_ShouldThrowException() {
        // Given
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("__url", "");
        OpenAiRequest request = buildRequest(metadata, REQUEST_URL);

        // When & Then
        IllegalArgumentException exception = assertThrows(
                IllegalArgumentException.class,
                () -> xiaomiRouter.url(request, null, "test")
        );
        assertEquals("Url can not be empty", exception.getMessage());
    }

    @Test
    @DisplayName("URL优先级3: metadata中__url为null且request.getUrl()不为空时，应返回request.getUrl()")
    void testUrl_MetadataUrlNullAndRequestUrlExists_ShouldReturnRequestUrl() throws Exception {
        // Given
        Map<String, Object> metadata = new HashMap<>();
        // __url不存在
        OpenAiRequest request = buildRequest(metadata, REQUEST_URL);

        // When
        String result = xiaomiRouter.url(request, null, "test");

        // Then
        assertEquals(REQUEST_URL, result);
    }

    @Test
    @DisplayName("URL优先级4: metadata中__url为空且request.getUrl()为空时，应返回默认url")
    void testUrl_BothUrlsEmpty_ShouldReturnDefaultUrl() throws Exception {
        // Given
        Map<String, Object> metadata = new HashMap<>();
        OpenAiRequest request = buildRequest(metadata, null);

        // When
        String result = xiaomiRouter.url(request, null, "test");

        // Then
        assertEquals(TEST_URL, result);
    }

    @Test
    @DisplayName("URL优先级4: request.getUrl()返回空字符串时，应返回默认url")
    void testUrl_RequestUrlEmptyString_ShouldReturnDefaultUrl() throws Exception {
        // Given
        Map<String, Object> metadata = new HashMap<>();
        OpenAiRequest request = buildRequest(metadata, "");

        // When
        String result = xiaomiRouter.url(request, null, "test");

        // Then
        assertEquals(TEST_URL, result);
    }

    // ========================================
    // 异常路径测试
    // ========================================

    @Test
    @DisplayName("所有URL来源都为空时，应抛出IllegalArgumentException")
    void testUrl_AllUrlsEmpty_ShouldThrowException() throws Exception {
        // Given
        Map<String, Object> metadata = new HashMap<>();
        OpenAiRequest request = buildRequest(metadata, null);

        xiaomiRouter.setUrl(null); // 默认url也为空

        // When & Then
        IllegalArgumentException exception = assertThrows(
                IllegalArgumentException.class,
                () -> xiaomiRouter.url(request, null, "test")
        );
        assertTrue(exception.getMessage().contains("Url can not be empty"));
    }

    @Test
    @DisplayName("默认url为空字符串时，应抛出IllegalArgumentException")
    void testUrl_DefaultUrlEmptyString_ShouldThrowException() throws Exception {
        // Given
        Map<String, Object> metadata = new HashMap<>();
        OpenAiRequest request = buildRequest(metadata, null);

        xiaomiRouter.setUrl(""); // 默认url为空字符串

        // When & Then
        IllegalArgumentException exception = assertThrows(
                IllegalArgumentException.class,
                () -> xiaomiRouter.url(request, null, "test")
        );
        assertTrue(exception.getMessage().contains("Url can not be empty"));
    }

    // ========================================
    // Getter/Setter测试
    // ========================================

    @Test
    @DisplayName("URL getter/setter测试")
    void testUrlGetterSetter() {
        // Given
        String newUrl = "https://new.api.com/v1";
        
        // When
        xiaomiRouter.setUrl(newUrl);
        
        // Then
        assertEquals(newUrl, xiaomiRouter.getUrl());
    }

    @Test
    @DisplayName("NAME常量测试")
    void testNameConstant() {
        // Given & When & Then
        assertEquals("XiaomiRouter", XiaomiRouter.NAME);
    }

    // ========================================
    // 边界条件测试
    // ========================================

    @Test
    @DisplayName("metadata中包含多个key时，应正确提取__url")
    void testUrl_MultipleMetadataKeys_ShouldExtractCorrectUrl() throws Exception {
        // Given
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("__url", METADATA_URL);
        metadata.put("other_key", "other_value");
        metadata.put("another_key", 123);
        OpenAiRequest request = buildRequest(metadata, null);

        // When
        String result = xiaomiRouter.url(request, null, "test");

        // Then
        assertEquals(METADATA_URL, result);
    }

    @Test
    @DisplayName("metadata中__url为null时，应使用request.getUrl()")
    void testUrl_MetadataUrlNull_ShouldUseRequestUrl() throws Exception {
        // Given
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("__url", null);
        OpenAiRequest request = buildRequest(metadata, REQUEST_URL);

        // When
        String result = xiaomiRouter.url(request, null, "test");

        // Then
        assertEquals(REQUEST_URL, result);
    }

    private OpenAiRequest buildRequest(Map<String, Object> metadata, String requestUrl) {
        OpenAiRequest request = new OpenAiRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery(metadata));
        request.setMessage(message);
        request.setUrl(requestUrl);
        return request;
    }
}
