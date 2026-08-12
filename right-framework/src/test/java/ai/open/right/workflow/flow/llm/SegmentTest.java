package ai.open.right.workflow.flow.llm;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.protocol.Protocol;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.notify.Notifier;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for Segment and its inner classes.
 * Converted to JUnit 5 and expanded coverage.
 */
public class SegmentTest {

    @Test
    public void testSetMetadata1() {
        SegmentDelegate s = new SegmentDelegate();
        s.setMetadata("KEY", "VAL");
        assertEquals("VAL", s.getMetadata().get("KEY"));
    }

    @Test
    public void testSetMetadata2() {
        SegmentDelegate s = new SegmentDelegate();
        Map<String, Object> mt = new HashMap<>();
        mt.put("KEY", "VAL");
        s.setMetadata(mt);
        assertEquals("VAL", s.getMetadata().get("KEY"));
    }

    @Test
    public void testDelMetadata1() {
        SegmentDelegate s = new SegmentDelegate();
        s.setMetadata("KEY", "VAL");
        s.delMetadata();
        assertTrue(s.getMetadata().isEmpty());
    }

    @Test
    public void testContentWithNull() {
        SegmentDelegate s = new SegmentDelegate();
        s.setContent("HELLO WORLD");
        assertEquals(11, s.getContent().length());
        s.mark();

        // 设置 null 后应清空内容，后续 mark 不应因 contentBuffer 为 null 抛出异常
        s.setContent(null);
        assertDoesNotThrow(s::mark);
        assertNull(s.getContent());
        assertNull(s.getStart());

        // 未初始化内容时，读取应返回 null
        assertNull(new SegmentDelegate().getContent());
    }

    @Test
    public void constructorWithNullQueryKeepsContentNull() {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(null);
        SegmentDelegate segment = new SegmentDelegate(workflowTask);

        assertNull(segment.getContent());
        assertDoesNotThrow(segment::mark);
        assertNull(segment.getStart());
    }

    @Test
    public void testConfig() {
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("KEY_1", "VAL_1");
        Segment segment = Segment.build(ObjectBuilder.buildWorkflowTask(), Segment.SegmentConfig.builder()
                .metadata(metadata)
                .upstream("HELLO")
                .pureMeta(true)
                .deepness(100)
                .build());
        assertEquals(1, segment.getMetadata().size());
        assertEquals("HELLO", segment.getUpstream());
        assertEquals(Integer.valueOf(100), segment.getDeepness());
        assertEquals("VAL_1", segment.getMetadata().get("KEY_1"));
    }

    @Test
    public void testClone1() {
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("KEY_1", "VAL_1");
        Segment segment = Segment.build(ObjectBuilder.buildWorkflowTask(), Segment.SegmentConfig.builder()
                .metadata(metadata)
                .upstream("HELLO")
                .pureMeta(true)
                .deepness(100)
                .build());
        segment = segment.copyWithWorkflow("WORKFLOW2");
        assertEquals("WORKFLOW2", segment.getWorkflow());
        assertEquals(1, segment.getMetadata().size());
        assertEquals("HELLO", segment.getUpstream());
        assertEquals(Integer.valueOf(100), segment.getDeepness());
        assertEquals("VAL_1", segment.getMetadata().get("KEY_1"));
    }

    @Test
    public void testClone2() {
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("KEY_1", "VAL_1");
        Segment segment = Segment.build(ObjectBuilder.buildWorkflowTask(), Segment.SegmentConfig.builder()
                .metadata(metadata)
                .upstream("HELLO")
                .pureMeta(true)
                .deepness(100)
                .build());
        segment = segment.copy();
        assertEquals("UNKNOWN", segment.getWorkflow());
        assertEquals(1, segment.getMetadata().size());
        assertEquals("HELLO", segment.getUpstream());
        assertEquals(Integer.valueOf(100), segment.getDeepness());
        assertEquals("VAL_1", segment.getMetadata().get("KEY_1"));
    }

    @Test
    public void testClone3() {
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("KEY_1", "VAL_1");
        Segment segment = Segment.build(ObjectBuilder.buildWorkflowTask(), Segment.SegmentConfig.builder()
                .metadata(metadata)
                .upstream("HELLO")
                .pureMeta(true)
                .deepness(100)
                .build());
        segment = segment.copyWithNotifier("NOTIFIER");
        assertEquals("NOTIFIER", segment.getNotifier());
    }

    @Test
    public void testFailed() {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        Exception exception = new Exception(new RuntimeException(new WorkflowException("OK")));
        Segment segment = Segment.failed(workflowTask, exception, Notifier.ENDPOINT, 501);
        assertEquals("OK", segment.getContent());
        assertEquals(Integer.valueOf(501), segment.getCode());
    }

    @Test
    public void testFailedWithMessage() {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        String message = "Custom Error Message";
        Integer code = 502;
        Segment segment = Segment.failed(workflowTask, message, Notifier.ENDPOINT, code);

        assertEquals(message, segment.getContent());
        assertEquals(code, segment.getCode());
        assertEquals(Notifier.ENDPOINT, segment.getNotifier());
    }

    @Test
    public void testFailedWithNullMessageBuildsNullContentSegment() {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(null);

        Segment segment = Segment.failed(workflowTask, (String) null, Notifier.ENDPOINT, ProtocolCode.C500);

        assertNull(segment.getContent());
    }

    @Test
    public void testBuildDefault() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        Segment.SegmentConfig config = Segment.SegmentConfig.builder().build();
        Segment segment = Segment.build(task, config);
        assertNotNull(segment.getNotifier());
        assertNotNull(segment.getCode());
    }

    @Test
    public void testFailedNoCause() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        Exception ex = new Exception("ERROR");
        Segment segment = Segment.failed(task, ex, Notifier.ENDPOINT, 500);
        assertEquals("ERROR", segment.getContent());
    }

    @Test
    public void testSegmentConfigSetDefault() {
        // 测试 SegmentConfig.setDefault 逻辑，手动将字段设为 null
        Segment.SegmentConfig config = Segment.SegmentConfig.builder()
                .pureMeta(null)
                .finished(null)
                .protocol(null)
                .stream(null)
                .index(null)
                .code(null)
                .build();

        config.setDefault();

        assertEquals(false, config.getPureMeta());
        assertEquals(true, config.getFinished());
        assertEquals(Protocol.CHAT, config.getProtocol());
        assertEquals(false, config.getStream());
        assertEquals(0, config.getIndex());
        assertEquals(ProtocolCode.C200, config.getCode());
    }

    @Test
    public void testSegmentCheckerCheck() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        Segment segment = Segment.build(task, Segment.SegmentConfig.builder().build());

        // 正常情况不应抛出异常
        assertDoesNotThrow(() -> Segment.SegmentChecker.check(segment));
    }

    @Test
    public void testSegmentCheckerCheckFails() {
        // 使用有效的 WorkflowTask 构建基础 Segment
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();

        // 测试 workflow 为空的情况
        SegmentDelegate s1 = new SegmentDelegate(task);
        s1.setWorkflow(null);
        assertThrows(IllegalArgumentException.class, () -> Segment.SegmentChecker.check(s1));

        // 测试 biz 为空的情况
        SegmentDelegate s2 = new SegmentDelegate(task);
        s2.setBiz(null);
        assertThrows(IllegalArgumentException.class, () -> Segment.SegmentChecker.check(s2));

        // 测试 userContext 为空的情况
        SegmentDelegate s3 = new SegmentDelegate(task);
        s3.setUserContext(null);
        assertThrows(IllegalArgumentException.class, () -> Segment.SegmentChecker.check(s3));

        // 测试 notifier 为空的情况
        SegmentDelegate s4 = new SegmentDelegate(task);
        s4.setNotifier(null);
        assertThrows(IllegalArgumentException.class, () -> Segment.SegmentChecker.check(s4));

        // 测试 deepness 为空的情况
        SegmentDelegate s5 = new SegmentDelegate(task);
        s5.setDeepness(null);
        assertThrows(IllegalArgumentException.class, () -> Segment.SegmentChecker.check(s5));

        // 测试 conversation 为空的情况 (通过修改 task)
        NettyRequest taskNoConv = (NettyRequest) ObjectBuilder.buildWorkflowTask();
        taskNoConv.setConversation(null);
        SegmentDelegate s6 = new SegmentDelegate(taskNoConv);
        assertThrows(IllegalArgumentException.class, () -> Segment.SegmentChecker.check(s6));

        // 测试 chat 为空的情况 (通过修改 task)
        NettyRequest taskNoChat = (NettyRequest) ObjectBuilder.buildWorkflowTask();
        taskNoChat.setChat(null);
        SegmentDelegate s7 = new SegmentDelegate(taskNoChat);
        assertThrows(IllegalArgumentException.class, () -> Segment.SegmentChecker.check(s7));
    }

    @Test
    public void testBuildWithPureMeta() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        // 修复：使用 putMetadata 设置初始元数据
        task.putMetadata("OLD_KEY", "OLD_VAL");

        Map<String, Object> newMeta = new HashMap<>();
        newMeta.put("NEW_KEY", "NEW_VAL");

        Segment.SegmentConfig config = Segment.SegmentConfig.builder()
                .pureMeta(true)
                .metadata(newMeta)
                .build();

        Segment segment = Segment.build(task, config);

        // 当 pureMeta 为 true 时，task 中的旧 metadata 应该被清理（不包含在 segment 中）
        assertNull(segment.getMetadata().get("OLD_KEY"));
        assertEquals("NEW_VAL", segment.getMetadata().get("NEW_KEY"));
    }

    @Test
    public void testFailedWithNullMessage() {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        // 模拟异常及其 Cause 的 getMessage() 均为 null 的情况
        Exception cause = new Exception((String) null);
        Exception exception = new Exception(null, cause);

        Segment segment = Segment.failed(workflowTask, exception, Notifier.ENDPOINT, 500);

        // 根据 Segment.failed 的逻辑，如果 message 为空，则使用 exception.getClass().getSimpleName()
        assertEquals("Exception", segment.getContent());
        assertEquals(Integer.valueOf(500), segment.getCode());
    }

    @Test
    public void testSegmentConfigSetDefaultWithMixedNulls() {
        // 混合 null 和非 null 字段调用 setDefault
        Segment.SegmentConfig config = Segment.SegmentConfig.builder()
                .pureMeta(null)
                .finished(false) // 非 null
                .protocol(null)
                .stream(true)    // 非 null
                .index(null)
                .code(ProtocolCode.C400) // 非 null
                .build();

        config.setDefault();

        assertEquals(false, config.getPureMeta()); // null -> false
        assertEquals(false, config.getFinished()); // 保持 false
        assertEquals(Protocol.CHAT, config.getProtocol()); // null -> Protocol.CHAT
        assertEquals(true, config.getStream()); // 保持 true
        assertEquals(0, config.getIndex()); // null -> 0
        assertEquals(ProtocolCode.C400, config.getCode()); // 保持 C400
    }

    @Test
    public void testCopyWithStart() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        SegmentDelegate segment = new SegmentDelegate(task);
        segment.setContent("HELLO WORLD");

        Segment copied = segment.copyWithStart(6);

        assertEquals("WORLD", copied.getContent());
        assertEquals(Integer.valueOf(6), copied.getStart());
    }

    // SegmentConfig Lombok: @Getter + @Builder + @Builder.Default 覆盖
    @Test
    public void testSegmentConfigBuilderAndGetters() {
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("K", "V");
        StringBuffer content = new StringBuffer("content");
        String notifier = Notifier.ENDPOINT;
        String workflow = "wf";
        String upstream = "up";
        String protocol = Protocol.CHAT;

        Segment.SegmentConfig config = Segment.SegmentConfig.builder()
                .metadata(metadata)
                .content(content)
                .pureMeta(true)
                .finished(false)
                .deepness(1)
                .notifier(notifier)
                .workflow(workflow)
                .upstream(upstream)
                .protocol(protocol)
                .stream(true)
                .index(2)
                .code(ProtocolCode.C200)
                .build();

        assertSame(metadata, config.getMetadata());
        assertSame(content, config.getContent());
        assertEquals(true, config.getPureMeta());
        assertEquals(false, config.getFinished());
        assertEquals(Integer.valueOf(1), config.getDeepness());
        assertEquals(notifier, config.getNotifier());
        assertEquals(workflow, config.getWorkflow());
        assertEquals(upstream, config.getUpstream());
        assertEquals(protocol, config.getProtocol());
        assertEquals(true, config.getStream());
        assertEquals(Integer.valueOf(2), config.getIndex());
        assertEquals(ProtocolCode.C200, config.getCode());
    }

    @Test
    public void testSegmentConfigBuilderDefaultNotifier() {
        Segment.SegmentConfig config = Segment.SegmentConfig.builder().build();
        assertEquals(Notifier.LOCALHOST, config.getNotifier());
    }

    @Test
    public void testSegmentConfigPartialBuilder() {
        Segment.SegmentConfig config = Segment.SegmentConfig.builder()
                .pureMeta(true)
                .deepness(10)
                .build();
        assertNull(config.getMetadata());
        assertNull(config.getContent());
        assertEquals(true, config.getPureMeta());
        assertNull(config.getFinished());
        assertEquals(Integer.valueOf(10), config.getDeepness());
        assertEquals(Notifier.LOCALHOST, config.getNotifier());
        assertNull(config.getWorkflow());
        assertNull(config.getUpstream());
        assertNull(config.getProtocol());
        assertNull(config.getStream());
        assertNull(config.getIndex());
        assertNull(config.getCode());
    }
}
