package ai.open.right.workflow.ratelimit.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import org.junit.Assert;
import org.junit.Test;

public class GuavaRateLimitServiceTest {

    @Test
    public void test() throws Exception {
        GuavaRateLimitService guavaRateLimitService = new GuavaRateLimitService();
        guavaRateLimitService.setLimit(1024);
        guavaRateLimitService.init();
        guavaRateLimitService.checkLimit(ObjectBuilder.buildWorkflowTask());
        Assert.assertNotNull(guavaRateLimitService.getRateLimiter());
    }

    @Test(expected = WorkflowException.class)
    public void testFailed() throws Exception {
        GuavaRateLimitService guavaRateLimitService = new GuavaRateLimitService();
        guavaRateLimitService.setLimit(1);
        guavaRateLimitService.init();
        guavaRateLimitService.checkLimit(ObjectBuilder.buildWorkflowTask());
        guavaRateLimitService.checkLimit(ObjectBuilder.buildWorkflowTask());
    }

    @Test
    public void testInit() throws Exception {
        GuavaRateLimitService.InitConfig initConfig = new GuavaRateLimitService.InitConfig();
        initConfig.setLimit(1024);
        GuavaRateLimitService guavaRateLimitService = (GuavaRateLimitService) initConfig.rateLimitService();
        Assert.assertEquals(guavaRateLimitService.getLimit(), initConfig.getLimit());
    }
}
