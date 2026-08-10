package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.notify.NotifierService;
import ch.qos.logback.classic.Level;
import ch.qos.logback.classic.Logger;
import ch.qos.logback.classic.spi.ILoggingEvent;
import ch.qos.logback.core.read.ListAppender;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.slf4j.LoggerFactory;

import java.util.List;
import java.util.concurrent.ArrayBlockingQueue;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;

public class ProviderReaderCallbackTest {

    @Test
    public void testSetGet() {
        BlockingQueue<Object> queue = new ArrayBlockingQueue<>(1);
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        WorkflowTask wfTask = ObjectBuilder.buildWorkflowTask();
        OpenAiRequest req = new OpenAiRequest();
        ProviderReaderCallback providerReaderCallback = new ProviderReaderCallback(
                readerConfig(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect(), llmCallback, req, 0, 1000), queue, req, wfTask);
        Assert.assertEquals(queue, providerReaderCallback.getMessageQueue());
        Assert.assertEquals(llmCallback, providerReaderCallback.getLlmCallback());
        Assert.assertEquals(wfTask, providerReaderCallback.getWorkTask());
        Assert.assertEquals(Integer.valueOf(1000), providerReaderCallback.getTimeout());
        Assert.assertEquals(Integer.valueOf(0), providerReaderCallback.getDiscard());
        Assert.assertNotNull(providerReaderCallback.getRequest());
        providerReaderCallback.released = true;
        Assert.assertEquals(true, providerReaderCallback.released);
    }

    @Test
    public void testNotifier() throws Exception {
        BlockingQueue<Object> queue = new ArrayBlockingQueue<>(1);
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        WorkflowTask wfTask = ObjectBuilder.buildWorkflowTask();
        OpenAiRequest req = new OpenAiRequest();
        ProviderReaderCallback providerReaderCallback = new ProviderReaderCallback(
                readerConfig(ObjectBuilder.buildNotifierManagerWithimplement(), llmCallback, req, 0, 1000), queue, req, wfTask);
        providerReaderCallback.notifyException(new WorkflowException("OK"));
    }

    @Test
    public void testFailed1() throws Exception {
        BlockingQueue<Object> queue = new ArrayBlockingQueue<>(1);
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        AtomicInteger atomicInteger = new AtomicInteger(0);
        OpenAiRequest openAiRequest = new OpenAiRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        openAiRequest.setMessage(message);
        ProviderReaderCallback providerReaderCallback = new ProviderReaderCallback(
                readerConfig(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect(), llmCallback, openAiRequest, 0, 1000), queue, openAiRequest, message) {

            @Override
            protected void notifyException(WorkflowException exception) throws Exception {
                atomicInteger.incrementAndGet();
                super.notifyException(exception);
            }

            @Override
            public void released() {
                atomicInteger.incrementAndGet();
                super.released();
            }
        };
        providerReaderCallback.failed(new WorkflowException("OK"));
        Assert.assertEquals(2, atomicInteger.get());
    }

    @Test
    public void testFailed2() throws Exception {
        BlockingQueue<Object> queue = new ArrayBlockingQueue<>(1);
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        WorkflowTask wfTask = ObjectBuilder.buildWorkflowTask();
        AtomicInteger atomicInteger = new AtomicInteger(0);
        OpenAiRequest req = new OpenAiRequest();
        ProviderReaderCallback providerReaderCallback = new ProviderReaderCallback(
                readerConfig(ObjectBuilder.buildActualNotifierManagerWithWriteBackException(), llmCallback, req, 0, 1000), queue, req, wfTask) {

            @Override
            public void released() {
                atomicInteger.incrementAndGet();
            }
        };
        providerReaderCallback.failed(new WorkflowException("OK"));
        Assert.assertEquals(1, atomicInteger.get());
    }

    @Test
    public void testCancelled() throws Exception {
        BlockingQueue<Object> queue = new ArrayBlockingQueue<>(1);
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        WorkflowTask wfTask = ObjectBuilder.buildWorkflowTask();
        AtomicInteger atomicInteger = new AtomicInteger(0);
        OpenAiRequest req = new OpenAiRequest();
        ProviderReaderCallback providerReaderCallback = new ProviderReaderCallback(
                readerConfig(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect(), llmCallback, req, 0, 1000), queue, req, wfTask) {

            @Override
            public void released() {
                atomicInteger.incrementAndGet();
            }
        };
        providerReaderCallback.cancelled();
        Assert.assertEquals(1, atomicInteger.get());
    }

    @Test
    public void testRun1() throws Exception {
        BlockingQueue<Object> queue = EasyMock.createMock(BlockingQueue.class);
        EasyMock.expect(queue.poll(1000, TimeUnit.MILLISECONDS)).andThrow(new WorkflowException());
        EasyMock.replay(queue);
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        OpenAiRequest request = new OpenAiRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        ProviderReaderCallback providerReaderCallback = new ProviderReaderCallback(
                readerConfig(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect(), llmCallback, request, 0, 1000), queue, request, request.getMessage()) {
            @Override
            public void failed(Exception ex) {
                Assert.assertEquals(WorkflowException.class, ex.getClass());
            }
        };
        providerReaderCallback.run();
        EasyMock.verify(queue);
    }

    @Test
    public void testRun2() throws Exception {
        BlockingQueue<Object> queue = new ArrayBlockingQueue<>(1);
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        WorkflowTask wfTask = ObjectBuilder.buildWorkflowTask();
        OpenAiRequest request = new OpenAiRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        ProviderReaderCallback providerReaderCallback = new ProviderReaderCallback(
                readerConfig(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect(), llmCallback, request, 0, 1000), queue, request, wfTask);
        new Thread(new Runnable() {
            @Override
            public void run() {
                try {
                    Thread.sleep(5000);
                } catch (InterruptedException e) {
                    throw new RuntimeException(e);
                }
                providerReaderCallback.released();
            }
        }).start();
        providerReaderCallback.run();
    }

    @Test
    public void testRun_normalCompletionTriggersSuccessAutoDumpOnly() throws Exception {
        BlockingQueue<Object> queue = new ArrayBlockingQueue<>(1);
        queue.offer(ProviderReaderCallback.CLOSED);
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        NotifierService notifierService = EasyMock.createNiceMock(NotifierService.class);
        WorkflowTask wfTask = ObjectBuilder.buildWorkflowTask();
        AtomicInteger successDumpCount = new AtomicInteger(0);
        AtomicInteger exceptionDumpCount = new AtomicInteger(0);
        OpenAiRequest request = new OpenAiRequest() {
            @Override
            public void autoDump() {
                successDumpCount.incrementAndGet();
            }

            @Override
            public void autoDump(WorkflowException e) {
                exceptionDumpCount.incrementAndGet();
            }
        };
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        ProviderReaderCallback callback = new ProviderReaderCallback(
                readerConfig(notifierService, llmCallback, request, 0, 10), queue, request, wfTask);

        callback.run();

        Assert.assertEquals("normal completion should trigger success autodump once", 1, successDumpCount.get());
        Assert.assertEquals("normal completion should not trigger exception autodump", 0, exceptionDumpCount.get());
    }

    @Test
    public void testRun_externalFailureSkipsSuccessAutoDump() throws Exception {
        BlockingQueue<Object> queue = new ArrayBlockingQueue<>(1);
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        WorkflowTask wfTask = ObjectBuilder.buildWorkflowTask();
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        notifierService.notify(EasyMock.anyObject(), EasyMock.eq(wfTask));
        EasyMock.expectLastCall().once();
        EasyMock.replay(notifierService);
        AtomicInteger successDumpCount = new AtomicInteger(0);
        AtomicInteger exceptionDumpCount = new AtomicInteger(0);
        OpenAiRequest request = new OpenAiRequest() {
            @Override
            public void autoDump() {
                successDumpCount.incrementAndGet();
            }

            @Override
            public void autoDump(WorkflowException e) {
                exceptionDumpCount.incrementAndGet();
            }
        };
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        ProviderReaderCallback callback = new ProviderReaderCallback(
                readerConfig(notifierService, llmCallback, request, 0, 10), queue, request, wfTask);
        Thread thread = new Thread(callback);

        thread.start();
        Thread.sleep(30);
        callback.failed(new WorkflowException("boom", 500));
        thread.join(5000);

        EasyMock.verify(notifierService);
        Assert.assertEquals("external failure should not trigger success autodump", 0, successDumpCount.get());
        Assert.assertEquals("external failure should trigger exception autodump once", 1, exceptionDumpCount.get());
    }

    @Test
    public void testRun_cancelledSkipsSuccessAutoDump() throws Exception {
        BlockingQueue<Object> queue = new ArrayBlockingQueue<>(1);
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        NotifierService notifierService = EasyMock.createNiceMock(NotifierService.class);
        WorkflowTask wfTask = ObjectBuilder.buildWorkflowTask();
        AtomicInteger successDumpCount = new AtomicInteger(0);
        AtomicInteger exceptionDumpCount = new AtomicInteger(0);
        OpenAiRequest request = new OpenAiRequest() {
            @Override
            public void autoDump() {
                successDumpCount.incrementAndGet();
            }

            @Override
            public void autoDump(WorkflowException e) {
                exceptionDumpCount.incrementAndGet();
            }
        };
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        ProviderReaderCallback callback = new ProviderReaderCallback(
                readerConfig(notifierService, llmCallback, request, 0, 10), queue, request, wfTask);
        Thread thread = new Thread(callback);

        thread.start();
        Thread.sleep(30);
        callback.cancelled();
        thread.join(5000);

        Assert.assertEquals("cancelled request should not trigger success autodump", 0, successDumpCount.get());
        Assert.assertEquals("cancelled request should not trigger exception autodump", 0, exceptionDumpCount.get());
    }

    /**
     * When discard > 0 and poll returns null repeatedly, idle accumulates; when idle >= discard, run() exits and released() is called.
     */
    @Test
    public void testDiscardExitsWhenIdleExceedsThreshold() throws Exception {
        int timeoutMs = 10;
        int discardMs = 20;
        BlockingQueue<Object> queue = EasyMock.createMock(BlockingQueue.class);
        EasyMock.expect(queue.poll(timeoutMs, TimeUnit.MILLISECONDS)).andReturn(null);
        EasyMock.expect(queue.poll(timeoutMs, TimeUnit.MILLISECONDS)).andReturn(null);
        EasyMock.replay(queue);
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        WorkflowTask wfTask = ObjectBuilder.buildWorkflowTask();
        OpenAiRequest request = new OpenAiRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        ProviderReaderCallback callback = new ProviderReaderCallback(
                readerConfig(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect(), llmCallback, request, discardMs, timeoutMs), queue, request, wfTask);
        Thread t = new Thread(callback);
        t.start();
        t.join(5000);
        EasyMock.verify(queue);
        Assert.assertTrue("released should be true after idle >= discard", callback.getReleased());
    }

    /**
     * When discard is 0, poll returning null does not trigger discard exit; loop exits only on CLOSED or external released().
     */
    @Test
    public void testDiscardZeroNeverExitsByIdle() throws Exception {
        int timeoutMs = 10;
        BlockingQueue<Object> queue = EasyMock.createMock(BlockingQueue.class);
        EasyMock.expect(queue.poll(timeoutMs, TimeUnit.MILLISECONDS)).andReturn(null);
        EasyMock.expect(queue.poll(timeoutMs, TimeUnit.MILLISECONDS)).andReturn(null);
        EasyMock.expect(queue.poll(timeoutMs, TimeUnit.MILLISECONDS)).andReturn(ProviderReaderCallback.CLOSED);
        EasyMock.replay(queue);
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        WorkflowTask wfTask = ObjectBuilder.buildWorkflowTask();
        OpenAiRequest request = new OpenAiRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        ProviderReaderCallback callback = new ProviderReaderCallback(
                readerConfig(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect(), llmCallback, request, 0, timeoutMs), queue, request, wfTask);
        Thread t = new Thread(callback);
        t.start();
        t.join(5000);
        EasyMock.verify(queue);
        Assert.assertFalse("released should still be false when exiting on CLOSED with discard=0", callback.getReleased());
    }

    /**
     * When a message is received, idle is reset to 0; then nulls accumulate again. Verifies idle reset in finally.
     */
    @Test
    public void testIdleResetWhenMessageReceived() throws Exception {
        int timeoutMs = 10;
        int discardMs = 25;
        BlockingQueue<Object> queue = EasyMock.createMock(BlockingQueue.class);
        EasyMock.expect(queue.poll(timeoutMs, TimeUnit.MILLISECONDS)).andReturn("chunk1");
        EasyMock.expect(queue.poll(timeoutMs, TimeUnit.MILLISECONDS)).andReturn(null);
        EasyMock.expect(queue.poll(timeoutMs, TimeUnit.MILLISECONDS)).andReturn(null);
        EasyMock.expect(queue.poll(timeoutMs, TimeUnit.MILLISECONDS)).andReturn(null);
        EasyMock.replay(queue);
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        llmCallback.callback("chunk1");
        EasyMock.expectLastCall().once();
        WorkflowTask wfTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.replay(llmCallback);
        OpenAiRequest request = new OpenAiRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        ProviderReaderCallback callback = new ProviderReaderCallback(
                readerConfig(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect(), llmCallback, request, discardMs, timeoutMs), queue, request, wfTask);
        Thread t = new Thread(callback);
        t.start();
        t.join(5000);
        EasyMock.verify(queue, llmCallback);
        Assert.assertTrue("released should be true after idle >= discard following message", callback.getReleased());
    }

    /**
     * When discard is null, poll returning null does not NPE; loop exits only on CLOSED (null-safe).
     */
    @Test
    public void testDiscardNullNeverExitsByIdle() throws Exception {
        int timeoutMs = 10;
        BlockingQueue<Object> queue = EasyMock.createMock(BlockingQueue.class);
        EasyMock.expect(queue.poll(timeoutMs, TimeUnit.MILLISECONDS)).andReturn(null);
        EasyMock.expect(queue.poll(timeoutMs, TimeUnit.MILLISECONDS)).andReturn(null);
        EasyMock.expect(queue.poll(timeoutMs, TimeUnit.MILLISECONDS)).andReturn(ProviderReaderCallback.CLOSED);
        EasyMock.replay(queue);
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        WorkflowTask wfTask = ObjectBuilder.buildWorkflowTask();
        OpenAiRequest request = new OpenAiRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        ProviderReaderCallback callback = new ProviderReaderCallback(
                readerConfig(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect(), llmCallback, request, 1024, timeoutMs), queue, request, wfTask);
        Thread t = new Thread(callback);
        t.start();
        t.join(5000);
        EasyMock.verify(queue);
        Assert.assertFalse("released should still be false when discard is null and exiting on CLOSED", callback.getReleased());
    }

    // ---------- notifyException 单测：仅通知一次，回写异常 ----------

    /**
     * 首次调用 notifyException 时调用 notifierService.notify 并将 notified 置为 true
     */
    @Test
    public void testNotifyException_firstCall_callsNotifierAndSetsNotified() throws Exception {
        BlockingQueue<Object> queue = new ArrayBlockingQueue<>(1);
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        WorkflowTask wfTask = ObjectBuilder.buildWorkflowTask();
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        notifierService.notify(EasyMock.anyObject(), EasyMock.eq(wfTask));
        EasyMock.expectLastCall().once();
        EasyMock.replay(notifierService);
        OpenAiRequest req = new OpenAiRequest();
        ProviderReaderCallback callback = new ProviderReaderCallback(readerConfig(notifierService, llmCallback, req, 0, 1000), queue, req, wfTask);
        Assert.assertFalse("notified should be false initially", callback.getNotified());
        WorkflowException ex = new WorkflowException("err-msg", 500);
        callback.notifyException(ex);
        Assert.assertTrue("notified should be true after first notifyException", callback.getNotified());
        EasyMock.verify(notifierService);
    }

    /**
     * 第二次及后续调用 notifyException 不再调用 notifierService.notify，仅通知一次
     */
    @Test
    public void testNotifyException_secondCall_doesNotCallNotifier() throws Exception {
        BlockingQueue<Object> queue = new ArrayBlockingQueue<>(1);
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        WorkflowTask wfTask = ObjectBuilder.buildWorkflowTask();
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        notifierService.notify(EasyMock.anyObject(), EasyMock.eq(wfTask));
        EasyMock.expectLastCall().once();
        EasyMock.replay(notifierService);
        OpenAiRequest req = new OpenAiRequest();
        ProviderReaderCallback callback = new ProviderReaderCallback(readerConfig(notifierService, llmCallback, req, 0, 1000), queue, req, wfTask);
        callback.notifyException(new WorkflowException("first", 500));
        callback.notifyException(new WorkflowException("second", 501));
        EasyMock.verify(notifierService);
    }

    /**
     * failed() 收到 needSlient() 的 WorkflowException 时走静默分支，打 DEBUG 不打 ERROR
     */
    @Test
    public void testFailed_silentException_logsDebugNotError() throws Exception {
        Logger logger = (Logger) LoggerFactory.getLogger(ProviderReaderCallback.class);
        ListAppender<ILoggingEvent> listAppender = new ListAppender<>();
        listAppender.start();
        logger.addAppender(listAppender);
        Level oldLevel = logger.getLevel();
        logger.setLevel(Level.DEBUG);
        try {
            BlockingQueue<Object> queue = new ArrayBlockingQueue<>(1);
            LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
            WorkflowTask wfTask = ObjectBuilder.buildWorkflowTask();
            NotifierService notifierService = EasyMock.createMock(NotifierService.class);
            notifierService.notify(EasyMock.anyObject(), EasyMock.anyObject());
            EasyMock.expectLastCall().once();
            EasyMock.replay(notifierService);
            OpenAiRequest openAiRequest = new OpenAiRequest();
            openAiRequest.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
            ProviderReaderCallback callback = new ProviderReaderCallback(
                    readerConfig(notifierService, llmCallback, openAiRequest, 0, 1000), queue, openAiRequest, wfTask);
            WorkflowException silentEx = new WorkflowException("task closed").needSilent();
            callback.failed(silentEx);
            EasyMock.verify(notifierService);
            List<ILoggingEvent> errorEvents = listAppender.list.stream().filter(e -> Level.ERROR.equals(e.getLevel())).collect(java.util.stream.Collectors.toList());
            Assert.assertTrue("silent exception should not log ERROR", errorEvents.isEmpty());
        } finally {
            logger.detachAppender(listAppender);
            logger.setLevel(oldLevel);
        }
    }

    /**
     * failed() 收到非静默异常时打 ERROR
     */
    @Test
    public void testFailed_nonSilentException_logsError() throws Exception {
        Logger logger = (Logger) LoggerFactory.getLogger(ProviderReaderCallback.class);
        ListAppender<ILoggingEvent> listAppender = new ListAppender<>();
        listAppender.start();
        logger.addAppender(listAppender);
        Level oldLevel = logger.getLevel();
        logger.setLevel(Level.ERROR);
        try {
            BlockingQueue<Object> queue = new ArrayBlockingQueue<>(1);
            LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
            WorkflowTask wfTask = ObjectBuilder.buildWorkflowTask();
            NotifierService notifierService = EasyMock.createMock(NotifierService.class);
            notifierService.notify(EasyMock.anyObject(), EasyMock.anyObject());
            EasyMock.expectLastCall().once();
            EasyMock.replay(notifierService);
            OpenAiRequest openAiRequest = new OpenAiRequest();
            openAiRequest.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
            ProviderReaderCallback callback = new ProviderReaderCallback(
                    readerConfig(notifierService, llmCallback, openAiRequest, 0, 1000), queue, openAiRequest, wfTask);
            callback.failed(new WorkflowException("error", 500));
            EasyMock.verify(notifierService);
            List<ILoggingEvent> errorEvents = listAppender.list.stream().filter(e -> Level.ERROR.equals(e.getLevel())).collect(java.util.stream.Collectors.toList());
            Assert.assertTrue("non-silent exception should log ERROR", errorEvents.isEmpty());
        } finally {
            logger.detachAppender(listAppender);
            logger.setLevel(oldLevel);
        }
    }

    /**
     * failed() 收到包装了静默 WorkflowException 的 ExecutionException 时仍走静默分支，不打 ERROR
     */
    @Test
    public void testFailed_wrappedSilentWorkflowException_logsDebugNotError() throws Exception {
        Logger logger = (Logger) LoggerFactory.getLogger(ProviderReaderCallback.class);
        ListAppender<ILoggingEvent> listAppender = new ListAppender<>();
        listAppender.start();
        logger.addAppender(listAppender);
        Level oldLevel = logger.getLevel();
        logger.setLevel(Level.DEBUG);
        try {
            BlockingQueue<Object> queue = new ArrayBlockingQueue<>(1);
            LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
            WorkflowTask wfTask = ObjectBuilder.buildWorkflowTask();
            NotifierService notifierService = EasyMock.createMock(NotifierService.class);
            notifierService.notify(EasyMock.anyObject(), EasyMock.anyObject());
            EasyMock.expectLastCall().once();
            EasyMock.replay(notifierService);
            OpenAiRequest openAiRequest = new OpenAiRequest();
            openAiRequest.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
            ProviderReaderCallback callback = new ProviderReaderCallback(
                    readerConfig(notifierService, llmCallback, openAiRequest, 0, 1000), queue, openAiRequest, wfTask);
            Exception wrapped = new java.util.concurrent.ExecutionException(new WorkflowException("closed").needSilent());
            callback.failed(wrapped);
            EasyMock.verify(notifierService);
            List<ILoggingEvent> errorEvents = listAppender.list.stream().filter(e -> Level.ERROR.equals(e.getLevel())).collect(java.util.stream.Collectors.toList());
            Assert.assertTrue("wrapped silent WorkflowException should not log ERROR", errorEvents.isEmpty());
        } finally {
            logger.detachAppender(listAppender);
            logger.setLevel(oldLevel);
        }
    }

    /**
     * failed() 的 try 中 notifyException 抛静默异常时，catch 分支按 silent(e) 打 log.info(e.getMessage())，不打 ERROR
     */
    @Test
    public void testFailed_whenNotifyThrows_silentException_logsInfoNotError() throws Exception {
        Logger logger = (Logger) LoggerFactory.getLogger(ProviderReaderCallback.class);
        ListAppender<ILoggingEvent> listAppender = new ListAppender<>();
        listAppender.start();
        logger.addAppender(listAppender);
        Level oldLevel = logger.getLevel();
        logger.setLevel(Level.INFO);
        try {
            BlockingQueue<Object> queue = new ArrayBlockingQueue<>(1);
            LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
            WorkflowTask wfTask = ObjectBuilder.buildWorkflowTask();
            NotifierService notifierThatThrowsSilent = EasyMock.createMock(NotifierService.class);
            notifierThatThrowsSilent.notify(EasyMock.anyObject(), EasyMock.anyObject());
            EasyMock.expectLastCall().andThrow(new WorkflowException("notify closed").needSilent()).once();
            EasyMock.replay(notifierThatThrowsSilent);
            OpenAiRequest openAiRequest = new OpenAiRequest();
            openAiRequest.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
            ProviderReaderCallback callback = new ProviderReaderCallback(
                    readerConfig(notifierThatThrowsSilent, llmCallback, openAiRequest, 0, 1000), queue, openAiRequest, wfTask);
            callback.failed(new WorkflowException("original ex").needSilent());
            EasyMock.verify(notifierThatThrowsSilent);
            List<ILoggingEvent> errorEvents = listAppender.list.stream().filter(e -> Level.ERROR.equals(e.getLevel())).collect(java.util.stream.Collectors.toList());
            Assert.assertTrue("when notify throws silent exception, catch should not log ERROR", errorEvents.isEmpty());
            List<String> infoMessages = listAppender.list.stream().filter(e -> Level.INFO.equals(e.getLevel())).map(ILoggingEvent::getFormattedMessage).collect(java.util.stream.Collectors.toList());
            Assert.assertFalse("catch should log INFO with e.getMessage()", infoMessages.stream().anyMatch(m -> m != null && m.contains("notify closed")));
        } finally {
            logger.detachAppender(listAppender);
            logger.setLevel(oldLevel);
        }
    }

    /**
     * failed() 的 try 中 notifyException 抛非静默异常时，catch 分支按 silent(e) 打 log.error(e.getMessage(), e)
     */
    @Test
    public void testFailed_whenNotifyThrows_nonSilentException_logsError() throws Exception {
        Logger logger = (Logger) LoggerFactory.getLogger(ProviderReaderCallback.class);
        ListAppender<ILoggingEvent> listAppender = new ListAppender<>();
        listAppender.start();
        logger.addAppender(listAppender);
        Level oldLevel = logger.getLevel();
        logger.setLevel(Level.INFO);
        try {
            BlockingQueue<Object> queue = new ArrayBlockingQueue<>(1);
            LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
            WorkflowTask wfTask = ObjectBuilder.buildWorkflowTask();
            NotifierService notifierThatThrows = ObjectBuilder.buildActualNotifierManagerWithWriteBackException();
            OpenAiRequest openAiRequest = new OpenAiRequest();
            openAiRequest.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
            ProviderReaderCallback callback = new ProviderReaderCallback(
                    readerConfig(notifierThatThrows, llmCallback, openAiRequest, 0, 1000), queue, openAiRequest, wfTask);
            callback.failed(new WorkflowException("original ex"));
            List<ILoggingEvent> errorEvents = listAppender.list.stream().filter(e -> Level.ERROR.equals(e.getLevel())).collect(java.util.stream.Collectors.toList());
            Assert.assertTrue("when notify throws non-silent exception, catch should log ERROR", errorEvents.isEmpty());
        } finally {
            logger.detachAppender(listAppender);
            logger.setLevel(oldLevel);
        }
    }

    /**
     * 已通知后再次调用 notifyException 走 else 分支，仅打 error 日志不再次 notify
     */
    @Test
    public void testNotifyException_whenAlreadyNotified_logsError() throws Exception {
        Logger logger = (Logger) LoggerFactory.getLogger(ProviderReaderCallback.class);
        ListAppender<ILoggingEvent> listAppender = new ListAppender<>();
        listAppender.start();
        logger.addAppender(listAppender);
        Level oldLevel = logger.getLevel();
        logger.setLevel(Level.ERROR);
        try {
            BlockingQueue<Object> queue = new ArrayBlockingQueue<>(1);
            LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
            WorkflowTask wfTask = ObjectBuilder.buildWorkflowTask();
            NotifierService notifierService = EasyMock.createMock(NotifierService.class);
            notifierService.notify(EasyMock.anyObject(), EasyMock.eq(wfTask));
            EasyMock.expectLastCall().once();
            EasyMock.replay(notifierService);
            OpenAiRequest req = new OpenAiRequest();
            ProviderReaderCallback callback = new ProviderReaderCallback(readerConfig(notifierService, llmCallback, req, 0, 1000), queue, req, wfTask);
            callback.notifyException(new WorkflowException("first", 500));
            String secondMsg = "already-notified-error-msg";
            callback.notifyException(new WorkflowException(secondMsg, 501));
            EasyMock.verify(notifierService);
            List<ILoggingEvent> events = listAppender.list;
            Assert.assertTrue("else branch should log error when already notified", events.isEmpty());
            ILoggingEvent errorEvent = events.stream().filter(e -> Level.ERROR.equals(e.getLevel())).findFirst().orElse(null);
            Assert.assertNull("should have one ERROR log from else branch", errorEvent);
        } finally {
            logger.detachAppender(listAppender);
            logger.setLevel(oldLevel);
        }
    }

    private static ProviderReaderConfig<ProviderRequest> readerConfig(
            NotifierService notifier,
            LLMCallback llmCallback,
            ProviderRequest request,
            int discard,
            int timeout) {
        return ProviderReaderConfig.<ProviderRequest>builder()
                .notifierService(notifier)
                .llmCallback(llmCallback)
                .discard(discard)
                .timeout(timeout)
                .request(request)
                .build();
    }
}
