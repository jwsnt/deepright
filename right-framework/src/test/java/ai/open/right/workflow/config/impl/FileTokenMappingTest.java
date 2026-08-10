package ai.open.right.workflow.config.impl;
import java.util.Collections;
import java.util.Map;
import java.util.HashMap;
import ai.open.right.ObjectBuilder;
import ai.open.right.resouce.PlaceholderResolver;
import ai.open.right.workflow.config.TokenEntry;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.io.FileNotFoundException;

public class FileTokenMappingTest {


    @Test
    public void initTest() {
        PlaceholderResolver placeholderResolver = EasyMock.createMock(PlaceholderResolver.class);
        EasyMock.replay(placeholderResolver);
        FileTokenMapping.InitConfig initConfig = new FileTokenMapping.InitConfig();
        initConfig.setPath("PATH");
        initConfig.setResourceService(ObjectBuilder.buildResourceService());
        initConfig.setPlaceholderResolver(placeholderResolver);
        Assert.assertNotNull(initConfig.getResourceService());
        Assert.assertEquals(placeholderResolver, initConfig.getPlaceholderResolver());
        Assert.assertEquals("PATH", initConfig.getPath());
    }

    @Test
    public void test() throws Exception {
        FileTokenMapping fileTokenMapping = new FileTokenMapping();
        fileTokenMapping.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileTokenMapping.setPath("classpath:Token_Mapping.json");
        fileTokenMapping.setResourceService(ObjectBuilder.buildResourceService());
        fileTokenMapping.init();
        TokenEntry tokenEntry = fileTokenMapping.entry(ObjectBuilder.buildWorkflowTask(), "sk-8fee4359303d4c31835f63437a6e46bf");
        Assert.assertNotNull(fileTokenMapping.getResourceService());
        Assert.assertEquals("example/example80", tokenEntry.getBiz());
        Assert.assertEquals("workflow1", tokenEntry.getWorkflow());
    }

    @Test
    public void testWithNull() throws Exception {
        FileTokenMapping fileTokenMapping = new FileTokenMapping();
        fileTokenMapping.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileTokenMapping.setPath("classpath:Token_Mapping.json");
        fileTokenMapping.setResourceService(ObjectBuilder.buildResourceService());
        fileTokenMapping.init();
        Assert.assertNull(fileTokenMapping.entry(ObjectBuilder.buildWorkflowTask(), "sk-8fee4359303d4c31835f63437a6e46b1"));
    }

    @Test(expected = FileNotFoundException.class)
    public void testWithException() throws Exception {
        FileTokenMapping fileTokenMapping = new FileTokenMapping();
        fileTokenMapping.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileTokenMapping.setResourceService(ObjectBuilder.buildResourceService());
        fileTokenMapping.setPath("classpath:Token_Mapping_1.json");
        fileTokenMapping.init();
        Assert.assertNull(fileTokenMapping.entry(ObjectBuilder.buildWorkflowTask(), "sk-8fee4359303d4c31835f63437a6e46bf"));
    }

    @Test
    public void testWithNullFile() throws Exception {
        FileTokenMapping fileTokenMapping = new FileTokenMapping();
        fileTokenMapping.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileTokenMapping.init();
        Assert.assertNull(fileTokenMapping.entry(ObjectBuilder.buildWorkflowTask(), "sk-8fee4359303d4c31835f63437a6e46b1"));
    }
    @Test
    public void testEntryEmptyMapping() throws Exception {
        FileTokenMapping mapping = new FileTokenMapping();
        mapping.setMapping(Collections.emptyMap());
        Assert.assertNull(mapping.entry(ObjectBuilder.buildWorkflowTask(), "TOKEN"));
    }

    @Test
    public void testEntryWhitespace() throws Exception {
        FileTokenMapping mapping = new FileTokenMapping();
        Map<String, Map<String, String>> map = new HashMap<>();
        map.put("TOKEN", Collections.singletonMap("workflow", "W"));
        mapping.setMapping(map);
        Assert.assertNotNull(mapping.entry(ObjectBuilder.buildWorkflowTask(), "  TOKEN  "));
    }
}
