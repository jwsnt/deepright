package ai.open.right.workflow.config.impl;

import ai.open.right.workflow.config.Prompt;
import ai.open.right.workflow.config.PromptSearch;
import ai.open.right.workflow.config.PromptService;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

public class PromptServiceImplTest {

    @Test
    public void testGet() throws Exception {
        PromptService delegate = EasyMock.createMock(PromptService.class);
        PromptSearch search = EasyMock.createMock(PromptSearch.class);
        Prompt prompt = EasyMock.createMock(Prompt.class);
        LLMConfig llmConfig = EasyMock.createMock(LLMConfig.class);
        
        EasyMock.expect(search.getLlmConfig()).andReturn(llmConfig).anyTimes();
        EasyMock.expect(llmConfig.hasDynamicPrompt()).andReturn(false).anyTimes();
        EasyMock.expect(delegate.get(search)).andReturn(prompt).once();
        EasyMock.replay(delegate, search, prompt, llmConfig);
        
        PromptServiceImpl service = new PromptServiceImpl();
        Map<String, PromptService> services = new HashMap<>();
        services.put("test", delegate);
        service.setPromptService(services);
        service.setInstance("test");
        
        Assertions.assertEquals(prompt, service.get(search));
        EasyMock.verify(delegate, search, prompt, llmConfig);
    }

    @Test
    public void testInitConfig() throws Exception {
        PromptServiceImpl.InitConfig config = new PromptServiceImpl.InitConfig();
        Map<String, PromptService> services = new HashMap<>();
        config.setPromptService(services);
        config.setInstance("test");
        Assertions.assertNotNull(config.promptService());
    }

    /**
     * buildDynamic：promptService 为 null 时与生产代码一致触发 Assert.notNull
     */
    @Test
    public void testBuildDynamicFailsWhenPromptServiceMapNull() {
        PromptServiceImpl service = new PromptServiceImpl();
        service.setPromptService(null);
        IllegalArgumentException ex = Assertions.assertThrows(IllegalArgumentException.class, service::buildDynamic);
        Assertions.assertTrue(ex.getMessage().contains("The dynamic prompt can not be empty"));
    }

    /**
     * buildDynamic：从 Map 中取 DyPromptService.NAME 对应实现并返回
     */
    @Test
    public void testBuildDynamicReturnsDyPromptDelegate() throws Exception {
        PromptService dynamicDelegate = EasyMock.createMock(PromptService.class);
        EasyMock.replay(dynamicDelegate);
        PromptServiceImpl service = new PromptServiceImpl();
        Map<String, PromptService> services = new HashMap<>();
        services.put(DyPromptService.NAME, dynamicDelegate);
        service.setPromptService(services);
        Assertions.assertSame(dynamicDelegate, service.buildDynamic());
        EasyMock.verify(dynamicDelegate);
    }
}
