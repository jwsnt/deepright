package ai.open.right.workflow.flow.llm.provider.anthropic;

import ai.open.right.WorkflowException;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertSame;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

/**
 * Unit test for {@link AnthropicRequest}
 */
class AnthropicRequestTest {

    @Test
    void getResponseSchema_sameInstanceAsResponseFormat() {
        AnthropicRequest request = new AnthropicRequest();
        assertNull(request.getResponseSchema());
        Map<String, Object> responseFormat = new HashMap<>();
        responseFormat.put("type", "json_object");
        request.setResponseFormat(responseFormat);
        assertSame(request.getResponseFormat(), request.getResponseSchema());
    }

    @Test
    void testGettersAndSetters() {
        AnthropicRequest request = new AnthropicRequest();
        Map<String, Object> thinking = new HashMap<>();
        request.setThinking(thinking);
        Assertions.assertEquals(thinking, request.getThinking());

        // Test responseFormat
        Map<String, Object> responseFormat = new HashMap<>();
        responseFormat.put("type", "json_object");
        request.setResponseFormat(responseFormat);
        assertEquals(responseFormat, request.getResponseFormat());
        assertEquals(responseFormat, request.getResponseSchema());

        // Test extraBody
        Map<String, Object> extraBody = new HashMap<>();
        extraBody.put("key", "value");
        request.setExtraBody(extraBody);
        assertEquals(extraBody, request.getExtraBody());

        // Test anthropicMedia
        AnthropicMedia media = new AnthropicRequest.DefaultMedia();
        request.setAnthropicMedia(media);
        assertEquals(media, request.getAnthropicMedia());

        // Test temperature
        Double temperature = 0.7;
        request.setTemperature(temperature);
        assertEquals(temperature, request.getTemperature());

        // Test maxTokens
        Integer maxTokens = 1024;
        request.setMaxTokens(maxTokens);
        assertEquals(maxTokens, request.getMaxTokens());

        // Test mimeType
        String mimeType = "application/json";
        request.setMimeType(mimeType);
        assertEquals(mimeType, request.getMimeType());

        // Test model
        String model = "claude-3-opus-20240229";
        request.setModel(model);
        assertEquals(model, request.getModel());

        // Test topP
        Double topP = 0.9;
        request.setTopP(topP);
        assertEquals(topP, request.getTopP());
    }

    @Test
    void testEmptyConstant() {
        assertNotNull(AnthropicRequest.EMPTY);
        assertTrue(AnthropicRequest.EMPTY instanceof AnthropicRequest.DefaultMedia);
        
        AnthropicRequest request = new AnthropicRequest();
        // Verify default value
        assertEquals(AnthropicRequest.EMPTY, request.getAnthropicMedia());
    }

    /**
     * {@link AnthropicRequest.DefaultMedia#getMimes()}、{@code checkValid}、{@code trimType} 及 {@link AnthropicRequest.DefaultMedia#getType(String)}。
     */
    @Test
    void defaultMedia_getMimes_containsImageAndPdf() throws Exception {
        AnthropicRequest.DefaultMedia media = new AnthropicRequest.DefaultMedia();
        Set<String> mimes = media.getMimes();
        assertNotNull(mimes);
        assertTrue(mimes.contains("image/jpeg"));
        assertTrue(mimes.contains("image/png"));
        assertTrue(mimes.contains("image/bmp"));
        assertTrue(mimes.contains("application/pdf"));
        assertEquals(mimes, media.getMimes());
    }

    @Test
    void defaultMedia_trimType_stripsInlinePrefix() throws Exception {
        AnthropicRequest.DefaultMedia media = new AnthropicRequest.DefaultMedia();
        assertEquals("image/png", media.trimType("inline:image/png"));
        assertEquals("application/pdf", media.trimType("inline:application/pdf"));
        assertEquals("image/jpeg", media.trimType("image/jpeg"));
    }

    @Test
    void defaultMedia_checkValid_acceptsWhitelistedMimeAndInline() throws Exception {
        AnthropicRequest.DefaultMedia media = new AnthropicRequest.DefaultMedia();
        media.checkValid("image/png");
        media.checkValid("image/jpeg");
        media.checkValid("image/bmp");
        media.checkValid("application/pdf");
        media.checkValid("inline:image/png");
    }

    @Test
    void defaultMedia_checkValid_rejectsUnknownMime() {
        AnthropicRequest.DefaultMedia media = new AnthropicRequest.DefaultMedia();
        WorkflowException ex = assertThrows(WorkflowException.class, () -> media.checkValid("video/mp4"));
        assertTrue(ex.getMessage().contains("The mime type is invalid"));
        assertTrue(ex.getMessage().contains("video/mp4"));
    }

    @Test
    void defaultMedia_getType_imageMime_returnsImage() throws Exception {
        AnthropicRequest.DefaultMedia media = new AnthropicRequest.DefaultMedia();
        assertEquals("image", media.getType("image/png"));
        assertEquals("image", media.getType("inline:image/jpeg"));
    }

    @Test
    void defaultMedia_getType_pdf_returnsDocument() throws Exception {
        AnthropicRequest.DefaultMedia media = new AnthropicRequest.DefaultMedia();
        assertEquals("document", media.getType("application/pdf"));
        assertEquals("document", media.getType("inline:application/pdf"));
    }
}

