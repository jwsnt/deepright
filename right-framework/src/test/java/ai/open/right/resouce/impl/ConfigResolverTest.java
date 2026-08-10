package ai.open.right.resouce.impl;

import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.core.env.Environment;

public class ConfigResolverTest {

    @Test
    public void testWithEmpty1() throws Exception {
        Environment environment = EasyMock.createMock(Environment.class);
        EasyMock.expect(environment.getProperty("host", "${host}")).andReturn("${host}").anyTimes();
        EasyMock.replay(environment);
        PlaceholderResolverImpl placeholderResolver = new PlaceholderResolverImpl();
        placeholderResolver.setEnvironment(environment);
        placeholderResolver.setPrefix("right_");
        Assert.assertEquals("", placeholderResolver.replace(""));
        EasyMock.verify(environment);
    }

    @Test
    public void testWithEmpty2() throws Exception {
        Environment environment = EasyMock.createMock(Environment.class);
        EasyMock.expect(environment.getProperty("host", "${host}")).andReturn("${host}").anyTimes();
        EasyMock.replay(environment);
        PlaceholderResolverImpl placeholderResolver = new PlaceholderResolverImpl();
        placeholderResolver.setEnvironment(environment);
        Assert.assertEquals("", placeholderResolver.replace(""));
        EasyMock.verify(environment);
    }

    @Test
    public void testWithNotReplace1() throws Exception {
        Environment environment = EasyMock.createMock(Environment.class);
        EasyMock.expect(environment.getProperty("right_host", "${right_host}")).andReturn("${right_host}").anyTimes();
        EasyMock.replay(environment);
        PlaceholderResolverImpl placeholderResolver = new PlaceholderResolverImpl();
        placeholderResolver.setPrefix("right_");
        placeholderResolver.setEnvironment(environment);
        Assert.assertEquals("http://${right_host}/abc", placeholderResolver.replace("http://${right_host}/abc"));
        EasyMock.verify(environment);
    }

    @Test
    public void testWithNotReplace2() throws Exception {
        Environment environment = EasyMock.createMock(Environment.class);
        EasyMock.expect(environment.getProperty("host", "${host}")).andReturn("${host}").anyTimes();
        EasyMock.replay(environment);
        PlaceholderResolverImpl placeholderResolver = new PlaceholderResolverImpl();
        placeholderResolver.setEnvironment(environment);
        Assert.assertEquals("http://${host}/abc", placeholderResolver.replace("http://${host}/abc"));
        EasyMock.verify(environment);
    }

    @Test
    public void testWithReplace1() throws Exception {
        Environment environment = EasyMock.createMock(Environment.class);
        EasyMock.expect(environment.getProperty("right_host", "${right_host}")).andReturn("hello_world").anyTimes();
        EasyMock.replay(environment);
        PlaceholderResolverImpl placeholderResolver = new PlaceholderResolverImpl();
        placeholderResolver.setEnvironment(environment);
        placeholderResolver.setPrefix("right_");
        Assert.assertEquals("http://hello_world/abc", placeholderResolver.replace("http://${right_host}/abc"));
        EasyMock.verify(environment);
    }

    @Test
    public void testWithReplace2() throws Exception {
        Environment environment = EasyMock.createMock(Environment.class);
        EasyMock.expect(environment.getProperty("host", "${host}")).andReturn("hello_world").anyTimes();
        EasyMock.replay(environment);
        PlaceholderResolverImpl placeholderResolver = new PlaceholderResolverImpl();
        placeholderResolver.setEnvironment(environment);
        Assert.assertEquals("http://hello_world/abc", placeholderResolver.replace("http://${host}/abc"));
        EasyMock.verify(environment);
    }

    @Test
    public void testWithMultiOneReplace1() throws Exception {
        Environment environment = EasyMock.createMock(Environment.class);
        EasyMock.expect(environment.getProperty("right_host", "${right_host}")).andReturn("hello_world").anyTimes();
        EasyMock.expect(environment.getProperty("right_path", "${right_path}")).andReturn("${right_path}").anyTimes();
        EasyMock.replay(environment);
        PlaceholderResolverImpl placeholderResolver = new PlaceholderResolverImpl();
        placeholderResolver.setEnvironment(environment);
        placeholderResolver.setPrefix("right_");
        Assert.assertEquals("http://hello_world/${right_path}", placeholderResolver.replace("http://${right_host}/${right_path}"));
        EasyMock.verify(environment);
    }

    @Test
    public void testWithMultiOneReplace2() throws Exception {
        Environment environment = EasyMock.createMock(Environment.class);
        EasyMock.expect(environment.getProperty("right_host", "${right_host}")).andReturn("hello_world").anyTimes();
        EasyMock.expect(environment.getProperty("right_path", "${right_path}")).andReturn("${right_path}").anyTimes();
        EasyMock.replay(environment);
        PlaceholderResolverImpl placeholderResolver = new PlaceholderResolverImpl();
        placeholderResolver.setEnvironment(environment);
        placeholderResolver.setPrefix("right_");
        Assert.assertEquals("http://hello_world/${right_path}", placeholderResolver.replace("http://${right_host}/${right_path}"));
        EasyMock.verify(environment);
    }

    @Test
    public void testWithMultiAllReplace1() throws Exception {
        Environment environment = EasyMock.createMock(Environment.class);
        EasyMock.expect(environment.getProperty("right_host", "${right_host}")).andReturn("hello_world").anyTimes();
        EasyMock.expect(environment.getProperty("right_path", "${right_path}")).andReturn("abc").anyTimes();
        EasyMock.replay(environment);
        PlaceholderResolverImpl placeholderResolver = new PlaceholderResolverImpl();
        placeholderResolver.setEnvironment(environment);
        placeholderResolver.setPrefix("right_");
        Assert.assertEquals("http://hello_world/abc", placeholderResolver.replace("http://${right_host}/${right_path}"));
        EasyMock.verify(environment);
    }

    @Test
    public void testWithMultiAllReplace2() throws Exception {
        Environment environment = EasyMock.createMock(Environment.class);
        EasyMock.expect(environment.getProperty("host", "${host}")).andReturn("hello_world").anyTimes();
        EasyMock.expect(environment.getProperty("path", "${path}")).andReturn("abc").anyTimes();
        EasyMock.replay(environment);
        PlaceholderResolverImpl placeholderResolver = new PlaceholderResolverImpl();
        placeholderResolver.setEnvironment(environment);
        Assert.assertEquals("http://hello_world/abc", placeholderResolver.replace("http://${host}/${path}"));
        EasyMock.verify(environment);
    }
}
