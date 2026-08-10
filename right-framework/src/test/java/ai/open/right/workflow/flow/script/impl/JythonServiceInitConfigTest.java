package ai.open.right.workflow.flow.script.impl;

import ai.open.right.workflow.notify.NotifierService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.concurrent.ExecutorService;

public class JythonServiceInitConfigTest {

    @Test
    public void shouldCreateJythonService() throws Exception {
        JythonService.InitConfig init = new JythonService.InitConfig();
        ExecutorService executorService = EasyMock.createMock(ExecutorService.class);
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        Integer timeout4Condition = 10086;
        Integer timeout4Corrector = 10087;
        Integer timeout = 10088;
        EasyMock.replay(notifierService, executorService);
        init.setExecutorService(executorService);
        init.setNotifierService(notifierService);
        init.setTimeout4Condition(timeout4Condition);
        init.setTimeout4Corrector(timeout4Corrector);
        init.setTimeout(timeout);
        JythonService bean = init.jythonService();
        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof JythonService);
        Assert.assertEquals(executorService, bean.getExecutorService());
        Assert.assertEquals(notifierService, bean.getNotifierService());
        Assert.assertEquals(timeout4Condition, bean.getTimeout4Condition());
        Assert.assertEquals(timeout4Corrector, bean.getTimeout4Corrector());
        Assert.assertEquals(timeout, bean.getTimeout());
        EasyMock.verify(notifierService, executorService);
        Assert.assertNull(bean.getObjectPool());
        bean.init();
        Assert.assertNotNull(bean.getObjectPool());
        bean.destroy();
    }
}
