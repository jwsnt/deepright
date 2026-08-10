package ai.open.right.workflow.config;

import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class ConfigTest {

    @Test
    public void testGetSet() {
        Map<String, Object> configs = new HashMap<>();
        Config config = new Config("BIZ", configs);
        config.setBiz("BIZ");
        config.setConfigs(configs);
        Assert.assertEquals(configs, config.getConfigs());
        Assert.assertEquals("BIZ", config.getBiz());
    }
}
