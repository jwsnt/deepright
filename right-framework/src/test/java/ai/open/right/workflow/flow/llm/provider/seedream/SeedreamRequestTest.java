package ai.open.right.workflow.flow.llm.provider.seedream;

import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit test for {@link SeedreamRequest}
 */
class SeedreamRequestTest {

    @Test
    void getResponseSchema_inheritsDefaultNull() {
        assertNull(new SeedreamRequest().getResponseSchema());
    }

    @Test
    void testGettersAndSetters() {
        SeedreamRequest request = new SeedreamRequest();
        request.setFormat("URL");
        assertEquals("URL", request.getFormat());
        // Test SeedMedia
        SeedreamMedia media = new SeedreamRequest.DefaultMedia();
        request.setSeedMedia(media);
        assertEquals(media, request.getSeedMedia());
        // Test mimeType
        String mimeType = "application/json";
        request.setMimeType(mimeType);
        assertEquals(mimeType, request.getMimeType());
        // Test model
        String model = "claude-3-opus-20240229";
        request.setModel(model);
        assertEquals(model, request.getModel());
    }

    @Test
    void testDefaultMedia() {
        SeedreamRequest.DefaultMedia defaultMedia = new SeedreamRequest.DefaultMedia();
        // Test getPrefix
        String type = "image/jpeg";
        String expectedPrefix = "data:image/jpeg;base64,";
        assertEquals(expectedPrefix, defaultMedia.getPrefix(type));

        // Test getKeyUrl
        assertEquals("", defaultMedia.getKeyUrl(type));
    }

    @Test
    void testEmptyConstant() {
        assertNotNull(SeedreamRequest.EMPTY);
        assertTrue(SeedreamRequest.EMPTY instanceof SeedreamRequest.DefaultMedia);
        SeedreamRequest request = new SeedreamRequest();
        // Verify default value
        assertEquals(SeedreamRequest.EMPTY, request.getSeedMedia());
    }

    @Test
    void testGettersAndSetters2() {
        SeedreamRequest request = new SeedreamRequest();

        // 准备测试数据
        Map<String, Object> seqOpts = new HashMap<>();
        seqOpts.put("key1", "val1");

        Map<String, Object> optOpts = new HashMap<>();
        optOpts.put("key2", 100);

        Double guidance = 7.5;
        Integer images = 4;
        Integer seed = 12345;

        // 执行 Setter
        request.setSequentialOptions(seqOpts);
        request.setOptimizeOptions(optOpts);
        request.setSequential("seq_value");
        request.setGuidance(guidance);
        request.setMimeType("image/png");
        request.setImages(images);
        request.setFormat("webp");
        request.setSeed(seed);
        request.setModel("gen-model-v1");
        request.setSize("1024x1024");

        // 断言验证 Getter
        assertAll("Verify all fields",
                () -> assertEquals(seqOpts, request.getSequentialOptions()),
                () -> assertEquals(optOpts, request.getOptimizeOptions()),
                () -> assertEquals("seq_value", request.getSequential()),
                () -> assertEquals(guidance, request.getGuidance()),
                () -> assertEquals("image/png", request.getMimeType()),
                () -> assertEquals(images, request.getImages()),
                () -> assertEquals("webp", request.getFormat()),
                () -> assertEquals(seed, request.getSeed()),
                () -> assertEquals("gen-model-v1", request.getModel()),
                () -> assertEquals("1024x1024", request.getSize())
        );
    }

    @Test
    void testSeedMediaAndStaticConstants() {
        SeedreamRequest request = new SeedreamRequest();

        // 验证默认初始化
        assertNotNull(SeedreamRequest.EMPTY, "静态常量 EMPTY 不应为空");
        assertEquals(SeedreamRequest.EMPTY, request.getSeedMedia(), "默认 seedMedia 应指向 EMPTY");

        // 验证自定义 Setter
        SeedreamRequest.DefaultMedia customMedia = new SeedreamRequest.DefaultMedia();
        request.setSeedMedia(customMedia);
        assertEquals(customMedia, request.getSeedMedia());
    }

    @Test
    void testDefaultMediaMethods() {
        SeedreamRequest.DefaultMedia media = new SeedreamRequest.DefaultMedia();

        // 测试 getPrefix
        String prefix = media.getPrefix("image/jpeg");
        assertEquals("data:image/jpeg;base64,", prefix);

        // 测试 getKeyUrl
        String keyUrl = media.getKeyUrl("anyType");
        assertEquals("", keyUrl);
    }

    @Test
    void getImageConfig_returnsEmptyMapWhenUnset() {
        assertEquals(Map.of(), new SeedreamRequest().getImageConfig());
    }

    @Test
    void imageConfig_roundTripPreservesAllSupportedFields() {
        SeedreamRequest request = new SeedreamRequest();
        Map<String, Object> sequentialOptions = Map.of("max_images", 2);
        Map<String, Object> optimizeOptions = Map.of("auto", true);

        request.setImageConfig(new HashMap<>(Map.of(
                "sequentialOptions", sequentialOptions,
                "optimizeOptions", optimizeOptions,
                "sequential", "enabled",
                "guidance", 3.5,
                "mimeType", "image/png",
                "images", 2,
                "format", "b64_json",
                "seed", 42,
                "model", "seedream-4.0",
                "size", "2k"
        )));

        assertEquals(sequentialOptions, request.getSequentialOptions());
        assertEquals(optimizeOptions, request.getOptimizeOptions());
        assertEquals("enabled", request.getSequential());
        assertEquals(3.5, request.getGuidance());
        assertEquals("image/png", request.getMimeType());
        assertEquals(2, request.getImages());
        assertEquals("b64_json", request.getFormat());
        assertEquals(42, request.getSeed());
        assertEquals("seedream-4.0", request.getModel());
        assertEquals("2k", request.getSize());

        assertEquals(Map.of(
                "sequentialOptions", sequentialOptions,
                "optimizeOptions", optimizeOptions,
                "sequential", "enabled",
                "guidance", 3.5,
                "mimeType", "image/png",
                "images", 2,
                "format", "b64_json",
                "seed", 42,
                "model", "seedream-4.0",
                "size", "2k"
        ), request.getImageConfig());
    }

    @Test
    void setImageConfig_doesNotOverrideSizeWithEmptyString() {
        SeedreamRequest request = new SeedreamRequest();
        request.setSize("2k");

        Map<String, Object> imageConfig = new HashMap<>();
        imageConfig.put("size", "");
        request.setImageConfig(imageConfig);

        assertEquals("2k", request.getSize());
    }

    @Test
    void setImageConfig_keepsExistingValuesWhenIncomingValuesAreEmpty() {
        SeedreamRequest request = new SeedreamRequest();
        Map<String, Object> sequentialOptions = new HashMap<>();
        sequentialOptions.put("existing", "seq");
        Map<String, Object> optimizeOptions = new HashMap<>();
        optimizeOptions.put("existing", "opt");

        request.setSequentialOptions(sequentialOptions);
        request.setOptimizeOptions(optimizeOptions);
        request.setSequential("enabled");
        request.setGuidance(2.5);
        request.setMimeType("image/png");
        request.setImages(2);
        request.setFormat("url");
        request.setSeed(7);
        request.setModel("seedream-4.0");
        request.setSize("2k");

        Map<String, Object> imageConfig = new HashMap<>();
        imageConfig.put("sequentialOptions", Map.of());
        imageConfig.put("optimizeOptions", Map.of());
        imageConfig.put("sequential", "");
        imageConfig.put("guidance", null);
        imageConfig.put("mimeType", "");
        imageConfig.put("images", null);
        imageConfig.put("format", "");
        imageConfig.put("seed", null);
        imageConfig.put("model", "");
        imageConfig.put("size", "");
        request.setImageConfig(imageConfig);

        assertAll("existing values are preserved",
                () -> assertEquals(sequentialOptions, request.getSequentialOptions()),
                () -> assertEquals(optimizeOptions, request.getOptimizeOptions()),
                () -> assertEquals("enabled", request.getSequential()),
                () -> assertEquals(2.5, request.getGuidance()),
                () -> assertEquals("image/png", request.getMimeType()),
                () -> assertEquals(2, request.getImages()),
                () -> assertEquals("url", request.getFormat()),
                () -> assertEquals(7, request.getSeed()),
                () -> assertEquals("seedream-4.0", request.getModel()),
                () -> assertEquals("2k", request.getSize())
        );
    }

    @Test
    void setImageConfig_acceptsNullConfigWithoutChangingState() {
        SeedreamRequest request = new SeedreamRequest();
        request.setSequential("enabled");
        request.setGuidance(1.5);

        request.setImageConfig(null);

        assertAll(
                () -> assertEquals("enabled", request.getSequential()),
                () -> assertEquals(1.5, request.getGuidance())
        );
    }

    @Test
    void setImageConfig_acceptsEmptyConfigWithoutChangingState() {
        SeedreamRequest request = new SeedreamRequest();
        request.setMimeType("image/png");
        request.setImages(2);

        request.setImageConfig(Map.of());

        assertAll(
                () -> assertEquals("image/png", request.getMimeType()),
                () -> assertEquals(2, request.getImages())
        );
    }

    @Test
    void getImageConfig_omitsEmptyMapsEmptyStringsAndNullValues() {
        SeedreamRequest request = new SeedreamRequest();
        request.setSequentialOptions(Map.of());
        request.setOptimizeOptions(Map.of());
        request.setSequential("");
        request.setGuidance(null);
        request.setMimeType("");
        request.setImages(null);
        request.setFormat("");
        request.setSeed(null);
        request.setModel("");
        request.setSize("");

        assertEquals(Map.of(), request.getImageConfig());
    }
}
