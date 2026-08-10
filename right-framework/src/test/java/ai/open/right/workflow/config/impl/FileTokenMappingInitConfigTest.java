package ai.open.right.workflow.config.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.resouce.PlaceholderResolver;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

public class FileTokenMappingInitConfigTest {
    @Test
    public void testInit() throws Exception {
        String body = IOUtils.toString(ResourceUtils.getURL("classpath:Token_Mapping.json").openStream());
        PlaceholderResolver placeholderResolver = EasyMock.createMock(PlaceholderResolver.class);
        EasyMock.expect(placeholderResolver.replace(body)).andReturn(body).anyTimes();
        EasyMock.replay(placeholderResolver);
        FileTokenMapping.InitConfig initConfig = new FileTokenMapping.InitConfig();
        initConfig.setPath("classpath:Token_Mapping.json");
        initConfig.setPlaceholderResolver(placeholderResolver);
        initConfig.setResourceService(ObjectBuilder.buildResourceService());
        FileTokenMapping empty = (FileTokenMapping) initConfig.tokenMapping();
        Assert.assertEquals("classpath:Token_Mapping.json", empty.getPath());
        Assert.assertEquals(placeholderResolver, empty.getPlaceholderResolver());
        empty.init();
        Assert.assertNotNull(empty.getMapping());
        EasyMock.verify(placeholderResolver);
    }
}
