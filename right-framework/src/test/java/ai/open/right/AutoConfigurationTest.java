package ai.open.right;

import org.easymock.EasyMockRunner;
import org.junit.Assert;
import org.junit.Test;
import org.junit.runner.RunWith;


@RunWith(EasyMockRunner.class)
public class AutoConfigurationTest {

    @Test
    public void testStaticInitialization() {
        try {
            new AutoConfiguration();
            Assert.assertTrue(true);
        } catch (RuntimeException e) {
            Assert.fail("Static initialization failed: " + e.getMessage());
        }
    }
} 

