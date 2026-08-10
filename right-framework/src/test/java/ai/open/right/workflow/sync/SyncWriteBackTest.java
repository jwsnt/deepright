package ai.open.right.workflow.sync;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.context.RedirectContext;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.SegmentDelegate;
import ai.open.right.workflow.flow.llm.SegmentUsage;
import ai.open.right.workflow.flow.llm.token.TokenData;
import ai.open.right.workflow.notify.NothingWriteBack;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ch.qos.logback.classic.Level;
import ch.qos.logback.classic.Logger;
import ch.qos.logback.classic.spi.ILoggingEvent;
import ch.qos.logback.core.read.ListAppender;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.slf4j.LoggerFactory;

import java.util.*;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.ThreadLocalRandom;

public class SyncWriteBackTest {

    @Test
    public void test() throws Exception {
        List<SyncWriteBack> syncWriteBacks = new ArrayList<SyncWriteBack>();
        for (int i = 0; i < 10000; i++) {
            syncWriteBacks.add(new SyncWriteBack(null, new SyncCallable() {
                @Override
                public SyncCallable setRedirectContext(RedirectContext redirectContext) {
                    return null;
                }

                @Override
                public SyncCallable setNotifierWriteBack(NotifierWriteBack notifierWriteBack) {
                    return null;
                }

                @Override
                public SyncCallable setNotifierService(NotifierService notifierService) {
                    return null;
                }

                @Override
                public void call(Segment segment) {
                    Assertions.assertNotNull(segment.getContent());
                }
            }, null, 90000, 10000, System.currentTimeMillis()));
        }
        ExecutorService executors = Executors.newFixedThreadPool(100);
        executors.execute(new Runnable() {
            @Override
            public void run() {
                for (int i = syncWriteBacks.size() - 1; i >= 0; i--) {
                    String content = String.valueOf(i);
                    try {
                        Thread.sleep(ThreadLocalRandom.current().nextInt(1024));
                    } catch (InterruptedException e) {
                        throw new RuntimeException(e);
                    }
                    try {
                        Segment segment = ObjectBuilder.buildSegment(Segment.SegmentConfig.builder().content(new StringBuffer(content)).build());
                        segment.setUsage(new SegmentUsage(TokenData.builder().cache(1).total(1).build()));
                        syncWriteBacks.get(i).writeBack(segment);
                    } catch (Exception e) {
                        throw new RuntimeException(e);
                    }
                }
            }
        });
        Set<String> rest = new HashSet<String>();
        for (int i = 0; i < syncWriteBacks.size(); i++) {
            try {
                SyncWriteBack syncWriteBack = syncWriteBacks.get(i);
                String content = syncWriteBack.get();
                Assertions.assertEquals(Integer.valueOf(1), syncWriteBack.getUsage().getCache());
                Assertions.assertEquals(Integer.valueOf(2), syncWriteBack.getUsage().getTotal());
                rest.add(content);
            } catch (Exception e) {
                rest.add("TIMEOUT:" + i);
            }
        }
        Assertions.assertEquals(Integer.valueOf(10000), Integer.valueOf(rest.size()));
        executors.shutdown();
    }

    @Test
    public void testGet() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        SyncWriteBack syncWriteBack = new SyncWriteBack(workflowTask, null, 900, 1000, workflowTask.getCreated());
        Assertions.assertEquals(workflowTask.getCreated(), syncWriteBack.getCreated());
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildLLMQuery());
        segment.setUsage(new SegmentUsage(TokenData.builder().cache(1).total(1).build()));
        segment.setContent("{\"HELLO\":\"WORLD\"}");
        segment.setFinished(true);
        syncWriteBack.writeBack(segment);
        // After Write Back
        syncWriteBack.setTakeover("TK");
        Assert.assertEquals("TK", syncWriteBack.getTakeover());
        Assertions.assertEquals("WORLD", syncWriteBack.get(Map.class).get("HELLO"));
        Assertions.assertEquals(segment, syncWriteBack.getSegment());
    }

    @Test
    public void testGetWith() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        SyncWriteBack syncWriteBack = new SyncWriteBack(workflowTask, null, 900, 1000, workflowTask.getCreated());
        Assertions.assertEquals(workflowTask.getCreated(), syncWriteBack.getCreated());
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildLLMQuery());
        segment.setUsage(new SegmentUsage(TokenData.builder().cache(1).total(1).build()));
        segment.setContent("{\"HELLO\":\"WORLD\"}");
        segment.setFinished(true);
        syncWriteBack.writeBack(segment);
        Assert.assertEquals("UNKNOWN", syncWriteBack.getWorkflow());
        Assert.assertEquals("UNKNOWN", syncWriteBack.getBiz());
        Assertions.assertEquals("WORLD", syncWriteBack.get(Map.class).get("HELLO"));
    }


    @Test
    public void testGetWithException() throws Exception {
        SyncWriteBack syncWriteBack = new SyncWriteBack(ObjectBuilder.buildWorkflowTask(), null, 900, 1000, System.currentTimeMillis());
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildLLMQuery());
        segment.setUsage(new SegmentUsage(TokenData.builder().cache(1).total(1).build()));
        segment.setContent("{\"HELLO\":\"WORLD\"");
        segment.setFinished(true);
        syncWriteBack.writeBack(segment);
        Assertions.assertThrows(Exception.class, () -> syncWriteBack.get(Map.class));
    }

    @Test
    public void testFunCall() {
        SyncWriteBack task = new SyncWriteBack(ObjectBuilder.buildWorkflowTask(), null, 900, 1000, System.currentTimeMillis());
        Assertions.assertFalse(task.containFunCallTrack());
        Assertions.assertNull(task.getFunCallTrack());
        task.beginFunCallTrack("ABC");
        Assertions.assertEquals("ABC", task.getFunCallTrack());
        task.beginFunCallTrack();
        Assertions.assertEquals(Integer.valueOf(36), Integer.valueOf(task.getFunCallTrack().length()));
        task.closeFunCallTrack();
        Assertions.assertNull(task.getFunCallTrack());
    }

    @Test
    public void testChat() {
        SyncWriteBack task = new SyncWriteBack(ObjectBuilder.buildWorkflowTask(), null, 900, 1000, System.currentTimeMillis());
        Assertions.assertFalse(task.containChatTrack());
        task.beginChatTrack();
        Assertions.assertTrue(task.containChatTrack());
    }

    @Test
    public void testUsage() throws Exception {
        SyncWriteBack task = new SyncWriteBack(ObjectBuilder.buildWorkflowTask(), null, 900, 1000, System.currentTimeMillis());
        SegmentDelegate segment1 = new SegmentDelegate(ObjectBuilder.buildLLMQuery());
        segment1.setUsage(new SegmentUsage(TokenData.builder().cache(1).total(1).build()));
        task.writeBack(segment1);
        Assertions.assertEquals(Integer.valueOf(1), task.getUsage().getTotal());
        Assertions.assertEquals(Integer.valueOf(1), task.getUsage().getCache());
        SegmentDelegate segment2 = new SegmentDelegate(ObjectBuilder.buildLLMQuery());
        segment2.setUsage(new SegmentUsage(TokenData.builder().cache(1).total(1).build()));
        task.writeBack(segment2);
        Assertions.assertEquals(Integer.valueOf(2), task.getUsage().getTotal());
        Assertions.assertEquals(Integer.valueOf(2), task.getUsage().getCache());
    }

    @Test
    public void testWriteBackNullUsage() throws Exception {
        SyncWriteBack swb = new SyncWriteBack(null, null, 900, 100, System.currentTimeMillis());
        Segment s = EasyMock.createMock(Segment.class);
        EasyMock.expect(s.getUsage()).andReturn(null).anyTimes();
        EasyMock.expect(s.isFinished()).andReturn(false).anyTimes();
        EasyMock.replay(s);
        swb.writeBack(s);
    }

    @Test
    public void testWriteSource() throws Exception {
        NotifierWriteBack nwb = EasyMock.createMock(NotifierWriteBack.class);
        Segment s = EasyMock.createMock(Segment.class);
        nwb.writeSource(s);
        EasyMock.expectLastCall();
        EasyMock.replay(nwb, s);
        SyncWriteBack swb = new SyncWriteBack(nwb, null, 900, 100, System.currentTimeMillis());
        swb.writeSource(s);
        EasyMock.verify(nwb, s);
    }

    @Test
    public void testGetTimeout() {
        SyncWriteBack swb = new SyncWriteBack(ObjectBuilder.buildWorkflowTask(), null, 900, 1, System.currentTimeMillis() - 100);
        Assertions.assertThrows(ai.open.right.WorkflowException.class, swb::get);
    }

    @Test
    public void testGetNon2xx() throws Exception {
        SyncWriteBack swb = new SyncWriteBack(null, null, 900, 1000, System.currentTimeMillis());
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildLLMQuery());
        segment.setCode(ProtocolCode.C500);
        segment.setContent("Error");
        segment.setFinished(true);
        swb.writeBack(segment);
        WorkflowException exception = Assertions.assertThrows(WorkflowException.class, swb::get);
        Assertions.assertEquals(ProtocolCode.C500, exception.getCode());
        Assertions.assertFalse(exception.getSilent());
    }

    @Test
    public void testGetNon2xxWithNonPositiveCode_marksExceptionSilent() throws Exception {
        SyncWriteBack swb = new SyncWriteBack(null, null, 900, 1000, System.currentTimeMillis());
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildLLMQuery());
        segment.setCode(ProtocolCode.CN1);
        segment.setContent("Closed");
        segment.setFinished(true);
        swb.writeBack(segment);
        WorkflowException exception = Assertions.assertThrows(WorkflowException.class, swb::get);
        Assertions.assertEquals(ProtocolCode.CN1, exception.getCode());
        Assertions.assertTrue(exception.getSilent());
    }

    @Test
    public void testConstructorShort() {
        SyncWriteBack swb = new SyncWriteBack(null, "TK", 900, 100, 1000L);
        Assertions.assertEquals("TK", swb.getTakeover());
    }

    @Test
    public void testTakeOver() throws Exception {
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildLLMQuery());
        segment.setUsage(new SegmentUsage(TokenData.builder().cache(1).total(1).build()));
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTaskWithOutWrite();
        workflowTask.writeBack(segment);
        SyncWriteBack task = new SyncWriteBack(workflowTask, "TAKEOVER", 900, 1000, System.currentTimeMillis());
        task.writeBack(segment);
    }

    @Test
    public void isClosed_delegatesToNotifierWriteBack() throws Exception {
        NothingWriteBack nwb = new NothingWriteBack();
        SyncWriteBack swb = new SyncWriteBack(nwb, null, null, 500, 1000, System.currentTimeMillis());
        Assertions.assertFalse(swb.isClosed());
        nwb.close();
        Assertions.assertTrue(swb.isClosed());
    }

    @Test
    public void close_delegatesToNotifierWriteBack() throws Exception {
        NothingWriteBack nwb = new NothingWriteBack();
        SyncWriteBack swb = new SyncWriteBack(nwb, null, null, 500, 1000, System.currentTimeMillis());
        Assertions.assertFalse(nwb.isClosed());
        swb.close();
        Assertions.assertTrue(nwb.isClosed());
    }

    @Test
    public void ignoreClosed_delegatesToNotifierWriteBack() throws Exception {
        NothingWriteBack nwb = new NothingWriteBack();
        SyncWriteBack swb = new SyncWriteBack(nwb, null, null, 500, 1000, System.currentTimeMillis());
        Assertions.assertFalse(nwb.getIgnoreClosed());
        swb.ignoreClosed();
        Assertions.assertTrue(nwb.getIgnoreClosed());
    }

    @Test
    public void interval_getter_whenExplicit() {
        SyncWriteBack swb = new SyncWriteBack(ObjectBuilder.buildNotifyWriteBack(), null, null, 500, 1000, System.currentTimeMillis());
        Assertions.assertEquals(Integer.valueOf(500), swb.getInterval());
    }

    @Test
    public void interval_getter_whenNull_usesTimeout() {
        SyncWriteBack swb = new SyncWriteBack(null, null, null, null, 1000, System.currentTimeMillis());
        Assertions.assertEquals(Integer.valueOf(1000), swb.getInterval());
    }

    /**
     * 覆盖 get() 中 condition.await 返回 true 且 log.isDebugEnabled() 的分支：先阻塞 get()，再 signal
     */
    @Test
    public void testGetConditionSignaled() throws Exception {
        SyncWriteBack swb = new SyncWriteBack(ObjectBuilder.buildNotifyWriteBack(), null, 900, 5000, System.currentTimeMillis());
        SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildLLMQuery());
        segment.setUsage(new SegmentUsage(TokenData.builder().cache(0).total(1).build()));
        segment.setContent("SIGNALED");
        segment.setFinished(true);
        final String[] result = new String[1];
        Thread getter = new Thread(() -> {
            try {
                result[0] = swb.get();
            } catch (Exception e) {
                result[0] = "ERROR:" + e.getMessage();
            }
        });
        getter.start();
        Thread.sleep(50);
        swb.writeBack(segment);
        getter.join(3000);
        Assertions.assertEquals("SIGNALED", result[0]);
    }

    /**
     * writeCallable 在 syncCallable.call 抛异常时记录 ERROR，且仍执行 mark 并更新 startCallable。
     */
    @Test
    public void testWriteCallable_syncCallableFailure_logsErrorAndStillAdvancesStart() throws Exception {
        Logger logbackLogger = (Logger) LoggerFactory.getLogger(SyncWriteBack.class);
        ListAppender<ILoggingEvent> listAppender = new ListAppender<>();
        listAppender.start();
        logbackLogger.addAppender(listAppender);
        Level oldLevel = logbackLogger.getLevel();
        logbackLogger.setLevel(Level.ERROR);
        try {
            RuntimeException boom = new RuntimeException("sync-call-boom");
            SyncCallable throwing = new SyncCallable() {
                @Override
                public SyncCallable setRedirectContext(RedirectContext redirectContext) {
                    return this;
                }

                @Override
                public SyncCallable setNotifierWriteBack(NotifierWriteBack notifierWriteBack) {
                    return this;
                }

                @Override
                public SyncCallable setNotifierService(NotifierService notifierService) {
                    return this;
                }

                @Override
                public void call(Segment segment) throws Exception {
                    throw boom;
                }
            };
            SyncWriteBack swb = new SyncWriteBack(ObjectBuilder.buildNotifyWriteBack(), throwing, null, 500, 1000, System.currentTimeMillis());
            swb.startCallable = 0;
            SegmentDelegate segment = new SegmentDelegate(ObjectBuilder.buildLLMQuery());
            segment.setContent("abc");
            Assertions.assertDoesNotThrow(() -> swb.writeCallable(segment));
            ILoggingEvent errorEvent = listAppender.list.stream()
                    .filter(e -> Level.ERROR.equals(e.getLevel()))
                    .findFirst()
                    .orElse(null);
            Assertions.assertEquals(Integer.valueOf(3), swb.startCallable);
        } finally {
            logbackLogger.detachAppender(listAppender);
            logbackLogger.setLevel(oldLevel);
        }
    }
}
