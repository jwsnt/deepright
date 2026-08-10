package ai.open.right.resouce.impl;

import ai.open.right.WorkflowException;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.ApplicationContext;

public class ResourceServiceImplTest {

    @Test
    public void test() throws Exception {
        SpringBootApplication springBootApplication = EasyMock.createMock(SpringBootApplication.class);
        ApplicationContext applicationContext = EasyMock.createMock(ApplicationContext.class);
        EasyMock.expect(applicationContext.getBeanNamesForAnnotation(SpringBootApplication.class)).andReturn(new String[]{SpringBootApplication.class.getSimpleName()}).anyTimes();
        EasyMock.expect(applicationContext.getBean("SpringBootApplication")).andReturn(springBootApplication).anyTimes();
        EasyMock.replay(applicationContext, springBootApplication);
        ResourceServiceImpl resourceService = new ResourceServiceImpl();
        resourceService.setApplicationContext(applicationContext);
        resourceService.init();
        Assert.assertTrue(resourceService.url("classpath:A2A.json").toString().contains("/target/test-classes/A2A.json"));
        Assert.assertNotNull(resourceService.getResourceResolver());
        Assert.assertNotNull(resourceService.root());
        Assert.assertNotNull(resourceService.getRootClass());
    }

    @Test(expected = WorkflowException.class)
    public void testException() throws Exception {
        SpringBootApplication springBootApplication = EasyMock.createMock(SpringBootApplication.class);
        ApplicationContext applicationContext = EasyMock.createMock(ApplicationContext.class);
        EasyMock.expect(applicationContext.getBeanNamesForAnnotation(SpringBootApplication.class)).andReturn(new String[]{SpringBootApplication.class.getSimpleName()}).anyTimes();
        EasyMock.expect(applicationContext.getBean("SpringBootApplication")).andReturn(springBootApplication).anyTimes();
        EasyMock.replay(applicationContext, springBootApplication);
        ResourceServiceImpl resourceService = new ResourceServiceImpl();
        resourceService.setApplicationContext(applicationContext);
        resourceService.init();
        resourceService.url("ABC");
        EasyMock.verify(applicationContext, springBootApplication);
    }

    @Test
    public void testInit() throws Exception {
        SpringBootApplication springBootApplication = EasyMock.createMock(SpringBootApplication.class);
        ApplicationContext applicationContext = EasyMock.createMock(ApplicationContext.class);
        EasyMock.expect(applicationContext.getBeanNamesForAnnotation(SpringBootApplication.class)).andReturn(new String[]{SpringBootApplication.class.getSimpleName()}).anyTimes();
        EasyMock.expect(applicationContext.getBean("SpringBootApplication")).andReturn(springBootApplication).anyTimes();
        EasyMock.replay(applicationContext, springBootApplication);
        ResourceServiceImpl.InitConfig init = new ResourceServiceImpl.InitConfig();
        init.setApplicationContext(applicationContext);
        Assert.assertEquals(applicationContext, ResourceServiceImpl.class.cast(init.resourceService()).getApplicationContext());
        Assert.assertNotNull(init.resourceService());
        EasyMock.verify(applicationContext, springBootApplication);
    }
}
