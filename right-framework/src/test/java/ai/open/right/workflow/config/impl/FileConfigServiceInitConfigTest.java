package ai.open.right.workflow.config.impl;

import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import ai.open.right.resouce.PlaceholderResolver;

public class FileConfigServiceInitConfigTest {

    @Test
    public void shouldCreateFileConfigServiceWithProvidedProperties() throws Exception {
        FileConfigService.InitConfig init = new FileConfigService.InitConfig();

        PlaceholderResolver placeholder = EasyMock.createMock(PlaceholderResolver.class);

        EasyMock.replay(placeholder);

        // 设置属性
        init.setPlaceholderResolver(placeholder);
        init.setSuffix(".xml");
        init.setPath("classpath:test/");

        FileConfigService bean = FileConfigService.class.cast(init.configService());

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof FileConfigService);

        EasyMock.verify(placeholder);
    }

    @Test
    public void shouldCreateFileConfigServiceWithDefaults() throws Exception {
        FileConfigService.InitConfig init = new FileConfigService.InitConfig();

        PlaceholderResolver placeholder = EasyMock.createMock(PlaceholderResolver.class);
        init.setPlaceholderResolver(placeholder);

        EasyMock.replay(placeholder);

        FileConfigService bean = FileConfigService.class.cast(init.configService());

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof FileConfigService);
        Assert.assertEquals(placeholder, bean.getPlaceholderResolver());

        EasyMock.verify(placeholder);
    }
}
