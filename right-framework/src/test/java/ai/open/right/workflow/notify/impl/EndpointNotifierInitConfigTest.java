package ai.open.right.workflow.notify.impl;

import org.junit.Assert;
import org.junit.Test;

public class EndpointNotifierInitConfigTest {

    @Test
    public void shouldCreateEndpointNotifier() throws Exception {
        EndpointNotifier.InitConfig init = new EndpointNotifier.InitConfig();

        EndpointNotifier bean = init.endpointNotifier();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof EndpointNotifier);
    }
}
