package ai.open.right.config;

import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.core.env.PropertySource;

public class SysEnvPropertySourceTest {

    @Test
    public void test1() {
        PropertySource propertySource = EasyMock.createMock(PropertySource.class);
        EasyMock.expect(propertySource.getName()).andReturn("NAME").anyTimes();
        EasyMock.expect(propertySource.getSource()).andReturn("SOURCE").anyTimes();
        EasyMock.expect(propertySource.getProperty("a_b_c")).andReturn("ABC").once();
        EasyMock.replay(propertySource);
        PropertiesConfig.SysEnvPropertySource sysEnvPropertySource = new PropertiesConfig.SysEnvPropertySource(propertySource);
        Assert.assertEquals("ABC", sysEnvPropertySource.getProperty("a.b.c"));
        EasyMock.verify(propertySource);
    }

    @Test
    public void test2() {
        PropertySource propertySource = EasyMock.createMock(PropertySource.class);
        EasyMock.expect(propertySource.getName()).andReturn("NAME").anyTimes();
        EasyMock.expect(propertySource.getSource()).andReturn("SOURCE").anyTimes();
        EasyMock.expect(propertySource.getProperty("a_b_c")).andReturn(null).once();
        EasyMock.expect(propertySource.getProperty("A_B_C")).andReturn("abc").once();
        EasyMock.replay(propertySource);
        PropertiesConfig.SysEnvPropertySource sysEnvPropertySource = new PropertiesConfig.SysEnvPropertySource(propertySource);
        Assert.assertEquals("abc", sysEnvPropertySource.getProperty("a.b.c"));
        EasyMock.verify(propertySource);
    }

    @Test
    public void test3() {
        PropertySource propertySource = EasyMock.createMock(PropertySource.class);
        EasyMock.expect(propertySource.getName()).andReturn("NAME").anyTimes();
        EasyMock.expect(propertySource.getSource()).andReturn("SOURCE").anyTimes();
        EasyMock.expect(propertySource.getProperty("A_B_C")).andReturn(null).once();
        EasyMock.expect(propertySource.getProperty("a_b_c")).andReturn(null).once();
        EasyMock.expect(propertySource.getProperty("a.b.c")).andReturn("cba").once();
        EasyMock.replay(propertySource);
        PropertiesConfig.SysEnvPropertySource sysEnvPropertySource = new PropertiesConfig.SysEnvPropertySource(propertySource);
        Assert.assertEquals("cba", sysEnvPropertySource.getProperty("a.b.c"));
        EasyMock.verify(propertySource);
    }
}
