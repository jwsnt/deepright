package ai.open.right.config;

import ai.open.right.release.ResourceReleaser;
import org.junit.Assert;
import org.junit.Test;

public class ResourceConfigInitConfigTest {

    @Test
    public void shouldCreateResourceConfigBean() throws Exception {
        ResourceReleaser.InitConfig init = new ResourceReleaser.InitConfig();
        ResourceReleaser bean = init.resourceConfig();
        Assert.assertNotNull(bean);
    }
}
