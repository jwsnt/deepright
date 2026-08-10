package ai.open.right.workflow.config.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.config.Config;
import ai.open.right.workflow.config.ConfigSearch;
import org.junit.Assert;
import org.junit.Test;

public class FileConfigServiceTest {

    @Test
    public void testGet() throws Exception {
        FileConfigService fileConfigService = new FileConfigService();
        fileConfigService.setResourceService(ObjectBuilder.buildResourceService());
        fileConfigService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileConfigService.setPath("classpath:config");
        fileConfigService.setSuffix(".json");
        fileConfigService.init();
        Assert.assertNotNull(fileConfigService.getResourceService());
        ConfigSearch search = new ConfigSearch();
        search.setLanguage("LA");
        search.setDevice("DE");
        search.setBiz("prompt");
        Config config = fileConfigService.get(search);
        Assert.assertTrue(config.getConfigs().get("introduce").toString().length() > 20);
    }

    @Test
    public void testPath() {
        FileConfigService fileConfigService = new FileConfigService();
        fileConfigService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileConfigService.setPath("classpath:config/example/dynamic");
        fileConfigService.init();
        Assert.assertEquals("classpath:config/example/dynamic/", fileConfigService.getPath());
    }
}
