package ai.open.right.workflow.config;

import ai.open.right.workflow.config.impl.ConfigServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class ConfigManagerTest {

    @Test
    public void testGet() throws Exception {
        ConfigService configService = EasyMock.createMock(ConfigService.class);
        ConfigSearch configSearch = new ConfigSearch();
        Map<String, Object> configs = new HashMap<>();
        Config config = new Config("BIZ", configs);
        EasyMock.expect(configService.get(configSearch)).andReturn(config).anyTimes();
        Map<String, ConfigService> configServices = new HashMap<String, ConfigService>();
        configServices.put("CS", configService);
        EasyMock.replay(configService);
        ConfigServiceImpl configManager = new ConfigServiceImpl();
        configManager.setConfigService(configServices);
        configManager.setInstance("CS");
        Assert.assertEquals(config, configManager.get(configSearch));
        EasyMock.verify(configService);
    }

    @Test
    public void testSearch() throws Exception {
        ConfigService configService = EasyMock.createMock(ConfigService.class);
        ConfigSearch configSearch = new ConfigSearch();
        Map<String, Object> configs = new HashMap<>();
        Config config = new Config("BIZ", configs);
        EasyMock.expect(configService.search(configSearch)).andReturn(config).anyTimes();
        Map<String, ConfigService> configServices = new HashMap<String, ConfigService>();
        configServices.put("CS", configService);
        EasyMock.replay(configService);
        ConfigServiceImpl configManager = new ConfigServiceImpl();
        configManager.setConfigService(configServices);
        configManager.setInstance("CS");
        Assert.assertEquals(config, configManager.search(configSearch));
        EasyMock.verify(configService);
    }
}
