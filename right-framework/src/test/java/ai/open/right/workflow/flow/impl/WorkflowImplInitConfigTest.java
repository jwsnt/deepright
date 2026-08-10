package ai.open.right.workflow.flow.impl;

import java.lang.reflect.Field;
import java.util.HashMap;
import java.util.Map;

import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import ai.open.right.workflow.flow.assistant.Assistant;
import ai.open.right.workflow.flow.config.WorkflowConfigService;
import ai.open.right.workflow.flow.trigger.WorkflowTriggerService;
import ai.open.right.workflow.notify.NotifierService;

public class WorkflowImplInitConfigTest {

    private static void setField(Object target, String name, Object value) {
        try {
            Field f = target.getClass().getDeclaredField(name);
            f.setAccessible(true);
            f.set(target, value);
        } catch (Exception e) {
            throw new RuntimeException(e);
        }
    }

    private static Object getField(Object target, String name) {
        try {
            Field f = target.getClass().getDeclaredField(name);
            f.setAccessible(true);
            return f.get(target);
        } catch (Exception e) {
            throw new RuntimeException(e);
        }
    }

    @Test
    public void shouldCreateWorkflowWithProvidedProperties() throws Exception {
        WorkflowImpl.InitConfig init = new WorkflowImpl.InitConfig();

        WorkflowTriggerService triggerService = EasyMock.createMock(WorkflowTriggerService.class);
        WorkflowConfigService configService = EasyMock.createMock(WorkflowConfigService.class);
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        Assistant assistant = EasyMock.createMock(Assistant.class);
        Map<String, Assistant> assistantMap = new HashMap<String, Assistant>();
        assistantMap.put("a", assistant);

        setField(init, "workflowTriggerService", triggerService);
        setField(init, "workflowConfigService", configService);
        setField(init, "notifierService", notifierService);
        setField(init, "assistant", assistantMap);
        setField(init, "messageOnFailed", Boolean.TRUE);
        setField(init, "deepness", 99);
        WorkflowImpl bean = (WorkflowImpl) init.workflow();
        Assert.assertSame(triggerService, getField(bean, "workflowTriggerService"));
        Assert.assertSame(configService, getField(bean, "workflowConfigService"));
        Assert.assertSame(notifierService, getField(bean, "notifierService"));
        Assert.assertSame(assistantMap, getField(bean, "assistant"));
        Assert.assertEquals(Boolean.TRUE, getField(bean, "messageOnFailed"));
        Assert.assertEquals(Integer.valueOf(99), getField(bean, "deepness"));
    }

    @Test
    public void shouldCreateWorkflowWithDefaultsWhenNoPropertiesProvided() throws Exception {
        WorkflowImpl.InitConfig init = new WorkflowImpl.InitConfig();
        WorkflowImpl bean = (WorkflowImpl) init.workflow();
        Assert.assertNull(getField(bean, "workflowTriggerService"));
        Assert.assertNull(getField(bean, "workflowConfigService"));
        Assert.assertNull(getField(bean, "notifierService"));
        Assert.assertNull(getField(bean, "assistant"));
        Assert.assertNull(getField(bean, "messageOnFailed"));
        Assert.assertNull(getField(bean, "deepness"));
    }
}
