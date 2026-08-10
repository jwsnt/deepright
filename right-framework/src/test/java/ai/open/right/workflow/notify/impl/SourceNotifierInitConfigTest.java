package ai.open.right.workflow.notify.impl;

import org.junit.Assert;
import org.junit.Test;

public class SourceNotifierInitConfigTest {

    @Test
    public void shouldCreateSourceNotifier() throws Exception {
        SourceNotifier.InitConfig init = new SourceNotifier.InitConfig();

        SourceNotifier bean = init.sourceNotifier();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof SourceNotifier);
    }
}
