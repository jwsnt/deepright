package ai.open.right.workflow.flow.llm.provider.xiaomi;

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

public class XiaomiRequestServiceInitConfigTest {

    @Test
    public void testInit() throws Exception {
        McpDimensionService mcpDimensionService = EasyMock.createMock(McpDimensionService.class);
        McpRewriteService mcpRewriteService = EasyMock.createMock(McpRewriteService.class);
        McpTriggerService mcpTriggerService = EasyMock.createMock(McpTriggerService.class);
        McpClientService mcpClientService = EasyMock.createMock(McpClientService.class);
        LLMPromptService llmPromptService = EasyMock.createMock(LLMPromptService.class);
        NamesService namesService = EasyMock.createMock(NamesService.class);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.replay(mcpDimensionService, mcpRewriteService, mcpTriggerService, mcpClientService, llmPromptService, namesService, historyStore);

        XiaomiRequestService.InitConfig initConfig = new XiaomiRequestService.InitConfig();
        initConfig.setMcpDimensionService(mcpDimensionService);
        initConfig.setMcpRewriteService(mcpRewriteService);
        initConfig.setMcpTriggerService(mcpTriggerService);
        initConfig.setMcpClientService(mcpClientService);
        initConfig.setLlmPromptService(llmPromptService);
        initConfig.setNamesService(namesService);
        initConfig.setHistoryStore(historyStore);
        initConfig.setFunCallTimeout(100086);
        initConfig.setModel("MODEL");
        initConfig.setToken("TOKEN");

        XiaomiRequestService requestService = initConfig.xiaomiRequestService();

        Assert.assertEquals(initConfig.getModel(), requestService.getModel());
        Assert.assertEquals(initConfig.getToken(), requestService.getToken());
        Assert.assertEquals(initConfig.getFunCallTimeout(), requestService.getFunCallTimeout());
        Assert.assertEquals(mcpDimensionService, requestService.getMcpDimensionService());
        Assert.assertEquals(mcpRewriteService, requestService.getMcpRewriteService());
        Assert.assertEquals(mcpTriggerService, requestService.getMcpTriggerService());
        Assert.assertEquals(mcpClientService, requestService.getMcpClientService());
        Assert.assertEquals(llmPromptService, requestService.getLlmPromptService());
        Assert.assertEquals(namesService, requestService.getNamesService());
        Assert.assertEquals(historyStore, requestService.getHistoryStore());

        EasyMock.verify(mcpDimensionService, mcpRewriteService, mcpTriggerService, mcpClientService, llmPromptService, namesService, historyStore);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = XiaomiRequestService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = XiaomiRequestService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
