package ai.open.right.workflow.config;

import lombok.Getter;
import lombok.Setter;
import org.springframework.util.Assert;

import java.util.Map;

@Setter
@Getter
public class Config {

    protected Map<String, Object> configs;

    protected String biz;

    public Config(String biz, Map<String, Object> configs) {
        this.configs = configs;
        this.biz = biz;
    }

    public static class ConfigChecker {

        public static void check(Config config) {
            Assert.notNull(config.getConfigs(), "Configs can not be empty");
            Assert.hasText(config.getBiz(), "Biz can not be empty");
        }
    }
}