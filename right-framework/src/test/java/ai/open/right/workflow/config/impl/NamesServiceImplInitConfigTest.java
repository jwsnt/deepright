package ai.open.right.workflow.config.impl;

import org.junit.Assert;
import org.junit.Test;

public class NamesServiceImplInitConfigTest {

    @Test
    public void shouldCreateNamesServiceWithProvidedProperties() throws Exception {
        NamesServiceImpl.InitConfig init = new NamesServiceImpl.InitConfig();
        // 设置属性
        init.setLength(64);
        init.setEncode(false);
        NamesServiceImpl bean = (NamesServiceImpl) init.namesService();
        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof NamesServiceImpl);
        Assert.assertEquals(Integer.valueOf(64), bean.getLength());
        Assert.assertEquals(Boolean.FALSE, bean.getEncode());
    }

    @Test
    public void shouldCreateNamesServiceWithDefaults() throws Exception {
        NamesServiceImpl.InitConfig init = new NamesServiceImpl.InitConfig();
        NamesServiceImpl bean = (NamesServiceImpl) init.namesService();
        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof NamesServiceImpl);
        Assert.assertEquals(Integer.valueOf(8), bean.getLength());
        Assert.assertEquals(Boolean.FALSE, bean.getEncode());
    }
}
