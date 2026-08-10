package ai.open.right.workflow.flow.llm.provider.openai;

import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import java.util.HashMap;
import java.util.Map;

public class OpenAiRequestTest {

    @Test
    public void testHashCode1() throws Exception {
        // 使用 JUnit 5 风格进行反射实例化测试
        Object object = OpenAiRequest.class.getConstructor().newInstance();
        Assertions.assertEquals(object.hashCode(), object.hashCode());
        Assertions.assertEquals(object, object);
    }

    @Test
    public void testGetResponseSchemaIsSameAsResponseFormat() {
        OpenAiRequest request = new OpenAiRequest();
        Assertions.assertNull(request.getResponseSchema());
        Map<String, Object> responseFormat = new HashMap<>();
        responseFormat.put("type", "json_object");
        request.setResponseFormat(responseFormat);
        Assertions.assertSame(request.getResponseFormat(), request.getResponseSchema());
        Assertions.assertEquals("json_object", request.getResponseSchema().get("type"));
    }

    @Test
    public void testGetterSetter() {
        OpenAiRequest request = new OpenAiRequest();

        // 测试 responseFormat
        Map<String, Object> responseFormat = new HashMap<>();
        responseFormat.put("type", "json_object");
        request.setResponseFormat(responseFormat);
        Assertions.assertEquals(responseFormat, request.getResponseFormat());
        Assertions.assertEquals(responseFormat, request.getResponseSchema());

        // 测试 extraBody
        Map<String, Object> extraBody = new HashMap<>();
        extraBody.put("extra", "data");
        request.setExtraBody(extraBody);
        Assertions.assertEquals(extraBody, request.getExtraBody());

        // 测试 openAiMedia
        OpenAiRequest.DefaultMedia media = new OpenAiRequest.DefaultMedia();
        request.setOpenAiMedia(media);
        Assertions.assertEquals(media, request.getOpenAiMedia());

        // 测试 frequencyPenalty
        request.setFrequencyPenalty(1.0);
        Assertions.assertEquals(Double.valueOf(1.0), request.getFrequencyPenalty());

        // 测试 presencePenalty
        request.setPresencePenalty(0.5);
        Assertions.assertEquals(Double.valueOf(0.5), request.getPresencePenalty());

        // 测试 temperature
        request.setTemperature(0.7);
        Assertions.assertEquals(Double.valueOf(0.7), request.getTemperature());

        // 测试 maxTokens
        request.setMaxTokens(1024);
        Assertions.assertEquals(Integer.valueOf(1024), request.getMaxTokens());

        // 测试 mimeType
        request.setMimeType("application/json");
        Assertions.assertEquals("application/json", request.getMimeType());

        // 测试 model
        request.setModel("gpt-4");
        Assertions.assertEquals("gpt-4", request.getModel());

        // 测试 topP
        request.setTopP(0.9);
        Assertions.assertEquals(Double.valueOf(0.9), request.getTopP());
    }
}

