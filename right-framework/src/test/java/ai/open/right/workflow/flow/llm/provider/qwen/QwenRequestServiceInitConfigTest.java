package ai.open.right.workflow.flow.llm.provider.qwen;

import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.flow.llm.config.LLMPromptService;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.mcp.client.McpClientService;
import ai.open.right.workflow.mcp.client.rewrtier.McpRewriteService;
import ai.open.right.workflow.mcp.client.trigger.McpTriggerService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class QwenRequestServiceInitConfigTest {

    @Test
    public void testInit() throws Exception {
        McpRewriteService mcpRewriteService = EasyMock.createMock(McpRewriteService.class);
        McpTriggerService mcpTriggerService = EasyMock.createMock(McpTriggerService.class);
        McpClientService mcpClientService = EasyMock.createMock(McpClientService.class);
        LLMPromptService llmPromptService = EasyMock.createMock(LLMPromptService.class);
        NamesService namesService = EasyMock.createMock(NamesService.class);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        Integer timeout = 100086;
        EasyMock.replay(mcpRewriteService, mcpTriggerService, mcpClientService, llmPromptService, namesService, historyStore);
        QwenRequestService.InitConfig initConfig = new QwenRequestService.InitConfig();
        initConfig.setMcpRewriteService(mcpRewriteService);
        initConfig.setMcpTriggerService(mcpTriggerService);
        initConfig.setMcpClientService(mcpClientService);
        initConfig.setLlmPromptService(llmPromptService);
        initConfig.setNamesService(namesService);
        initConfig.setHistoryStore(historyStore);
        initConfig.setFunCallTimeout(timeout);
        initConfig.setToken("TOKEN");
        initConfig.setModel("MODEL");
        QwenRequestService requestService = initConfig.qwenRequestService();
        Assert.assertEquals(mcpRewriteService, requestService.getMcpRewriteService());
        Assert.assertEquals(mcpTriggerService, requestService.getMcpTriggerService());
        Assert.assertEquals(mcpClientService, requestService.getMcpClientService());
        Assert.assertEquals(llmPromptService, requestService.getLlmPromptService());
        Assert.assertEquals(namesService, requestService.getNamesService());
        Assert.assertEquals(historyStore, requestService.getHistoryStore());
        Assert.assertEquals(timeout, requestService.getFunCallTimeout());
        Assert.assertEquals("TOKEN", requestService.getToken());
        Assert.assertEquals("MODEL", requestService.getModel());
        EasyMock.verify(mcpRewriteService, mcpTriggerService, mcpClientService, llmPromptService, namesService, historyStore);
    }
}
