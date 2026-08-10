package ai.open.right.workflow.flow.llm.provider.google;

import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class GoogleRequestTest {

    @Test
    public void testGetResponseSchemaNullWhenUnset() {
        GoogleRequest googleRequest = new GoogleRequest();
        Assertions.assertNull(googleRequest.getResponseSchema());
    }

    @Test
    public void test() {
        GoogleRequest googleRequest = new GoogleRequest();
        googleRequest.setMaxOutputTokens(1024);
        googleRequest.setMediaResolution("LOW");
        googleRequest.setResponseModalities(List.of("HELLO"));
        googleRequest.setTopK(1);
        googleRequest.setTopP(2.0);
        Assertions.assertEquals(Integer.valueOf(1), googleRequest.getTopK());
        Assertions.assertEquals(Double.valueOf(2.0), googleRequest.getTopP());
        Assertions.assertEquals(Integer.valueOf(1024), googleRequest.getMaxOutputTokens());
        
        Map<String, Object> schema = new HashMap<>();
        schema.put("A", "B");
        googleRequest.setResponseSchema(schema);
        
        Map<String, Object> imgConfig = new HashMap<>();
        imgConfig.put("C", "D");
        googleRequest.setImageConfig(imgConfig);
        
        Assertions.assertEquals("LOW", googleRequest.getMediaResolution());
        Assertions.assertEquals("D", googleRequest.getImageConfig().get("C"));
        Assertions.assertEquals("B", googleRequest.getResponseSchema().get("A"));
        Assertions.assertEquals("HELLO", googleRequest.getResponseModalities().get(0));
        Assertions.assertEquals("LOW", googleRequest.configs().get(GoogleRequestService.KEY_MEDIA_RESOLUTION));
        Assertions.assertEquals("application/json", googleRequest.configs().get(GoogleRequestService.KEY_RESPONSE_MIME_TYPE));
        Assertions.assertEquals("B", ((Map) googleRequest.configs().get(GoogleRequestService.KEY_RESPONSE_SCHEMA)).get("A"));
    }

    @Test
    public void testEmpty() {
        GoogleRequest googleRequest = new GoogleRequest();
        googleRequest.setTemperature(null);
        Assertions.assertNull(googleRequest.configs());
    }

    @Test
    public void testSetImageConfigMergesMutableMaps() {
        GoogleRequest googleRequest = new GoogleRequest();
        Map<String, Object> imageConfig = new HashMap<>();
        imageConfig.put("imageSize", "2K");
        googleRequest.setImageConfig(imageConfig);
        googleRequest.setImageConfig(Map.of("aspectRatio", "16:9"));

        Assertions.assertEquals("2K", googleRequest.getImageConfig().get("imageSize"));
        Assertions.assertEquals("16:9", googleRequest.getImageConfig().get("aspectRatio"));
    }

    @Test
    public void testGetRepositoriesSceneExists() {
        GoogleRequest req = new GoogleRequest();
        req.setScene("S");
        req.setRepositories(new ArrayList<>(Arrays.asList("S")));
        Assertions.assertEquals(1, req.getRepositories().size());
    }

    @Test
    public void testConfigsFull() {
        GoogleRequest req = new GoogleRequest();
        req.setThinkingConfig(Map.of("think", "true"));
        req.setFrequencyPenalty(1.1);
        req.setPresencePenalty(1.2);
        req.setMimeType("text/plain");
        req.setSeed(123);
        req.setTemperature(0.5);
        req.setSafetySettings(List.of(Map.of("category", "HATE")));
        req.setToolsConfig(Map.of("tool", "test"));

        Map<String, Object> configs = req.configs();
        Assertions.assertNotNull(configs);
        Assertions.assertEquals(Map.of("think", "true"), configs.get(GoogleRequestService.KEY_THINKING_CONFIG));
        Assertions.assertEquals(1.1, configs.get(GoogleRequestService.KEY_FREQUENCY_PENALTY));
        Assertions.assertEquals(1.2, configs.get(GoogleRequestService.KEY_PRESENCE_PENALTY));
        Assertions.assertEquals("text/plain", configs.get(GoogleRequestService.KEY_MIMETYPE));
        Assertions.assertEquals(123, configs.get(GoogleRequestService.KEY_SEED));
        Assertions.assertEquals(0.5, configs.get(GoogleRequestService.KEY_TEMPERATURE));
        
        Assertions.assertEquals(List.of(Map.of("category", "HATE")), req.getSafetySettings());
        Assertions.assertEquals(Map.of("tool", "test"), req.getToolsConfig());
    }

    @Test
    public void testConfigsWithOnlyOneField() {
        GoogleRequest req = new GoogleRequest();
        req.setTemperature(null);
        
        req.setMaxOutputTokens(100);
        Assertions.assertNotNull(req.configs());
        Assertions.assertEquals(100, req.configs().get(GoogleRequestService.KEY_MAX_OUTPUT_TOKENS));
        
        req.setMaxOutputTokens(null);
        req.setMediaResolution("HIGH");
        Assertions.assertNotNull(req.configs());
        Assertions.assertEquals("HIGH", req.configs().get(GoogleRequestService.KEY_MEDIA_RESOLUTION));
        
        req.setMediaResolution(null);
        req.setMimeType("image/png");
        Assertions.assertNotNull(req.configs());
        Assertions.assertEquals("image/png", req.configs().get(GoogleRequestService.KEY_MIMETYPE));
    }
}
