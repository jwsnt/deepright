package ai.open.right.workflow.config.impl;

import ai.open.right.resouce.PlaceholderResolver;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class FilePromptServiceInitConfigTest {

    @Test
    public void testInit() throws Exception {
        PlaceholderResolver placeholderResolver = EasyMock.createMock(PlaceholderResolver.class);
        EasyMock.replay(placeholderResolver);
        FilePromptService.InitConfig initConfig = new FilePromptService.InitConfig();
        initConfig.setPath("PATH");
        initConfig.setSuffix("SUFFIX");
        initConfig.setPlaceholderResolver(placeholderResolver);
        FilePromptService empty = (FilePromptService) initConfig.promptService();
        Assert.assertEquals(initConfig.getPath(), empty.getPath());
        Assert.assertEquals(placeholderResolver, empty.getPlaceholderResolver());
        Assert.assertEquals(initConfig.getSuffix(), empty.getSuffix());
        EasyMock.verify(placeholderResolver);
    }
}
