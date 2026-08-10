package ai.open.right.workflow.flow.iteration.impl;

import java.lang.reflect.Field;

import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.NotifierService;

public class IterationServiceImplInitConfigTest {

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
    public void shouldCreateIterationServiceWithProvidedProperties() throws Exception {
        IterationServiceImpl.InitConfig init = new IterationServiceImpl.InitConfig();

        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);

        setField(init, "trackFunCallService", trackService);
        setField(init, "notifierService", notifierService);
        setField(init, "maxTimes", 5);
        setField(init, "timeout", 123456);
        setField(init, "prefix", "p\n");
        setField(init, "suffix", "s\n");
        setField(init, "answer", "A");
        setField(init, "query", "Q");

        IterationServiceImpl bean = (IterationServiceImpl) init.iterationService();

        Assert.assertSame(trackService, getField(bean, "trackFunCallService"));
        Assert.assertSame(notifierService, getField(bean, "notifierService"));
        Assert.assertEquals(Integer.valueOf(5), getField(bean, "maxTimes"));
        Assert.assertEquals(Integer.valueOf(123456), getField(bean, "timeout"));
        Assert.assertEquals("p\n", getField(bean, "prefix"));
        Assert.assertEquals("s\n", getField(bean, "suffix"));
        Assert.assertEquals("A", getField(bean, "answer"));
        Assert.assertEquals("Q", getField(bean, "query"));
    }

    @Test
    public void shouldCreateIterationServiceWithDefaultsWhenNoPropertiesProvided() throws Exception {
        IterationServiceImpl.InitConfig init = new IterationServiceImpl.InitConfig();

        IterationServiceImpl bean = (IterationServiceImpl) init.iterationService();

        Assert.assertNull(getField(bean, "trackFunCallService"));
        Assert.assertNull(getField(bean, "notifierService"));
        Assert.assertNull(getField(bean, "maxTimes"));
        Assert.assertNull(getField(bean, "timeout"));
        Assert.assertEquals("##################\n", getField(bean, "prefix"));
        Assert.assertEquals("##################\n", getField(bean, "suffix"));
        Assert.assertEquals("The answer round", getField(bean, "answer"));
        Assert.assertEquals("The query round", getField(bean, "query"));
    }
}
