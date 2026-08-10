package ai.open.right.resouce.impl;

import ai.open.right.ObjectBuilder;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.core.env.Environment;

public class PlaceholderResolverImplTest {

    @Test
    public void testInit() throws Exception {
        Environment environment = EasyMock.createMock(Environment.class);
        PlaceholderResolverImpl.InitConfig initConfig = new PlaceholderResolverImpl.InitConfig();
        initConfig.setEnvironment(environment);
        PlaceholderResolverImpl empty = (PlaceholderResolverImpl) initConfig.placeholderResolver();
        Assert.assertEquals(empty.getEnvironment(), environment);
    }

    @org.junit.jupiter.api.Test
    public void testReplaceNoPlaceholder() throws Exception {
        PlaceholderResolverImpl resolver = new PlaceholderResolverImpl();
        String input = "no placeholder";
        org.junit.jupiter.api.Assertions.assertEquals(input, resolver.replace(input));
    }

    @org.junit.jupiter.api.Test
    public void testReplaceWithEnvironment() throws Exception {
        org.springframework.core.env.Environment environment = org.easymock.EasyMock.createMock(org.springframework.core.env.Environment.class);
        org.easymock.EasyMock.expect(environment.getProperty("key", "${key}")).andReturn("value").anyTimes();
        org.easymock.EasyMock.replay(environment);
        PlaceholderResolverImpl resolver = new PlaceholderResolverImpl();
        resolver.setEnvironment(environment);
        org.junit.jupiter.api.Assertions.assertEquals("value", resolver.replace("${key}"));
        org.easymock.EasyMock.verify(environment);
    }

    @org.junit.jupiter.api.Test
    public void testReplaceWithPrefix() throws Exception {
        org.springframework.core.env.Environment environment = org.easymock.EasyMock.createMock(org.springframework.core.env.Environment.class);
        org.easymock.EasyMock.expect(environment.getProperty("right.key", "${right.key}")).andReturn("value").anyTimes();
        org.easymock.EasyMock.replay(environment);
        PlaceholderResolverImpl resolver = new PlaceholderResolverImpl();
        resolver.setEnvironment(environment);
        resolver.setPrefix("right.");
        org.junit.jupiter.api.Assertions.assertEquals("value", resolver.replace("${right.key}"));
        org.junit.jupiter.api.Assertions.assertEquals("${other.key}", resolver.replace("${other.key}"));
    }

    @org.junit.jupiter.api.Test
    public void testReplaceNullOrEmpty() throws Exception {
        PlaceholderResolverImpl resolver = new PlaceholderResolverImpl();
        org.junit.jupiter.api.Assertions.assertNull(resolver.replace(null));
        org.junit.jupiter.api.Assertions.assertEquals("", resolver.replace(""));
    }

    @org.junit.jupiter.api.Test
    public void testReplaceMultiple() throws Exception {
        org.springframework.core.env.Environment environment = org.easymock.EasyMock.createMock(org.springframework.core.env.Environment.class);
        org.easymock.EasyMock.expect(environment.getProperty("k1", "${k1}")).andReturn("v1").anyTimes();
        org.easymock.EasyMock.expect(environment.getProperty("k2", "${k2}")).andReturn("v2").anyTimes();
        org.easymock.EasyMock.replay(environment);
        PlaceholderResolverImpl resolver = new PlaceholderResolverImpl();
        resolver.setEnvironment(environment);
        org.junit.jupiter.api.Assertions.assertEquals("v1 and v2", resolver.replace("${k1} and ${k2}"));
    }
}

