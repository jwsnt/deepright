package ai.open.right.workflow.config.impl;

import ai.open.right.workflow.config.TokenMapping;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class TokenMappingImplInitConfigTest {

    @Test
    public void testInit() throws Exception {
        TokenMappingImpl defToken = new TokenMappingImpl();
        Map<String, TokenMapping> tokenMapping = new HashMap<>();
        TokenMappingImpl.InitConfig initConfig = new TokenMappingImpl.InitConfig();
        initConfig.setTokenMapping(tokenMapping);
        initConfig.setDefMapping(defToken);
        initConfig.setInstance("INSTANCE");
        TokenMappingImpl empty = (TokenMappingImpl) initConfig.tokenManager();
        Assert.assertSame(defToken, empty.getDefMapping());
        Assert.assertSame(tokenMapping, empty.getTokenMapping());
        Assert.assertEquals("INSTANCE", empty.getInstance());
    }
}
