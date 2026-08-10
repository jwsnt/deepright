package ai.open.right.config;

import org.easymock.EasyMock;
import org.junit.Test;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.system.ApplicationHome;
import org.springframework.core.env.ConfigurableEnvironment;
import org.springframework.core.env.MutablePropertySources;
import org.springframework.core.env.PropertySource;
import org.springframework.core.env.MapPropertySource;

import java.io.File;
import java.util.ArrayList;
import java.util.List;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.junit.jupiter.api.Assertions.assertFalse;

public class PropertiesConfigTest {
    @Test
    public void test() {
        PropertySource propertySource = EasyMock.createMock(PropertySource.class);
        EasyMock.expect(propertySource.getName()).andReturn("systemEnvironment").anyTimes();
        EasyMock.expect(propertySource.getSource()).andReturn("SOURCE").anyTimes();
        MutablePropertySources propertySources = EasyMock.createMock(MutablePropertySources.class);
        propertySources.addFirst(EasyMock.anyObject());
        EasyMock.expectLastCall().anyTimes();
        List<PropertySource<?>> ps = new ArrayList<>();
        ps.add(propertySource);
        EasyMock.expect(propertySources.iterator()).andReturn(ps.iterator());
        propertySources.replace(EasyMock.anyObject(String.class), EasyMock.anyObject(PropertiesConfig.SysEnvPropertySource.class));
        EasyMock.expectLastCall().anyTimes();
        ConfigurableEnvironment environment = EasyMock.createMock(ConfigurableEnvironment.class);
        SpringApplication application = EasyMock.createMock(SpringApplication.class);
        EasyMock.expect(environment.getPropertySources()).andReturn(propertySources).anyTimes();
        ApplicationHome applicationHome = EasyMock.createMock(ApplicationHome.class);
        File source = new File(".");
        EasyMock.expect(applicationHome.getSource()).andReturn(source).anyTimes();
        EasyMock.replay(applicationHome, environment, application, propertySource, propertySources);
        PropertiesConfig propertiesConfig = new PropertiesConfig() {
            @Override
            protected ApplicationHome buildApp(SpringApplication application) {
                return applicationHome;
            }
        };
        propertiesConfig.postProcessEnvironment(environment, application);
        EasyMock.verify(applicationHome, environment, application, propertySource, propertySources);
    }

    @org.junit.jupiter.api.Test
    public void testPropertiesConfig() {
        PropertiesConfig config = new PropertiesConfig();
        assertNotNull(config);
    }

    @org.junit.jupiter.api.Test
    public void testSysEnvPropertySource() {
        PropertySource delegate = EasyMock.createMock(PropertySource.class);
        EasyMock.expect(delegate.getName()).andReturn("systemEnvironment").anyTimes();
        EasyMock.expect(delegate.getSource()).andReturn("SOURCE").anyTimes();

        // Case 1: dot to underscore
        EasyMock.expect(delegate.getProperty("my_prop")).andReturn("value1");

        // Case 2: dot to uppercase underscore
        EasyMock.expect(delegate.getProperty("my_prop_upper")).andReturn(null);
        EasyMock.expect(delegate.getProperty("MY_PROP_UPPER")).andReturn("value2");

        // Case 3: original name
        EasyMock.expect(delegate.getProperty("original_prop")).andReturn(null);
        EasyMock.expect(delegate.getProperty("ORIGINAL_PROP")).andReturn(null);
        EasyMock.expect(delegate.getProperty("original.prop")).andReturn("value3");

        EasyMock.replay(delegate);

        PropertiesConfig.SysEnvPropertySource source = new PropertiesConfig.SysEnvPropertySource(delegate);

        assertEquals("value1", source.getProperty("my.prop"));
        assertEquals("value2", source.getProperty("my.prop.upper"));
        assertEquals("value3", source.getProperty("original.prop"));

        EasyMock.verify(delegate);
    }

    @org.junit.jupiter.api.Test
    public void testSysEnvPropertySourceNoMatch() {
        PropertySource delegate = EasyMock.createMock(PropertySource.class);
        EasyMock.expect(delegate.getName()).andReturn("systemEnvironment").anyTimes();
        EasyMock.expect(delegate.getSource()).andReturn("SOURCE").anyTimes();

        // 模拟三次尝试均返回 null
        EasyMock.expect(delegate.getProperty("no_match")).andReturn(null);
        EasyMock.expect(delegate.getProperty("NO_MATCH")).andReturn(null);
        EasyMock.expect(delegate.getProperty("no.match")).andReturn(null);

        EasyMock.replay(delegate);

        PropertiesConfig.SysEnvPropertySource source = new PropertiesConfig.SysEnvPropertySource(delegate);
        assertNull(source.getProperty("no.match"));

        EasyMock.verify(delegate);
    }

    @org.junit.jupiter.api.Test
    public void testPostProcessEnvironmentWithoutSystemEnv() {
        ConfigurableEnvironment environment = EasyMock.createMock(ConfigurableEnvironment.class);
        SpringApplication application = EasyMock.createMock(SpringApplication.class);
        MutablePropertySources propertySources = EasyMock.createMock(MutablePropertySources.class);

        EasyMock.expect(environment.getPropertySources()).andReturn(propertySources).anyTimes();
        // 返回空迭代器，模拟不存在任何属性源
        EasyMock.expect(propertySources.iterator()).andReturn(new ArrayList<PropertySource<?>>().iterator());
        
        propertySources.addFirst(EasyMock.anyObject());
        EasyMock.expectLastCall().anyTimes();

        ApplicationHome applicationHome = EasyMock.createMock(ApplicationHome.class);
        EasyMock.expect(applicationHome.getSource()).andReturn(new File(".")).anyTimes();

        EasyMock.replay(environment, application, propertySources, applicationHome);

        PropertiesConfig propertiesConfig = new PropertiesConfig() {
            @Override
            protected ApplicationHome buildApp(SpringApplication application) {
                return applicationHome;
            }
        };

        // 即使缺少 systemEnvironment，也应正常执行不抛异常
        propertiesConfig.postProcessEnvironment(environment, application);

        EasyMock.verify(environment, application, propertySources, applicationHome);
    }

    @org.junit.jupiter.api.Test
    public void testBuildApp() {
        PropertiesConfig config = new PropertiesConfig();
        SpringApplication application = new SpringApplication(PropertiesConfig.class);
        ApplicationHome home = config.buildApp(application);
        assertNotNull(home);
    }

    @org.junit.jupiter.api.Test
    public void testBuildAppWithNullClass() {
        PropertiesConfig config = new PropertiesConfig();
        SpringApplication application = EasyMock.createMock(SpringApplication.class);
        EasyMock.expect(application.getMainApplicationClass()).andReturn(null).anyTimes();
        EasyMock.replay(application);
        
        ApplicationHome home = config.buildApp(application);
        assertNotNull(home);
        assertNull(home.getSource());
        
        EasyMock.verify(application);
    }

    @org.junit.jupiter.api.Test
    public void testBuildProjectWithJar() {
        ConfigurableEnvironment environment = EasyMock.createMock(ConfigurableEnvironment.class);
        SpringApplication application = EasyMock.createMock(SpringApplication.class);
        MutablePropertySources propertySources = EasyMock.createMock(MutablePropertySources.class);

        EasyMock.expect(environment.getPropertySources()).andReturn(propertySources).anyTimes();
        
        propertySources.addFirst(EasyMock.anyObject(MapPropertySource.class));
        EasyMock.expectLastCall().andAnswer(() -> {
            MapPropertySource ps = (MapPropertySource) EasyMock.getCurrentArguments()[0];
            String projectPath = (String) ps.getProperty("project");
            assertTrue(projectPath.contains("BOOT-INF" + File.separator + "classes"));
            assertFalse(projectPath.contains(".jar" + File.separator + "BOOT-INF"));
            return null;
        });

        ApplicationHome applicationHome = EasyMock.createMock(ApplicationHome.class);
        File jarFile = new File("test.jar");
        EasyMock.expect(applicationHome.getSource()).andReturn(jarFile).anyTimes();

        EasyMock.replay(environment, application, propertySources, applicationHome);

        PropertiesConfig propertiesConfig = new PropertiesConfig() {
            @Override
            protected ApplicationHome buildApp(SpringApplication application) {
                return applicationHome;
            }
        };

        propertiesConfig.buildProject(environment, application);

        EasyMock.verify(environment, application, propertySources, applicationHome);
    }

    @org.junit.jupiter.api.Test
    public void testBuildProjectWithJarUpperCase() {
        ConfigurableEnvironment environment = EasyMock.createMock(ConfigurableEnvironment.class);
        SpringApplication application = EasyMock.createMock(SpringApplication.class);
        MutablePropertySources propertySources = EasyMock.createMock(MutablePropertySources.class);

        EasyMock.expect(environment.getPropertySources()).andReturn(propertySources).anyTimes();
        
        propertySources.addFirst(EasyMock.anyObject(MapPropertySource.class));
        EasyMock.expectLastCall().andAnswer(() -> {
            MapPropertySource ps = (MapPropertySource) EasyMock.getCurrentArguments()[0];
            String projectPath = (String) ps.getProperty("project");
            assertTrue(projectPath.contains("BOOT-INF" + File.separator + "classes"));
            assertFalse(projectPath.contains(".JAR" + File.separator + "BOOT-INF"));
            return null;
        });

        ApplicationHome applicationHome = EasyMock.createMock(ApplicationHome.class);
        File jarFile = new File("test.JAR");
        EasyMock.expect(applicationHome.getSource()).andReturn(jarFile).anyTimes();

        EasyMock.replay(environment, application, propertySources, applicationHome);

        PropertiesConfig propertiesConfig = new PropertiesConfig() {
            @Override
            protected ApplicationHome buildApp(SpringApplication application) {
                return applicationHome;
            }
        };

        propertiesConfig.buildProject(environment, application);

        EasyMock.verify(environment, application, propertySources, applicationHome);
    }

    @org.junit.jupiter.api.Test
    public void testBuildProjectWithoutJar() {
        ConfigurableEnvironment environment = EasyMock.createMock(ConfigurableEnvironment.class);
        SpringApplication application = EasyMock.createMock(SpringApplication.class);
        MutablePropertySources propertySources = EasyMock.createMock(MutablePropertySources.class);

        EasyMock.expect(environment.getPropertySources()).andReturn(propertySources).anyTimes();
        
        propertySources.addFirst(EasyMock.anyObject(MapPropertySource.class));
        EasyMock.expectLastCall();

        ApplicationHome applicationHome = EasyMock.createMock(ApplicationHome.class);
        File dir = new File("test-dir");
        EasyMock.expect(applicationHome.getSource()).andReturn(dir).anyTimes();

        EasyMock.replay(environment, application, propertySources, applicationHome);

        PropertiesConfig propertiesConfig = new PropertiesConfig() {
            @Override
            protected ApplicationHome buildApp(SpringApplication application) {
                return applicationHome;
            }
        };

        propertiesConfig.buildProject(environment, application);

        EasyMock.verify(environment, application, propertySources, applicationHome);
    }

    @org.junit.jupiter.api.Test
    public void testBuildProjectWithNullSource() {
        ConfigurableEnvironment environment = EasyMock.createMock(ConfigurableEnvironment.class);
        SpringApplication application = EasyMock.createMock(SpringApplication.class);
        MutablePropertySources propertySources = EasyMock.createMock(MutablePropertySources.class);

        EasyMock.expect(environment.getPropertySources()).andReturn(propertySources).anyTimes();
        
        propertySources.addFirst(EasyMock.anyObject(MapPropertySource.class));
        EasyMock.expectLastCall();

        ApplicationHome applicationHome = EasyMock.createMock(ApplicationHome.class);
        EasyMock.expect(applicationHome.getSource()).andReturn(null).anyTimes();

        EasyMock.replay(environment, application, propertySources, applicationHome);

        PropertiesConfig propertiesConfig = new PropertiesConfig() {
            @Override
            protected ApplicationHome buildApp(SpringApplication application) {
                return applicationHome;
            }
        };

        propertiesConfig.buildProject(environment, application);

        EasyMock.verify(environment, application, propertySources, applicationHome);
    }
}
