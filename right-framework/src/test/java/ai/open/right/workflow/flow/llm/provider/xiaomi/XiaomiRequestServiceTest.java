package ai.open.right.workflow.flow.llm.provider.xiaomi;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.test.util.ReflectionTestUtils;

import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@DisplayName("XiaomiRequestService 单元测试")
class XiaomiRequestServiceTest {

    private XiaomiRequestService xiaomiRequestService;

    @Mock
    private WorkflowTask mockWorkflowTask;

    private static final String TEST_MODEL = "mimo-v2-flash";
    private static final String TEST_TOKEN = "test-api-token-12345";
    private static final String METADATA_MODEL = "mimo-v2-pro";
    private static final String METADATA_TOKEN = "metadata-token-67890";

    @BeforeEach
    void setUp() {
        xiaomiRequestService = new XiaomiRequestService();
        // 使用反射设置内部字段
        ReflectionTestUtils.setField(xiaomiRequestService, "model", TEST_MODEL);
        ReflectionTestUtils.setField(xiaomiRequestService, "token", TEST_TOKEN);
    }

    // ========================================
    // defToken() 方法测试 - 覆盖所有分支
    // ========================================

    @Test
    @DisplayName("defToken: metadata中存在__token时，应返回metadata中的token")
    void testDefToken_MetadataTokenExists_ShouldReturnMetadataToken() throws Exception {
        // Given
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("__token", METADATA_TOKEN);
        
        when(mockWorkflowTask.getMetadata()).thenReturn(metadata);
        
        // When
        String result = ReflectionTestUtils.invokeMethod(xiaomiRequestService, "defToken", mockWorkflowTask);
        
        // Then
        assertEquals(METADATA_TOKEN, result);
    }

    @Test
    @DisplayName("defToken: metadata中__token为空字符串时，应返回默认token")
    void testDefToken_MetadataTokenEmpty_ShouldReturnDefaultToken() throws Exception {
        // Given
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("__token", "");
        
        when(mockWorkflowTask.getMetadata()).thenReturn(metadata);
        
        // When
        String result = ReflectionTestUtils.invokeMethod(xiaomiRequestService, "defToken", mockWorkflowTask);
        
        // Then
        assertEquals(TEST_TOKEN, result);
    }

    @Test
    @DisplayName("defToken: metadata中__token为null时，应返回默认token")
    void testDefToken_MetadataTokenNull_ShouldReturnDefaultToken() throws Exception {
        // Given
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("__token", null);
        
        when(mockWorkflowTask.getMetadata()).thenReturn(metadata);
        
        // When
        String result = ReflectionTestUtils.invokeMethod(xiaomiRequestService, "defToken", mockWorkflowTask);
        
        // Then
        assertEquals(TEST_TOKEN, result);
    }

    @Test
    @DisplayName("defToken: metadata中不存在__token键时，应返回默认token")
    void testDefToken_MetadataTokenKeyMissing_ShouldReturnDefaultToken() throws Exception {
        // Given
        Map<String, Object> metadata = new HashMap<>();
        // 不添加__token键
        
        when(mockWorkflowTask.getMetadata()).thenReturn(metadata);
        
        // When
        String result = ReflectionTestUtils.invokeMethod(xiaomiRequestService, "defToken", mockWorkflowTask);
        
        // Then
        assertEquals(TEST_TOKEN, result);
    }

    @Test
    @DisplayName("defToken: metadata为null时，应返回默认token")
    void testDefToken_MetadataNull_ShouldReturnDefaultToken() throws Exception {
        // Given
        when(mockWorkflowTask.getMetadata()).thenReturn(null);
        
        // When
        String result = ReflectionTestUtils.invokeMethod(xiaomiRequestService, "defToken", mockWorkflowTask);
        
        // Then
        assertEquals(TEST_TOKEN, result);
    }

    @Test
    @DisplayName("defToken: 默认token为空且metadata中无__token时，应返回null")
    void testDefToken_DefaultTokenEmptyAndNoMetadataToken_ShouldReturnNull() throws Exception {
        // Given
        ReflectionTestUtils.setField(xiaomiRequestService, "token", null);
        
        Map<String, Object> metadata = new HashMap<>();
        when(mockWorkflowTask.getMetadata()).thenReturn(metadata);
        
        // When
        String result = ReflectionTestUtils.invokeMethod(xiaomiRequestService, "defToken", mockWorkflowTask);
        
        // Then
        assertNull(result);
    }

    @Test
    @DisplayName("defToken: 默认token为空字符串且metadata中无__token时，应返回空字符串")
    void testDefToken_DefaultTokenEmptyStringAndNoMetadataToken_ShouldReturnEmpty() throws Exception {
        // Given
        ReflectionTestUtils.setField(xiaomiRequestService, "token", "");
        
        Map<String, Object> metadata = new HashMap<>();
        when(mockWorkflowTask.getMetadata()).thenReturn(metadata);
        
        // When
        String result = ReflectionTestUtils.invokeMethod(xiaomiRequestService, "defToken", mockWorkflowTask);
        
        // Then
        assertEquals("", result);
    }

    // ========================================
    // defModel() 方法测试 - 覆盖所有分支
    // ========================================

    @Test
    @DisplayName("defModel: metadata中存在__model时，应返回metadata中的model")
    void testDefModel_MetadataModelExists_ShouldReturnMetadataModel() throws Exception {
        // Given
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("__model", METADATA_MODEL);
        
        when(mockWorkflowTask.getMetadata()).thenReturn(metadata);
        
        // When
        String result = ReflectionTestUtils.invokeMethod(xiaomiRequestService, "defModel", mockWorkflowTask);
        
        // Then
        assertEquals(METADATA_MODEL, result);
    }

    @Test
    @DisplayName("defModel: metadata中__model为空字符串时，应返回默认model")
    void testDefModel_MetadataModelEmpty_ShouldReturnDefaultModel() throws Exception {
        // Given
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("__model", "");
        
        when(mockWorkflowTask.getMetadata()).thenReturn(metadata);
        
        // When
        String result = ReflectionTestUtils.invokeMethod(xiaomiRequestService, "defModel", mockWorkflowTask);
        
        // Then
        assertEquals(TEST_MODEL, result);
    }

    @Test
    @DisplayName("defModel: metadata中__model为null时，应返回默认model")
    void testDefModel_MetadataModelNull_ShouldReturnDefaultModel() throws Exception {
        // Given
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("__model", null);
        
        when(mockWorkflowTask.getMetadata()).thenReturn(metadata);
        
        // When
        String result = ReflectionTestUtils.invokeMethod(xiaomiRequestService, "defModel", mockWorkflowTask);
        
        // Then
        assertEquals(TEST_MODEL, result);
    }

    @Test
    @DisplayName("defModel: metadata中不存在__model键时，应返回默认model")
    void testDefModel_MetadataModelKeyMissing_ShouldReturnDefaultModel() throws Exception {
        // Given
        Map<String, Object> metadata = new HashMap<>();
        // 不添加__model键
        
        when(mockWorkflowTask.getMetadata()).thenReturn(metadata);
        
        // When
        String result = ReflectionTestUtils.invokeMethod(xiaomiRequestService, "defModel", mockWorkflowTask);
        
        // Then
        assertEquals(TEST_MODEL, result);
    }

    @Test
    @DisplayName("defModel: metadata为null时，应返回默认model")
    void testDefModel_MetadataNull_ShouldReturnDefaultModel() throws Exception {
        // Given
        when(mockWorkflowTask.getMetadata()).thenReturn(null);
        
        // When
        String result = ReflectionTestUtils.invokeMethod(xiaomiRequestService, "defModel", mockWorkflowTask);
        
        // Then
        assertEquals(TEST_MODEL, result);
    }

    @Test
    @DisplayName("defModel: 所有来源都为空时，应抛出IllegalArgumentException")
    void testDefModel_AllSourcesEmpty_ShouldThrowException() throws Exception {
        // Given
        ReflectionTestUtils.setField(xiaomiRequestService, "model", null);
        
        Map<String, Object> metadata = new HashMap<>();
        when(mockWorkflowTask.getMetadata()).thenReturn(metadata);
        
        // When & Then
        IllegalArgumentException exception = assertThrows(
                IllegalArgumentException.class,
                () -> ReflectionTestUtils.invokeMethod(xiaomiRequestService, "defModel", mockWorkflowTask)
        );
        assertTrue(exception.getMessage().contains("model can not be empty"));
    }

    @Test
    @DisplayName("defModel: 默认model为空字符串时，应抛出IllegalArgumentException")
    void testDefModel_DefaultModelEmptyString_ShouldThrowException() throws Exception {
        // Given
        ReflectionTestUtils.setField(xiaomiRequestService, "model", "");
        
        Map<String, Object> metadata = new HashMap<>();
        when(mockWorkflowTask.getMetadata()).thenReturn(metadata);
        
        // When & Then
        assertThrows(
                IllegalArgumentException.class,
                () -> ReflectionTestUtils.invokeMethod(xiaomiRequestService, "defModel", mockWorkflowTask)
        );
    }

    @Test
    @DisplayName("defModel: 默认model为空白字符串时，应抛出IllegalArgumentException")
    void testDefModel_DefaultModelBlankString_ShouldThrowException() throws Exception {
        // Given
        ReflectionTestUtils.setField(xiaomiRequestService, "model", "   ");
        
        Map<String, Object> metadata = new HashMap<>();
        when(mockWorkflowTask.getMetadata()).thenReturn(metadata);
        
        // When & Then
        assertThrows(
                IllegalArgumentException.class,
                () -> ReflectionTestUtils.invokeMethod(xiaomiRequestService, "defModel", mockWorkflowTask)
        );
    }

    // ========================================
    // getModel() 方法测试
    // ========================================

    @Test
    @DisplayName("getModel: 应返回当前设置的model")
    void testGetModel_ShouldReturnCurrentModel() throws Exception {
        // When
        String result = xiaomiRequestService.getModel(mockWorkflowTask);
        
        // Then
        assertEquals(TEST_MODEL, result);
    }

    @Test
    @DisplayName("getModel: model为null时，应返回null")
    void testGetModel_ModelNull_ShouldReturnNull() throws Exception {
        // Given
        ReflectionTestUtils.setField(xiaomiRequestService, "model", null);
        
        // When
        String result = xiaomiRequestService.getModel(mockWorkflowTask);
        
        // Then
        assertNull(result);
    }

    // ========================================
    // reasoning() 方法测试 - 覆盖所有分支
    // ========================================

    @Test
    @DisplayName("reasoning: 请求metadata中的thinking非空时，应优先写入请求配置")
    void testReasoning_QueryThinkingExists_ShouldPreferQueryThinking() throws Exception {
        // Given
        Map<String, Object> queryThinking = new HashMap<>();
        queryThinking.put("type", "enabled");
        Map<String, Object> metadata = new HashMap<>();
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING, queryThinking);
        LLMConfig llmConfig = new LLMConfig();
        Map<String, Object> configThinking = new HashMap<>();
        configThinking.put("type", "disabled");
        llmConfig.getAdditional().put(ProviderRequestService.KEY_THINKING, configThinking);
        OpenAiRequest request = new OpenAiRequest();

        // When
        xiaomiRequestService.reasoning(request, llmConfig, ObjectBuilder.buildLLMQuery(metadata));

        // Then
        assertNotNull(request.getExtraBody());
        assertSame(queryThinking, request.getExtraBody().get(ProviderRequestService.KEY_THINKING));
    }

    @Test
    @DisplayName("reasoning: 请求metadata中的thinking为空时，应回退到模型配置")
    void testReasoning_QueryThinkingEmpty_ShouldFallbackToConfigThinking() throws Exception {
        // Given
        Map<String, Object> metadata = new HashMap<>();
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING, new HashMap<>());
        LLMConfig llmConfig = new LLMConfig();
        Map<String, Object> configThinking = new HashMap<>();
        configThinking.put("type", "enabled");
        llmConfig.getAdditional().put(ProviderRequestService.KEY_THINKING, configThinking);
        OpenAiRequest request = new OpenAiRequest();

        // When
        xiaomiRequestService.reasoning(request, llmConfig, ObjectBuilder.buildLLMQuery(metadata));

        // Then
        assertNotNull(request.getExtraBody());
        assertSame(configThinking, request.getExtraBody().get(ProviderRequestService.KEY_THINKING));
    }

    @Test
    @DisplayName("reasoning: 请求metadata和模型配置均无thinking时，不应写入请求参数")
    void testReasoning_NoThinkingConfigured_ShouldNotSetExtra() throws Exception {
        // Given
        LLMConfig llmConfig = new LLMConfig();
        OpenAiRequest request = new OpenAiRequest();

        // When
        xiaomiRequestService.reasoning(request, llmConfig, ObjectBuilder.buildLLMQueryWithEmptyMetadata());

        // Then
        assertNull(request.getExtraBody());
    }

    // ========================================
    // Getter/Setter 测试
    // ========================================

    @Test
    @DisplayName("model getter/setter测试")
    void testModelGetterSetter() {
        // Given
        String newModel = "mimo-v2-ultra";
        
        // When
        xiaomiRequestService.setModel(newModel);
        
        // Then
        assertEquals(newModel, xiaomiRequestService.getModel());
    }

    @Test
    @DisplayName("token getter/setter测试")
    void testTokenGetterSetter() {
        // Given
        String newToken = "new-api-token-99999";
        
        // When
        xiaomiRequestService.setToken(newToken);
        
        // Then
        assertEquals(newToken, xiaomiRequestService.getToken());
    }

    @Test
    @DisplayName("NAME常量测试")
    void testNameConstant() {
        // Given & When & Then
        assertEquals("XiaomiRequestService", XiaomiRequestService.NAME);
    }

    // ========================================
    // 组合场景测试
    // ========================================

    @Test
    @DisplayName("metadata中同时包含__model和__token时，应分别返回对应的值")
    void testBothModelAndTokenInMetadata_ShouldReturnCorrectValues() throws Exception {
        // Given
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("__model", METADATA_MODEL);
        metadata.put("__token", METADATA_TOKEN);
        
        when(mockWorkflowTask.getMetadata()).thenReturn(metadata);
        
        // When
        String modelResult = ReflectionTestUtils.invokeMethod(xiaomiRequestService, "defModel", mockWorkflowTask);
        String tokenResult = ReflectionTestUtils.invokeMethod(xiaomiRequestService, "defToken", mockWorkflowTask);
        
        // Then
        assertEquals(METADATA_MODEL, modelResult);
        assertEquals(METADATA_TOKEN, tokenResult);
    }

    @Test
    @DisplayName("metadata中包含其他key但无__model和__token时，应使用默认值")
    void testMetadataContainsOtherKeys_ShouldUseDefaults() throws Exception {
        // Given
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("other_key", "other_value");
        metadata.put("another_key", 123);
        
        when(mockWorkflowTask.getMetadata()).thenReturn(metadata);
        
        // When
        String modelResult = ReflectionTestUtils.invokeMethod(xiaomiRequestService, "defModel", mockWorkflowTask);
        String tokenResult = ReflectionTestUtils.invokeMethod(xiaomiRequestService, "defToken", mockWorkflowTask);
        
        // Then
        assertEquals(TEST_MODEL, modelResult);
        assertEquals(TEST_TOKEN, tokenResult);
    }
}
