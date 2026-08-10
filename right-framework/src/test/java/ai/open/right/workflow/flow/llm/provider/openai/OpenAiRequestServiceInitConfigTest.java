package ai.open.right.workflow.flow.llm.provider.openai;

import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.flow.llm.config.LLMPromptService;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.mcp.client.McpClientService;
import ai.open.right.workflow.mcp.client.dimension.McpDimensionService;
import ai.open.right.workflow.mcp.client.rewrtier.McpRewriteService;
import ai.open.right.workflow.mcp.client.trigger.McpTriggerService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class OpenAiRequestServiceInitConfigTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = OpenAiRequestService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = OpenAiRequestService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testInit() throws Exception {
        McpDimensionService mcpDimensionService = EasyMock.createMock(McpDimensionService.class);
        McpRewriteService mcpRewriteService = EasyMock.createMock(McpRewriteService.class);
        McpTriggerService mcpTriggerService = EasyMock.createMock(McpTriggerService.class);
        McpClientService mcpClientService = EasyMock.createMock(McpClientService.class);
        LLMPromptService llmPromptService = EasyMock.createMock(LLMPromptService.class);
        NamesService namesService = EasyMock.createMock(NamesService.class);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        Integer timeout = 100086;
        EasyMock.replay(mcpDimensionService, mcpRewriteService, mcpTriggerService, mcpClientService, llmPromptService, namesService, historyStore);
        OpenAiRequestService.InitConfig initConfig = new OpenAiRequestService.InitConfig();
        initConfig.setMcpDimensionService(mcpDimensionService);
        initConfig.setMcpRewriteService(mcpRewriteService);
        initConfig.setMcpTriggerService(mcpTriggerService);
        initConfig.setMcpClientService(mcpClientService);
        initConfig.setLlmPromptService(llmPromptService);
        initConfig.setNamesService(namesService);
        initConfig.setHistoryStore(historyStore);
        initConfig.setFunCallTimeout(timeout);
        OpenAiRequestService requestService = initConfig.openAiRequestService();
        Assert.assertEquals(mcpDimensionService, requestService.getMcpDimensionService());
        Assert.assertEquals(mcpRewriteService, requestService.getMcpRewriteService());
        Assert.assertEquals(mcpTriggerService, requestService.getMcpTriggerService());
        Assert.assertEquals(mcpClientService, requestService.getMcpClientService());
        Assert.assertEquals(llmPromptService, requestService.getLlmPromptService());
        Assert.assertEquals(namesService, requestService.getNamesService());
        Assert.assertEquals(historyStore, requestService.getHistoryStore());
        Assert.assertEquals(timeout, requestService.getFunCallTimeout());
        EasyMock.verify(mcpDimensionService, mcpRewriteService, mcpTriggerService, mcpClientService, llmPromptService, namesService, historyStore);
    }
}
