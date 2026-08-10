package ai.open.right.workflow.flow.script.impl;

import org.junit.Assert;
import org.junit.Test;

import java.lang.reflect.Field;

public class ScriptServiceImplInitConfigTest {

    @Test
    public void shouldCreateScriptServiceBeanAndCopyProperties() throws Exception {
        ScriptServiceImpl.InitConfig init = new ScriptServiceImpl.InitConfig();
        setPrivate(init, "commandService", new CommandService());
        setPrivate(init, "javaScriptService", new JavaScriptService());
        setPrivate(init, "jythonService", new JythonService());
        setPrivate(init, "pythonService", new PythonService());
        setPrivate(init, "luaService", new LuaService());
        ScriptServiceImpl bean = (ScriptServiceImpl) init.scriptService();
        Assert.assertSame(readPrivate(init, "commandService", CommandService.class), readPrivate(bean, "commandService", CommandService.class));
        Assert.assertSame(readPrivate(init, "javaScriptService", JavaScriptService.class), readPrivate(bean, "javaScriptService", JavaScriptService.class));
        Assert.assertSame(readPrivate(init, "jythonService", JythonService.class), readPrivate(bean, "jythonService", JythonService.class));
        Assert.assertSame(readPrivate(init, "pythonService", PythonService.class), readPrivate(bean, "pythonService", PythonService.class));
        Assert.assertSame(readPrivate(init, "luaService", LuaService.class), readPrivate(bean, "luaService", LuaService.class));
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
}
