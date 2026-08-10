package ai.open.right.workflow.config.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.config.TokenEntry;
import ai.open.right.workflow.config.TokenMapping;
import ai.open.right.workflow.flow.WorkflowTask;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.Map;

/**
 * {@link TokenMappingImpl} 单测：避免对 {@link TokenEntry}/{@link TokenMapping} 使用 EasyMock，
 * 以兼容新版 JDK 的类注入限制。
 */
public class TokenMappingImplTest {

    private static TokenMapping returnsEntry(TokenEntry entry) {
        return new NoneTokenMapping() {
            @Override
            public TokenEntry entry(WorkflowTask workTask, String token) throws Exception {
                return entry;
            }
        };
    }

    private static TokenMapping returnsNull() {
        return new NoneTokenMapping();
    }

    @Test
    void entry_usesMappingForTrimmedInstanceKey() throws Exception {
        TokenEntry expected = TokenEntry.builder().workflow("wf").biz("bz").build();
        TokenMappingImpl service = new TokenMappingImpl();
        Map<String, TokenMapping> mappings = new HashMap<>();
        mappings.put("test", returnsEntry(expected));
        service.setTokenMapping(mappings);
        service.setInstance("  test  ");
        service.init();
        Assertions.assertSame(expected, service.entry(ObjectBuilder.buildWorkflowTask(), "any-token"));
    }

    @Test
    void entry_whenInstanceKeyMissing_usesDefMapping() throws Exception {
        TokenEntry fromDef = TokenEntry.builder().workflow("defWf").build();
        TokenMappingImpl service = new TokenMappingImpl();
        Map<String, TokenMapping> mappings = new HashMap<>();
        mappings.put("other", returnsEntry(TokenEntry.builder().biz("unused").build()));
        service.setTokenMapping(mappings);
        service.setInstance("missing-key");
        service.setDefMapping(returnsEntry(fromDef));
        service.init();
        Assertions.assertSame(fromDef, service.entry(ObjectBuilder.buildWorkflowTask(), "token"));
    }

    @Test
    void entry_whenInstanceKeyMissing_usesFirstMappingIfDefMappingNull() throws Exception {
        TokenEntry fromFirst = TokenEntry.builder().biz("first").build();
        TokenMapping first = returnsEntry(fromFirst);
        TokenMappingImpl service = new TokenMappingImpl();
        Map<String, TokenMapping> map = new LinkedHashMap<>();
        map.put("only", first);
        service.setTokenMapping(map);
        service.setInstance("not-in-map");
        service.setDefMapping(null);
        service.init();
        Assertions.assertSame(first, service.getFirstMapping());
        Assertions.assertSame(fromFirst, service.entry(ObjectBuilder.buildWorkflowTask(), "tok"));
    }

    @Test
    void entry_whenConfiguredDelegateReturnsNull_throws() throws Exception {
        TokenMappingImpl service = new TokenMappingImpl();
        Map<String, TokenMapping> map = new HashMap<>();
        service.setTokenMapping(map);
        service.setInstance("test");
        service.init();
        try {
            service.entry(ObjectBuilder.buildWorkflowTask(), "t");
            Assertions.fail("expected IllegalArgumentException");
        } catch (IllegalArgumentException ex) {
            Assertions.assertTrue(ex.getMessage().contains("The final mapping can not be empty"));
        }
    }

    @Test
    void entry_whenNoDelegateAndNoFinalMapping_throws() throws Exception {
        TokenMappingImpl service = new TokenMappingImpl();
        service.setTokenMapping(new HashMap<>());
        service.setInstance("x");
        service.setDefMapping(null);
        service.init();
        try {
            service.entry(ObjectBuilder.buildWorkflowTask(), "t");
            Assertions.fail("expected IllegalArgumentException");
        } catch (IllegalArgumentException ex) {
            Assertions.assertTrue(ex.getMessage().contains("final mapping"));
        }
    }

    @Test
    void init_setsFirstMappingToFirstValueInInsertionOrder() throws Exception {
        TokenMapping m1 = new TokenMappingImpl();
        TokenMapping m2 = new TokenMappingImpl();
        TokenMappingImpl service = new TokenMappingImpl();
        Map<String, TokenMapping> map = new LinkedHashMap<>();
        map.put("a", m1);
        map.put("b", m2);
        service.setTokenMapping(map);
        service.init();
        Assertions.assertSame(m1, service.getFirstMapping());
    }

    @Test
    void init_emptyTokenMapping_leavesFirstMappingNull() throws Exception {
        TokenMappingImpl service = new TokenMappingImpl();
        service.setTokenMapping(new HashMap<>());
        service.init();
        Assertions.assertNull(service.getFirstMapping());
    }

    @Test
    void init_nullTokenMapping_leavesFirstMappingNull() throws Exception {
        TokenMappingImpl service = new TokenMappingImpl();
        service.setTokenMapping(null);
        service.init();
        Assertions.assertNull(service.getFirstMapping());
    }

    @Test
    void name_constant() {
        Assertions.assertEquals("TokenMappingImpl", TokenMappingImpl.NAME);
    }

    @Test
    void initConfig_createsBean() throws Exception {
        TokenMappingImpl.InitConfig config = new TokenMappingImpl.InitConfig();
        config.setTokenMapping(new HashMap<>());
        config.setDefMapping(new NoneTokenMapping());
        config.setInstance("test");
        TokenMapping bean = config.tokenManager();
        Assertions.assertNotNull(bean);
        Assertions.assertTrue(bean instanceof TokenMappingImpl);
        Assertions.assertEquals("test", ((TokenMappingImpl) bean).getInstance());
    }
}
