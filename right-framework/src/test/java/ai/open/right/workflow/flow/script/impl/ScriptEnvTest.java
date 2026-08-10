package ai.open.right.workflow.flow.script.impl;

import ai.open.right.ObjectBuilder;
import org.junit.Assert;
import org.junit.Test;

public class ScriptEnvTest {

    @Test
    public void testData() throws Exception {
        ScriptEnv scriptEnv = new ScriptEnv(ObjectBuilder.buildWorkflowTask());
        scriptEnv.env(ObjectBuilder.buildEvent());
        Assert.assertEquals("UNKNOWN", scriptEnv.get(ScriptEnv.KEY_DATA));
    }
}
