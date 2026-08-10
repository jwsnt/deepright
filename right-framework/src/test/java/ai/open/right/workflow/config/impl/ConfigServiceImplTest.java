package ai.open.right.workflow.config.impl;

import ai.open.right.workflow.config.Config;
import ai.open.right.workflow.config.ConfigSearch;
import ai.open.right.workflow.config.ConfigService;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

public class ConfigServiceImplTest {

    @Test
    public void testGet() throws Exception {
        ConfigService delegate = EasyMock.createMock(ConfigService.class);
        ConfigSearch search = EasyMock.createMock(ConfigSearch.class);
        Config config = EasyMock.createMock(Config.class);
        
        EasyMock.expect(delegate.get(search)).andReturn(config).once();
        EasyMock.replay(delegate, search, config);
        
        ConfigServiceImpl service = new ConfigServiceImpl();
        Map<String, ConfigService> services = new HashMap<>();
        services.put("test", delegate);
        service.setConfigService(services);
        service.setInstance("test");
        
        Assertions.assertEquals(config, service.get(search));
        EasyMock.verify(delegate, search, config);
    }

    @Test
    public void testSearch() throws Exception {
        ConfigService delegate = EasyMock.createMock(ConfigService.class);
        ConfigSearch search = EasyMock.createMock(ConfigSearch.class);
        Config config = EasyMock.createMock(Config.class);
        
        EasyMock.expect(delegate.search(search)).andReturn(config).once();
        EasyMock.replay(delegate, search, config);
        
        ConfigServiceImpl service = new ConfigServiceImpl();
        Map<String, ConfigService> services = new HashMap<>();
        services.put("test", delegate);
        service.setConfigService(services);
        service.setInstance("test");
        
        Assertions.assertEquals(config, service.search(search));
        EasyMock.verify(delegate, search, config);
    }

    @Test
    public void testInitConfig() throws Exception {
        ConfigServiceImpl.InitConfig config = new ConfigServiceImpl.InitConfig();
        Map<String, ConfigService> services = new HashMap<>();
        config.setConfigService(services);
        config.setInstance("test");
        Assertions.assertNotNull(config.configService());
    }
}
