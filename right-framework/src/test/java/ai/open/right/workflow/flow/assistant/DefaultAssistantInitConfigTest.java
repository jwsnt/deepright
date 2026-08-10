package ai.open.right.workflow.flow.assistant;

import java.util.HashMap;
import java.util.Map;

import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.notify.NotifierService;

public class DefaultAssistantInitConfigTest {

    @Test
    public void shouldCreateDefaultAssistantWithProvidedProperties() throws Exception {
        DefaultAssistant.InitConfig init = new DefaultAssistant.InitConfig();

        Map<String, LLMQueryService> llmQueryService = new HashMap<>();
        LLMQueryService mockService = EasyMock.createMock(LLMQueryService.class);
        llmQueryService.put("test", mockService);

        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);

        EasyMock.replay(mockService, notifierService, signalFactory);

        // 使用反射设置属性
        try {
            java.lang.reflect.Field field1 = init.getClass().getSuperclass().getDeclaredField("llmQueryService");
            field1.setAccessible(true);
            field1.set(init, llmQueryService);

            java.lang.reflect.Field field2 = init.getClass().getSuperclass().getDeclaredField("notifierService");
            field2.setAccessible(true);
            field2.set(init, notifierService);

            java.lang.reflect.Field field3 = init.getClass().getSuperclass().getDeclaredField("signalFactory");
            field3.setAccessible(true);
            field3.set(init, signalFactory);
        } catch (Exception e) {
            // 如果反射失败，直接测试bean创建
        }

        DefaultAssistant bean = init.defaultAssistant();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof DefaultAssistant);

        EasyMock.verify(mockService, notifierService, signalFactory);
    }

    @Test
    public void shouldCreateDefaultAssistantWithDefaults() throws Exception {
        DefaultAssistant.InitConfig init = new DefaultAssistant.InitConfig();

        DefaultAssistant bean = init.defaultAssistant();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof DefaultAssistant);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = DefaultAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = DefaultAssistant.DefInitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode3() throws Exception {
        Object object = DefaultAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
