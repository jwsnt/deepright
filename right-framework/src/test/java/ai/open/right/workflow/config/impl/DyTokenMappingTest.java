package ai.open.right.workflow.config.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.config.TokenEntry;
import org.junit.Assert;
import org.junit.Test;

public class DyTokenMappingTest {

    @Test
    public void test() throws Exception {
        DyTokenMapping dyTokenMapping = new DyTokenMapping();
        TokenEntry tokenEntry = dyTokenMapping.entry(ObjectBuilder.buildWorkflowTask(), "biz/example@workflow");
        Assert.assertEquals("biz/example", tokenEntry.getBiz());
        Assert.assertEquals("workflow", tokenEntry.getWorkflow());
    }

    @Test(expected = IllegalArgumentException.class)
    public void testException() throws Exception {
        DyTokenMapping dyTokenMapping = new DyTokenMapping();
        dyTokenMapping.entry(ObjectBuilder.buildWorkflowTask(), "biz/example");
    }

    @Test
    public void testInit() throws Exception {
        DyTokenMapping.InitConfig initConfig = new DyTokenMapping.InitConfig();
        Assert.assertNotNull(initConfig.tokenMapping());
    }
}
