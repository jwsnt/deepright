package ai.open.right.workflow.config;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.config.impl.DyTokenMapping;
import ai.open.right.workflow.config.impl.TokenMappingImpl;
import ai.open.right.workflow.flow.WorkflowTask;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class TokenManagerTest {

    @Test
    public void test() throws Exception {
        TokenMapping tokenMapping = EasyMock.createMock(TokenMapping.class);
        TokenEntry tokenEntry = TokenEntry.builder().build();
        WorkflowTask work = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(tokenMapping.entry(work, "HELLO")).andReturn(tokenEntry).anyTimes();
        EasyMock.replay(tokenMapping);
        Map<String, TokenMapping> tokenMappingMap = new HashMap<>();
        tokenMappingMap.put("A", tokenMapping);
        TokenMappingImpl tokenManager = new TokenMappingImpl();
        tokenManager.setTokenMapping(tokenMappingMap);
        tokenManager.setInstance("A");
        Assert.assertEquals(tokenEntry, tokenManager.entry(work, "HELLO"));
        EasyMock.verify(tokenMapping);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testWithInvalidInstance() throws Exception {
        TokenMapping tokenMapping = EasyMock.createMock(TokenMapping.class);
        TokenEntry tokenEntry = TokenEntry.builder().build();
        EasyMock.expect(tokenMapping.entry(ObjectBuilder.buildWorkflowTask(), "HELLO")).andReturn(tokenEntry).anyTimes();
        EasyMock.replay(tokenMapping);
        Map<String, TokenMapping> tokenMappingMap = new HashMap<>();
        tokenMappingMap.put("A", tokenMapping);
        TokenMappingImpl tokenManager = new TokenMappingImpl();
        tokenManager.setTokenMapping(tokenMappingMap);
        tokenManager.setInstance("B");
        try {
            tokenManager.entry(ObjectBuilder.buildWorkflowTask(), "HELLO");
        } finally {
            EasyMock.verify(tokenMapping);
        }
    }

    @Test
    public void testDefault() throws Exception {
        TokenMapping tokenMapping = EasyMock.createMock(TokenMapping.class);
        WorkflowTask work = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(tokenMapping.entry(work, "biz@workflow")).andReturn(TokenEntry.builder().workflow("workflow").biz("biz").build()).anyTimes();
        EasyMock.replay(tokenMapping);
        Map<String, TokenMapping> tokenMappingMap = new HashMap<>();
        tokenMappingMap.put("A", tokenMapping);
        TokenMappingImpl tokenManager = new TokenMappingImpl();
        tokenManager.setDefMapping(new DyTokenMapping());
        tokenManager.setTokenMapping(tokenMappingMap);
        tokenManager.setInstance("A");
        TokenEntry result = tokenManager.entry(work, "biz@workflow");
        Assert.assertEquals("workflow", result.getWorkflow());
        Assert.assertEquals("biz", result.getBiz());
        EasyMock.verify(tokenMapping);
    }
}
