package ai.open.right.workflow.flow.script.impl;

import ai.open.right.workflow.notify.NotifierService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.concurrent.ExecutorService;

public class PythonServiceInitConfigTest {

    @Test
    public void shouldCreatePythonService() throws Exception {
        PythonService.InitConfig init = new PythonService.InitConfig();
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        Integer timeout4Condition = 10086;
        Integer timeout4Corrector = 10087;
        Integer timeout = 10088;
        EasyMock.replay(notifierService);
        init.setNotifierService(notifierService);
        init.setTimeout4Condition(timeout4Condition);
        init.setTimeout4Corrector(timeout4Corrector);
        init.setTimeout(timeout);
        PythonService bean = init.pythonService();
        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof PythonService);
        Assert.assertEquals(notifierService, bean.getNotifierService());
        Assert.assertEquals(timeout4Condition, bean.getTimeout4Condition());
        Assert.assertEquals(timeout4Corrector, bean.getTimeout4Corrector());
        Assert.assertEquals(timeout, bean.getTimeout());
        EasyMock.verify(notifierService);
    }
}
