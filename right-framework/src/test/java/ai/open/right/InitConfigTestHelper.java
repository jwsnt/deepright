package ai.open.right;

import java.lang.reflect.Method;

import org.springframework.context.annotation.Bean;

public class InitConfigTestHelper {

    public static Object createBeanFromInitConfig(Class<?> initConfigClass) throws Exception {
        Object initInstance = initConfigClass.getDeclaredConstructor().newInstance();
        for (Method method : initConfigClass.getDeclaredMethods()) {
            if (method.isAnnotationPresent(Bean.class)) {
                method.setAccessible(true);
                return method.invoke(initInstance);
            }
        }
        throw new IllegalStateException("No @Bean method found on " + initConfigClass.getName());
    }
}
