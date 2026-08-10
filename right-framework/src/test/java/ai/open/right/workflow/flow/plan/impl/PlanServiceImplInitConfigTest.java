package ai.open.right.workflow.flow.plan.impl;

import java.lang.reflect.Field;

import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.NotifierService;

public class PlanServiceImplInitConfigTest {

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
    public void shouldCreatePlanServiceWithProvidedProperties() throws Exception {
        PlanServiceImpl.InitConfig init = new PlanServiceImpl.InitConfig();

        NotifierService notifierService = EasyMock.createMock(NotifierService.class);

        setField(init, "notifierService", notifierService);
        setField(init, "timeout4Llm", 111);

        PlanServiceImpl bean = (PlanServiceImpl) init.planService();

        Assert.assertSame(notifierService, getField(bean, "notifierService"));
        Assert.assertEquals(Integer.valueOf(111), getField(bean, "timeout4Llm"));
    }

    @Test
    public void shouldCreatePlanServiceWithDefaultsWhenNoPropertiesProvided() throws Exception {
        PlanServiceImpl.InitConfig init = new PlanServiceImpl.InitConfig();
        PlanServiceImpl bean = (PlanServiceImpl) init.planService();
        Assert.assertNull(getField(bean, "notifierService"));
        Assert.assertNull(getField(bean, "timeout4Llm"));
    }
}
