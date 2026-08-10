package ai.open.right.workflow.flow.function.impl;

import org.junit.Assert;
import org.junit.Test;

public class PythonFunctionInitConfigTest {

    @Test
    public void shouldCreatePythonFunction() throws Exception {
        PythonFunction.InitConfig init = new PythonFunction.InitConfig();

        PythonFunction bean = init.pythonFunction();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof PythonFunction);
    }
}
