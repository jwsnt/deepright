package ai.open.right.workflow.flow.assistant;

import ai.open.right.workflow.notify.NotifierService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class ChainInitConfigTest {

    @Test
    public void testInit() throws Exception {
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        EasyMock.replay(notifierService);
        ChainAssistant.ChainInitConfig initConfig = new ChainAssistant.ChainInitConfig();
        initConfig.setNotifierService(notifierService);
        Assert.assertEquals(notifierService, initConfig.getNotifierService());
        EasyMock.verify(notifierService);
    }
}
