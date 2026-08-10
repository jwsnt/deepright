package ai.open.right.netty.chat.server.http;

import org.junit.Assert;
import org.junit.Test;

public class NettyCorsHandlerInitConfigTest {

    @Test
    public void shouldCreateNettyCorsHandlerWithProvidedProperties() throws Exception {
        NettyCorsHandler.InitConfig init = new NettyCorsHandler.InitConfig();

        NettyCorsHandler bean = init.nettyCorsHandler();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof NettyCorsHandler);
    }

    @Test
    public void shouldCreateNettyCorsHandlerWithDefaults() throws Exception {
        NettyCorsHandler.InitConfig init = new NettyCorsHandler.InitConfig();
        NettyCorsHandler bean = init.nettyCorsHandler();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof NettyCorsHandler);
    }
}
