package ai.open.right.workflow.flow.script.impl;

import ai.open.right.workflow.notify.NotifierService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.concurrent.ExecutorService;

public class JavaScriptServiceInitConfigTest {

    @Test
    public void shouldCreateJavaScriptService() throws Exception {
        JavaScriptService.InitConfig init = new JavaScriptService.InitConfig();
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
        JavaScriptService bean = init.javaScriptService();
        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof JavaScriptService);
        Assert.assertEquals(executorService, bean.getExecutorService());
        Assert.assertEquals(notifierService, bean.getNotifierService());
        Assert.assertEquals(timeout4Condition, bean.getTimeout4Condition());
        Assert.assertEquals(timeout4Corrector, bean.getTimeout4Corrector());
        Assert.assertEquals(timeout, bean.getTimeout());
        EasyMock.verify(notifierService, executorService);
    }
}
