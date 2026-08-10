package ai.open.right.workflow.config.impl;

import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

import ai.open.right.workflow.config.PromptService;

public class PromptServiceImplInitConfigTest {

    @Test
    public void shouldCreatePromptServiceWithProvidedProperties() throws Exception {
        PromptServiceImpl.InitConfig init = new PromptServiceImpl.InitConfig();

        Map<String, PromptService> promptService = new HashMap<>();
        PromptService mockService = EasyMock.createMock(PromptService.class);
        promptService.put("test", mockService);

        EasyMock.replay(mockService);

        // 设置属性
        init.setPromptService(promptService);
        init.setInstance("test");

        PromptServiceImpl bean = (PromptServiceImpl) init.promptService();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof PromptServiceImpl);

        EasyMock.verify(mockService);
    }

    @Test
    public void shouldCreatePromptServiceWithDefaults() throws Exception {
        PromptServiceImpl.InitConfig init = new PromptServiceImpl.InitConfig();

        Map<String, PromptService> promptService = new HashMap<>();
        init.setPromptService(promptService);

        PromptServiceImpl bean = (PromptServiceImpl) init.promptService();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof PromptServiceImpl);
        Assert.assertEquals(promptService, bean.getPromptService());
    }
}
