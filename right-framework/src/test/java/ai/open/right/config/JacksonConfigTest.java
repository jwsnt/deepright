package ai.open.right.config;

import org.junit.Test;

public class JacksonConfigTest {

    @Test
    public void testObjectMapper() throws Exception {
        JacksonConfig jacksonConfig = new JacksonConfig();
        jacksonConfig.mapper();
    }

    @org.junit.jupiter.api.Test
    public void testMapper() throws Exception {
        JacksonConfig config = new JacksonConfig();
        com.fasterxml.jackson.databind.ObjectMapper mapper = config.mapper();
        org.junit.jupiter.api.Assertions.assertNotNull(mapper);
    }

    @org.junit.jupiter.api.Test
    public void testMapperInstance() throws Exception {
        JacksonConfig config = new JacksonConfig();
        org.junit.jupiter.api.Assertions.assertSame(ai.open.right.utils.JsonUtils.instance(), config.mapper());
    }

}
