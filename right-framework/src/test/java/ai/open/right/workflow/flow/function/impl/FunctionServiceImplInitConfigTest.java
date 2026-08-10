package ai.open.right.workflow.flow.function.impl;

import ai.open.right.workflow.flow.function.Function;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class FunctionServiceImplInitConfigTest {
    @Test
    public void initTest() throws Exception {
        Map<String, Function> functions = new HashMap<>();
        FunctionServiceImpl.InitConfig initConfig = new FunctionServiceImpl.InitConfig();
        initConfig.setFunctions(functions);
        FunctionServiceImpl empty = (FunctionServiceImpl) initConfig.functionService();
        Assert.assertEquals(functions, empty.getFunctions());
    }
}
