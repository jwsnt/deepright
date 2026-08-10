package ai.open.right.workflow.config.impl;

import ai.open.right.workflow.config.ConfigService;
import ai.open.right.workflow.config.impl.ConfigServiceImpl;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class ConfigServiceImplInitConfigTest {

    @Test
    public void testInit() throws Exception {
        Map<String, ConfigService> configService = new HashMap<>();
        ConfigServiceImpl.InitConfig initConfig = new ConfigServiceImpl.InitConfig();
        initConfig.setConfigService(configService);
        initConfig.setInstance("instance");
        ConfigServiceImpl empty = ConfigServiceImpl.class.cast(initConfig.configService());
        Assert.assertEquals(configService, empty.getConfigService());
        Assert.assertEquals("instance", empty.getInstance());
    }

    @org.junit.jupiter.api.Test
    public void testInstanceNull() {
        ConfigServiceImpl service = new ConfigServiceImpl();
        service.setConfigService(new java.util.HashMap<>());
        service.setInstance("none");
        // 修改预期异常为 IllegalArgumentException
        org.junit.jupiter.api.Assertions.assertThrows(IllegalArgumentException.class, () -> {
            service.instance();
        });
    }
}
