package ai.open.right.config;

import ai.open.right.utils.JsonUtils;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
@Configuration
public class JacksonConfig {
    @Bean
    @ConditionalOnMissingBean(value = ObjectMapper.class)
    public ObjectMapper mapper() throws Exception {
        return JsonUtils.instance();
    }
}
