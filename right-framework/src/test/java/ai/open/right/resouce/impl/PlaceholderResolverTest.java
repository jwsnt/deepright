package ai.open.right.resouce.impl;

import ai.open.right.resouce.PlaceholderResolver;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

public class PlaceholderResolverTest {

    @Test
    public void testAnonymousResolver() throws Exception {
        PlaceholderResolver resolver = input -> input.replace("{{name}}", "world");
        Assertions.assertEquals("hello world", resolver.replace("hello {{name}}"));
    }
}
