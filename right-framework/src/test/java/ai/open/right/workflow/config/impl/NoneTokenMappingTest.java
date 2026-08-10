package ai.open.right.workflow.config.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.config.TokenEntry;
import ai.open.right.workflow.config.TokenMapping;
import org.junit.Assert;
import org.junit.Test;

public class NoneTokenMappingTest {

    @Test
    public void entry_returnsSharedNoneConstant() throws Exception {
        NoneTokenMapping mapping = new NoneTokenMapping();
        TokenEntry a = mapping.entry(ObjectBuilder.buildWorkflowTask(), "any-token");
        TokenEntry b = mapping.entry(ObjectBuilder.buildWorkflowTask(), "other");
        Assert.assertSame(NoneTokenMapping.NONE, a);
        Assert.assertSame(a, b);
    }

    @Test
    public void entry_noneEntry_hasNullWorkflowAndBiz() throws Exception {
        NoneTokenMapping mapping = new NoneTokenMapping();
        TokenEntry entry = mapping.entry(ObjectBuilder.buildWorkflowTask(), "ignored");
        Assert.assertNull(entry.getWorkflow());
        Assert.assertNull(entry.getBiz());
    }

    @Test
    public void name_constant() {
        Assert.assertEquals("token.none", NoneTokenMapping.NAME);
    }

    @Test
    public void initConfig_createsNoneTokenMappingBean() throws Exception {
        NoneTokenMapping.InitConfig initConfig = new NoneTokenMapping.InitConfig();
        TokenMapping bean = initConfig.tokenMapping();
        Assert.assertTrue(bean instanceof NoneTokenMapping);
        Assert.assertSame(NoneTokenMapping.NONE, bean.entry(ObjectBuilder.buildWorkflowTask(), "x"));
    }
}
