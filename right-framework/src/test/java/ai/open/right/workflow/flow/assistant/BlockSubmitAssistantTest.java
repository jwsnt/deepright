package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.block.BlockService;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

/**
 * BlockSubmitAssistant 单元测试类
 */
public class BlockSubmitAssistantTest {

    @Test
    public void test() throws Exception {
        BlockService blockService = EasyMock.createMock(BlockService.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        blockService.submit(workflowTask);
        EasyMock.replay(blockService);
        
        BlockSubmitAssistant blockSubmitAssistant = new BlockSubmitAssistant() {
            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
                // 模拟实现
            }
        };
        blockSubmitAssistant.setBlockService(blockService);
        blockSubmitAssistant.execute(new WorkflowConfig(), workflowTask);
        
        EasyMock.verify(blockService);
    }

    @Test
    public void testInit() throws Exception {
        BlockService blockService = EasyMock.createMock(BlockService.class);
        EasyMock.replay(blockService);
        
        BlockSubmitAssistant.InitConfig initConfig = new BlockSubmitAssistant.InitConfig();
        initConfig.setBlockService(blockService);
        BlockSubmitAssistant blockSubmitAssistant = initConfig.blockSubmitAssistant();
        
        Assertions.assertEquals(blockService, blockSubmitAssistant.getBlockService());
    }

    @Test
    public void testGetterSetter() {
        BlockService blockService = EasyMock.createMock(BlockService.class);
        
        // 测试 BlockSubmitAssistant 的 getter/setter
        BlockSubmitAssistant assistant = new BlockSubmitAssistant();
        assistant.setBlockService(blockService);
        Assertions.assertEquals(blockService, assistant.getBlockService());

        // 测试 InitConfig 的 getter/setter
        BlockSubmitAssistant.InitConfig initConfig = new BlockSubmitAssistant.InitConfig();
        initConfig.setBlockService(blockService);
        Assertions.assertEquals(blockService, initConfig.getBlockService());
    }

    @Test
    public void testConstants() {
        // 验证 WORKFLOW_NAME 常量
        Assertions.assertEquals("def-block", BlockSubmitAssistant.WORKFLOW_NAME);
    }
}

