package ai.open.right.workflow.config.impl;

import ai.open.right.workflow.notify.NotifierService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class DyPromptServiceInitConfigTest {
    @Test
    public void testInit() throws Exception {
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        EasyMock.replay(notifierService);
        DyPromptService.InitConfig initConfig = new DyPromptService.InitConfig();
        initConfig.setNotifierService(notifierService);
        initConfig.setTimeout(10086);
        DyPromptService empty = DyPromptService.class.cast(initConfig.dyPromptService());
        Assert.assertEquals(Integer.valueOf(10086), empty.getTimeout());
        Assert.assertEquals(notifierService, empty.getNotifierService());
        EasyMock.verify(notifierService);
    }
}
