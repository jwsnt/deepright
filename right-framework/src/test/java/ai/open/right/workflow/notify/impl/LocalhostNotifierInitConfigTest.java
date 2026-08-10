package ai.open.right.workflow.notify.impl;

import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import ai.open.right.listener.impl.EventListenerServiceImpl;
import ai.open.right.workflow.flow.WorkflowQueue;

public class LocalhostNotifierInitConfigTest {

    @Test
    public void shouldCreateLocalhostNotifierWithProvidedProperties() throws Exception {
        LocalhostNotifier.InitConfig init = new LocalhostNotifier.InitConfig();

        EventListenerServiceImpl eventListenerService = EasyMock.createMock(EventListenerServiceImpl.class);
        WorkflowQueue workflowQueue = EasyMock.createMock(WorkflowQueue.class);

        EasyMock.replay(eventListenerService, workflowQueue);

        // 设置属性
        init.setEventListenerService(eventListenerService);
        init.setWorkflowQueue(workflowQueue);

        LocalhostNotifier bean = init.localhostNotifier();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof LocalhostNotifier);

        EasyMock.verify(eventListenerService, workflowQueue);
    }

    @Test
    public void shouldCreateLocalhostNotifierWithDefaults() throws Exception {
        LocalhostNotifier.InitConfig init = new LocalhostNotifier.InitConfig();

        LocalhostNotifier bean = init.localhostNotifier();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof LocalhostNotifier);
    }
}
