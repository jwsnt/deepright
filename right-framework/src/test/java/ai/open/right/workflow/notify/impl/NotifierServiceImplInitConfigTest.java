package ai.open.right.workflow.notify.impl;

import java.lang.reflect.Field;
import java.util.HashMap;
import java.util.Map;

import org.junit.Assert;
import org.junit.Test;

import ai.open.right.workflow.notify.Notifier;

public class NotifierServiceImplInitConfigTest {

    @Test
    public void shouldCreateNotifierServiceBeanAndCopyProperties() throws Exception {
        NotifierServiceImpl.InitConfig init = new NotifierServiceImpl.InitConfig();
        Map<String, Notifier> map = new HashMap<String, Notifier>();
        setPrivate(init, "notifier", map);

        NotifierServiceImpl bean = (NotifierServiceImpl) init.notifierService();

        Assert.assertSame(map, readPrivate(bean, "notifier", Map.class));
    }

    private static <T> T readPrivate(Object target, String fieldName, Class<T> type) {
        try {
            Field f = target.getClass().getDeclaredField(fieldName);
            f.setAccessible(true);
            Object value = f.get(target);
            return type.cast(value);
        } catch (Exception e) {
            throw new AssertionError("Failed to read field '" + fieldName + "'", e);
        }
    }

    private static void setPrivate(Object target, String fieldName, Object value) throws Exception {
        Field f = target.getClass().getDeclaredField(fieldName);
        f.setAccessible(true);
        f.set(target, value);
    }

    @org.junit.jupiter.api.Test
    public void testNotifyNullNotifier() {
        NotifierServiceImpl service = new NotifierServiceImpl();
        service.setNotifier(new java.util.HashMap<>());
        org.junit.jupiter.api.Assertions.assertThrows(IllegalArgumentException.class, () -> {
            service.notify((String) null, null, null, null);
        });
    }
}

