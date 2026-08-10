package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.flow.llm.config.LLMPromptService;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.mcp.client.McpClientService;
import ai.open.right.workflow.mcp.client.rewrtier.McpRewriteService;
import ai.open.right.workflow.mcp.client.trigger.McpTriggerService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class VertexRequestServiceInitConfigTest {

    @Test
    public void testInit() throws Exception {
        VertexTokenExchange vertexTokenExchange = EasyMock.createMock(VertexTokenExchange.class);
        McpRewriteService mcpRewriteService = EasyMock.createMock(McpRewriteService.class);
        McpTriggerService mcpTriggerService = EasyMock.createMock(McpTriggerService.class);
        McpClientService mcpClientService = EasyMock.createMock(McpClientService.class);
        LLMPromptService llmPromptService = EasyMock.createMock(LLMPromptService.class);
        NamesService namesService = EasyMock.createMock(NamesService.class);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        Integer timeout = 100086;
        EasyMock.replay(vertexTokenExchange, mcpRewriteService, mcpTriggerService, mcpClientService, llmPromptService, namesService, historyStore);
        VertexRequestService.InitConfig initConfig = new VertexRequestService.InitConfig();
        initConfig.setMcpRewriteService(mcpRewriteService);
        initConfig.setMcpTriggerService(mcpTriggerService);
        initConfig.setMcpClientService(mcpClientService);
        initConfig.setLlmPromptService(llmPromptService);
        initConfig.setNamesService(namesService);
        initConfig.setHistoryStore(historyStore);
        initConfig.setFunCallTimeout(timeout);
        initConfig.setTokenUri("FILE");
        initConfig.setVertexTokenExchange(vertexTokenExchange);
        VertexRequestService requestService = initConfig.vertexRequestService();
        Assert.assertEquals(mcpRewriteService, requestService.getMcpRewriteService());
        Assert.assertEquals(mcpTriggerService, requestService.getMcpTriggerService());
        Assert.assertEquals(mcpClientService, requestService.getMcpClientService());
        Assert.assertEquals(llmPromptService, requestService.getLlmPromptService());
        Assert.assertEquals(namesService, requestService.getNamesService());
        Assert.assertEquals(historyStore, requestService.getHistoryStore());
        Assert.assertEquals(timeout, requestService.getFunCallTimeout());
        Assert.assertEquals(vertexTokenExchange, requestService.getVertexTokenExchange());
        Assert.assertEquals("FILE", requestService.getTokenUri());
        EasyMock.verify(vertexTokenExchange, mcpRewriteService, mcpTriggerService, mcpClientService, llmPromptService, namesService, historyStore);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = VertexRequestService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = VertexRequestService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
