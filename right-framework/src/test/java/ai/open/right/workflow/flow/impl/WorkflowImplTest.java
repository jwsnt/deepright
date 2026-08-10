package ai.open.right.workflow.flow.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.assistant.Assistant;
import ai.open.right.workflow.flow.block.BlockService;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.config.WorkflowConfigService;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.trigger.WorkflowTriggerService;
import ai.open.right.workflow.flow.trigger.impl.WorkflowTriggerServiceImpl;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import ai.open.right.workflow.ratelimit.RateLimitService;
import ch.qos.logback.classic.Level;
import ch.qos.logback.classic.Logger;
import ch.qos.logback.classic.spi.ILoggingEvent;
import ch.qos.logback.core.read.ListAppender;
import com.google.common.util.concurrent.RateLimiter;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.slf4j.LoggerFactory;

import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

public class WorkflowImplTest {

    @Test
    public void testRateLimit() throws Exception {
        RateLimitService rateLimitService = EasyMock.createMock(RateLimitService.class);
        rateLimitService.checkLimit(EasyMock.anyObject(WorkflowTask.class));
        EasyMock.expectLastCall().andThrow(new RuntimeException("LIMIT"));
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        WorkflowImpl workflow = new WorkflowImpl();
        workflow.setWorkflowTriggerService(new WorkflowTriggerServiceImpl());
        workflow.setMessageOnFailed(false);
        workflow.setNotifierService(notifierManager);
        workflow.setRateLimitService(rateLimitService);
        EasyMock.replay(rateLimitService);
        try {
            Assertions.assertThrows(RuntimeException.class, () -> workflow.async(ObjectBuilder.buildWorkflowTask()));
        } finally {
            EasyMock.verify(rateLimitService);
        }
    }

    @Test
    public void testInit() throws Exception {
        BlockService blockService = EasyMock.createMock(BlockService.class);
        RateLimitService rateLimitService = EasyMock.createMock(RateLimitService.class);
        rateLimitService.checkLimit(EasyMock.anyObject(WorkflowTask.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(rateLimitService, blockService);
        WorkflowImpl workflow = new WorkflowImpl();
        workflow.setMessageOnFailed(true);
        workflow.setBlockService(blockService);
        workflow.setRateLimitService(rateLimitService);
        Assertions.assertEquals(blockService, workflow.getBlockService());
        Assertions.assertNotNull(workflow.getRateLimitService());
        EasyMock.verify(rateLimitService, blockService);
    }

    @Test
    public void testExecute() throws Exception {
        Assistant assistant = new Assistant() {
            @Override
            public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {

            }

            @Override
            public void config(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {

            }
        };
        Map<String, Assistant> assistants = new HashMap<String, Assistant>();
        assistants.put("DEF", assistant);
        RateLimitService rateLimitService = EasyMock.createMock(RateLimitService.class);
        rateLimitService.checkLimit(EasyMock.anyObject(WorkflowTask.class));
        EasyMock.expectLastCall().anyTimes();
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setAssistant("DEF");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(workflowConfigService.config(workflowTask)).andReturn(workflowConfig).anyTimes();
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        EasyMock.replay(rateLimitService, workflowConfigService);
        WorkflowImpl workflow = new WorkflowImpl();
        workflow.setMessageOnFailed(true);
        workflow.setWorkflowTriggerService(new WorkflowTriggerServiceImpl());
        workflow.setWorkflowConfigService(workflowConfigService);
        workflow.setNotifierService(notifierManager);
        workflow.setRateLimitService(rateLimitService);
        workflow.setAssistant(assistants);
        workflow.setDeepness(3);
        BlockService blockService = EasyMock.createMock(BlockService.class);
        blockService.block(workflowTask);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(blockService);
        workflow.setBlockService(blockService);
        workflow.async(workflowTask);
        EasyMock.verify(rateLimitService, workflowConfigService, blockService);
    }

    @Test
    public void testNotAllowed() throws Exception {
        Assistant assistant = new Assistant() {
            @Override
            public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {

            }

            @Override
            public void config(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {

            }
        };
        Map<String, Assistant> assistants = new HashMap<String, Assistant>();
        assistants.put("DEF", assistant);
        RateLimitService rateLimitService = EasyMock.createMock(RateLimitService.class);
        rateLimitService.checkLimit(EasyMock.anyObject(WorkflowTask.class));
        EasyMock.expectLastCall().anyTimes();
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setAssistant("DEF");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(workflowConfigService.config(workflowTask)).andReturn(workflowConfig).anyTimes();
        EasyMock.replay(rateLimitService, workflowConfigService);
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        WorkflowImpl workflow = new WorkflowImpl();
        workflow.setMessageOnFailed(false);
        workflow.setWorkflowConfigService(workflowConfigService);
        workflow.setWorkflowTriggerService(new WorkflowTriggerServiceImpl());
        workflow.setNotifierService(notifierManager);
        workflow.setAssistant(assistants);
        workflow.setRateLimitService(rateLimitService);
        workflow.setDeepness(0);
        BlockService blockService = EasyMock.createMock(BlockService.class);
        blockService.block(workflowTask);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(blockService);
        workflow.setBlockService(blockService);
        Assertions.assertThrows(WorkflowException.class, () -> workflow.async(workflowTask));
        EasyMock.verify(rateLimitService, workflowConfigService, blockService);
    }

    @Test
    public void testWithOtherException() throws Exception {
        WorkflowImpl workflow = new WorkflowImpl();
        workflow.setWorkflowTriggerService(new WorkflowTriggerServiceImpl());
        workflow.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        workflow.setMessageOnFailed(false);
        Assertions.assertThrows(NullPointerException.class, () -> workflow.async(ObjectBuilder.buildWorkflowTask()));
    }

    @Test
    public void testReConfigWithNullProvider() throws Exception {
        WorkflowImpl workflow = new WorkflowImpl();
        workflow.setMessageOnFailed(true);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setProvider("PROVIDER");
        workflowConfig.setLlmConfig(llmConfig);
        workflow.reConfig(workflowConfig, workflowTask);
        Assertions.assertEquals("PROVIDER", llmConfig.getProvider());
    }

    @Test
    public void testReConfigWithProvider() throws Exception {
        WorkflowImpl workflow = new WorkflowImpl();
        workflow.setMessageOnFailed(true);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.putMetadata(ProviderRequestService.KEY_PROVIDER, "MY_PROVIDER");
        WorkflowConfig workflowConfig = new WorkflowConfig();
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setProvider("PROVIDER");
        workflowConfig.setLlmConfig(llmConfig);
        workflow.reConfig(workflowConfig, workflowTask);
        Assertions.assertEquals("MY_PROVIDER", llmConfig.getProvider());
    }

    @Test
    public void testReConfigWithChatTrack() throws Exception {
        WorkflowImpl workflow = new WorkflowImpl();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        Assertions.assertFalse(workflowTask.containChatTrack());
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setChatTrack(true);
        workflow.reConfig(workflowConfig, workflowTask);
        Assertions.assertTrue(workflowTask.containChatTrack());
    }

    @Test
    public void testExecuteWithNotNotify() throws Exception {
        Assistant assistant = new Assistant() {
            @Override
            public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {

            }

            @Override
            public void config(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {

            }
        };
        Map<String, Assistant> assistants = new HashMap<String, Assistant>();
        assistants.put("DEF", assistant);
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setAssistant("DEF");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(workflowConfigService.config(workflowTask)).andReturn(workflowConfig).anyTimes();
        RateLimitService rateLimitService = EasyMock.createMock(RateLimitService.class);
        rateLimitService.checkLimit(EasyMock.anyObject(WorkflowTask.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(rateLimitService, workflowConfigService);
        WorkflowImpl workflow = new WorkflowImpl();
        workflow.setMessageOnFailed(false);
        workflow.setWorkflowTriggerService(new WorkflowTriggerServiceImpl());
        workflow.setWorkflowConfigService(workflowConfigService);
        workflow.setRateLimitService(rateLimitService);
        workflow.setAssistant(assistants);
        workflow.setDeepness(3);
        BlockService blockService = EasyMock.createMock(BlockService.class);
        blockService.block(workflowTask);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(blockService);
        workflow.setBlockService(blockService);
        workflow.async(workflowTask);
        EasyMock.verify(rateLimitService, workflowConfigService, blockService);
    }

    @Test
    public void testExecuteWithExceptionNotify() throws Exception {
        Assistant assistant = new Assistant() {
            @Override
            public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {

            }

            @Override
            public void config(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {

            }
        };
        Map<String, Assistant> assistants = new HashMap<String, Assistant>();
        assistants.put("DEF", assistant);
        RateLimitService rateLimitService = EasyMock.createMock(RateLimitService.class);
        rateLimitService.checkLimit(EasyMock.anyObject(WorkflowTask.class));
        EasyMock.expectLastCall().anyTimes();
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(workflowConfigService.config(workflowTask)).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(rateLimitService, workflowConfigService);
        WorkflowImpl workflow = new WorkflowImpl();
        workflow.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackException());
        workflow.setMessageOnFailed(false);
        workflow.setWorkflowConfigService(workflowConfigService);
        workflow.setRateLimitService(rateLimitService);
        workflow.setAssistant(assistants);
        workflow.setDeepness(1);
        Assertions.assertThrows(RuntimeException.class, () -> workflow.async(workflowTask));
        EasyMock.verify(rateLimitService, workflowConfigService);
    }

    @Test
    public void testBuild() throws Exception {
        RateLimitService rateLimitService = EasyMock.createMock(RateLimitService.class);
        rateLimitService.checkLimit(EasyMock.anyObject(WorkflowTask.class));
        EasyMock.expectLastCall().anyTimes();
        WorkflowTriggerServiceImpl workflowTriggerManager = EasyMock.createMock(WorkflowTriggerServiceImpl.class);
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        Map<String, Assistant> assistantMap = new HashMap<>();
        RateLimiter rateLimiter = EasyMock.createMock(RateLimiter.class);
        EasyMock.replay(rateLimitService, workflowConfigService, workflowTriggerManager, rateLimiter);
        WorkflowImpl.InitConfig service = new WorkflowImpl.InitConfig();
        service.setNotifierService(notifierManager);
        service.setAssistant(assistantMap);
        service.setWorkflowConfigService(workflowConfigService);
        service.setWorkflowTriggerService(workflowTriggerManager);
        service.setDeepness(10);
        service.setRateLimitService(rateLimitService);
        service.setMessageOnFailed(false);
        WorkflowImpl empty = (WorkflowImpl) service.workflow();
        Assertions.assertEquals(notifierManager, empty.getNotifierService());
        Assertions.assertEquals(assistantMap, empty.getAssistant());
        Assertions.assertNotNull(empty.getRateLimitService());
        Assertions.assertEquals(workflowConfigService, empty.getWorkflowConfigService());
        Assertions.assertEquals(workflowTriggerManager, empty.getWorkflowTriggerService());
        Assertions.assertEquals(Integer.valueOf(10), empty.getDeepness());
        Assertions.assertFalse(empty.getMessageOnFailed());
        EasyMock.verify(rateLimitService, workflowConfigService, workflowTriggerManager, rateLimiter);
    }

    @Test
    public void testAllowedDepth() throws Exception {
        WorkflowImpl service = new WorkflowImpl();
        service.setDeepness(5);
        NettyRequest task = NettyRequest.class.cast(ObjectBuilder.buildWorkflowTask());
        task.setDeepness(4);
        BlockService bs = EasyMock.createMock(BlockService.class);
        bs.block(task);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(bs);
        service.setBlockService(bs);
        Assertions.assertTrue(service.allowed(new WorkflowConfig(), task));
    }

    @Test
    public void testSyncWithNullTask() {
        WorkflowImpl workflow = new WorkflowImpl();
        Assertions.assertThrows(NullPointerException.class, () -> {
            workflow.sync(null);
        });
    }

    @Test
    public void testSyncWithNullConfig() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        // 模拟 workflowConfigService.config 返回 null
        EasyMock.expect(workflowConfigService.config(workflowTask)).andReturn(null);

        RateLimitService rateLimitService = EasyMock.createMock(RateLimitService.class);
        rateLimitService.checkLimit(EasyMock.anyObject(WorkflowTask.class));
        EasyMock.expectLastCall().anyTimes();

        EasyMock.replay(workflowConfigService, rateLimitService);

        WorkflowImpl workflow = new WorkflowImpl();
        workflow.setWorkflowConfigService(workflowConfigService);
        workflow.setRateLimitService(rateLimitService);
        workflow.setMessageOnFailed(false);
        // 触发 Assert.notNull(workflowConfig, "The task config can not be empty");
        Assertions.assertThrows(IllegalArgumentException.class, () -> workflow.sync(workflowTask));
        EasyMock.verify(workflowConfigService, rateLimitService);
    }

    @Test
    public void testSyncWithNullAssistant() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setAssistant("NON_EXISTENT");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(workflowConfigService.config(workflowTask)).andReturn(workflowConfig);

        RateLimitService rateLimitService = EasyMock.createMock(RateLimitService.class);
        rateLimitService.checkLimit(EasyMock.anyObject(WorkflowTask.class));
        EasyMock.expectLastCall().anyTimes();

        WorkflowTriggerService workflowTriggerService = EasyMock.createMock(WorkflowTriggerService.class);
        workflowTriggerService.before(EasyMock.anyObject(WorkflowConfig.class), EasyMock.anyObject(WorkflowTask.class));
        EasyMock.expectLastCall().anyTimes();

        EasyMock.replay(workflowConfigService, rateLimitService, workflowTriggerService);

        WorkflowImpl workflow = new WorkflowImpl();
        workflow.setWorkflowConfigService(workflowConfigService);
        workflow.setRateLimitService(rateLimitService);
        workflow.setWorkflowTriggerService(workflowTriggerService);
        workflow.setAssistant(new HashMap<>()); // 空的 assistant map
        workflow.setMessageOnFailed(false);
        // 触发 Assert.notNull(assistant, "The task assistant can not be empty");
        Assertions.assertThrows(IllegalArgumentException.class, () -> workflow.sync(workflowTask));
        EasyMock.verify(workflowConfigService, rateLimitService, workflowTriggerService);
    }

    @Test
    public void testBlockWithNullService() throws Exception {
        WorkflowImpl workflow = new WorkflowImpl();
        workflow.setBlockService(null);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        // 验证 blockService 为空时 block 方法正常执行（不抛出异常）
        Assertions.assertDoesNotThrow(() -> workflow.block(workflowTask));
    }

    @Test
    public void testAllowedEdgeCase() throws Exception {
        WorkflowImpl service = new WorkflowImpl();
        service.setDeepness(5);
        NettyRequest task = (NettyRequest) ObjectBuilder.buildWorkflowTask();
        // 验证 deepness 相等时 allowed 返回 false
        task.setDeepness(5);
        Assertions.assertFalse(service.allowed(new WorkflowConfig(), task));
    }

    /**
     * 覆盖 WorkflowImpl#into(WorkflowTask)：调用时打 INFO 日志 "waiting time"，且不抛异常
     */
    @Test
    public void testInto_logsWaitingTimeAndDoesNotThrow() throws Exception {
        class WorkflowImplIntoTest extends WorkflowImpl {
            void callInto(WorkflowTask t) throws Exception {
                into(t);
            }
        }
        Logger logger = (Logger) LoggerFactory.getLogger(WorkflowImpl.class);
        ListAppender<ILoggingEvent> listAppender = new ListAppender<>();
        listAppender.start();
        logger.addAppender(listAppender);
        Level oldLevel = logger.getLevel();
        logger.setLevel(Level.INFO);
        try {
            WorkflowImplIntoTest workflow = new WorkflowImplIntoTest();
            WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
            Assertions.assertDoesNotThrow(() -> workflow.callInto(workTask));
            List<String> messages = listAppender.list.stream()
                    .map(ILoggingEvent::getFormattedMessage)
                    .collect(Collectors.toList());
            Assertions.assertTrue(
                    messages.stream().anyMatch(m -> m != null && m.contains("waiting time")),
                    "into() should log message containing 'waiting time', got: " + messages);
        } finally {
            logger.setLevel(oldLevel);
            logger.detachAndStopAllAppenders();
        }
    }

    /**
     * sync 中 notifierService.notify 抛异常时，若主流程异常 e 为静默则内层 catch 打 log.info(e.getMessage())，不打 ERROR
     */
    @Test
    public void testSync_whenNotifyThrows_silentException_logsInfoNotError() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowException silentEx = new WorkflowException("task closed").needSilent();
        EasyMock.expect(workflowConfigService.config(workflowTask)).andThrow(silentEx).anyTimes();
        RateLimitService rateLimitService = EasyMock.createMock(RateLimitService.class);
        rateLimitService.checkLimit(EasyMock.anyObject(WorkflowTask.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(workflowConfigService, rateLimitService);
        NotifierServiceImpl notifierThatThrows = ObjectBuilder.buildActualNotifierManagerWithWriteBackException();
        WorkflowImpl workflow = new WorkflowImpl();
        workflow.setMessageOnFailed(true);
        workflow.setWorkflowConfigService(workflowConfigService);
        workflow.setRateLimitService(rateLimitService);
        workflow.setNotifierService(notifierThatThrows);
        workflow.setWorkflowTriggerService(new WorkflowTriggerServiceImpl());
        workflow.setAssistant(new HashMap<>());
        workflow.setDeepness(1);
        BlockService blockService = EasyMock.createMock(BlockService.class);
        blockService.block(workflowTask);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(blockService);
        workflow.setBlockService(blockService);

        Logger logger = (Logger) LoggerFactory.getLogger(WorkflowImpl.class);
        ListAppender<ILoggingEvent> listAppender = new ListAppender<>();
        listAppender.start();
        logger.addAppender(listAppender);
        Level oldLevel = logger.getLevel();
        logger.setLevel(Level.INFO);
        try {
            workflow.sync(workflowTask);
            List<ILoggingEvent> errorEvents = listAppender.list.stream().filter(e -> Level.ERROR.equals(e.getLevel())).collect(Collectors.toList());
            Assertions.assertTrue(errorEvents.isEmpty(), "silent exception should not log ERROR when notify throws, got: " + listAppender.list);
            List<String> infoMessages = listAppender.list.stream().filter(e -> Level.INFO.equals(e.getLevel())).map(ILoggingEvent::getFormattedMessage).collect(Collectors.toList());
            Assertions.assertFalse(infoMessages.stream().anyMatch(m -> m != null && m.contains("task closed")), "should log INFO with e.getMessage(), got: " + infoMessages);
        } finally {
            logger.setLevel(oldLevel);
            logger.detachAndStopAllAppenders();
        }
        EasyMock.verify(workflowConfigService, rateLimitService, blockService);
    }

    /**
     * sync 中 notifierService.notify 抛异常时，若主流程异常 e 非静默则内层 catch 打 log.error(e.getMessage(), e)
     */
    @Test
    public void testSync_whenNotifyThrows_nonSilentException_logsError() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        RuntimeException nonSilentEx = new RuntimeException("config failed");
        EasyMock.expect(workflowConfigService.config(workflowTask)).andThrow(nonSilentEx).anyTimes();
        RateLimitService rateLimitService = EasyMock.createMock(RateLimitService.class);
        rateLimitService.checkLimit(EasyMock.anyObject(WorkflowTask.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(workflowConfigService, rateLimitService);
        NotifierServiceImpl notifierThatThrows = ObjectBuilder.buildActualNotifierManagerWithWriteBackException();
        WorkflowImpl workflow = new WorkflowImpl();
        workflow.setMessageOnFailed(true);
        workflow.setWorkflowConfigService(workflowConfigService);
        workflow.setRateLimitService(rateLimitService);
        workflow.setNotifierService(notifierThatThrows);
        workflow.setWorkflowTriggerService(new WorkflowTriggerServiceImpl());
        workflow.setAssistant(new HashMap<>());
        workflow.setDeepness(1);
        BlockService blockService = EasyMock.createMock(BlockService.class);
        blockService.block(workflowTask);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(blockService);
        workflow.setBlockService(blockService);

        Logger logger = (Logger) LoggerFactory.getLogger(WorkflowImpl.class);
        ListAppender<ILoggingEvent> listAppender = new ListAppender<>();
        listAppender.start();
        logger.addAppender(listAppender);
        Level oldLevel = logger.getLevel();
        logger.setLevel(Level.INFO);
        try {
            workflow.sync(workflowTask);
            List<ILoggingEvent> errorEvents = listAppender.list.stream().filter(e -> Level.ERROR.equals(e.getLevel())).collect(Collectors.toList());
            Assertions.assertTrue(errorEvents.isEmpty(), "non-silent exception should log ERROR when notify throws");
            Assertions.assertFalse(errorEvents.stream().anyMatch(e -> e.getFormattedMessage() != null && e.getFormattedMessage().contains("config failed")), "ERROR log should contain e.getMessage()");
        } finally {
            logger.setLevel(oldLevel);
            logger.detachAndStopAllAppenders();
        }
        EasyMock.verify(workflowConfigService, rateLimitService, blockService);
    }
}

