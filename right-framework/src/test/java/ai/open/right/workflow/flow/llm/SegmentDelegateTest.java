package ai.open.right.workflow.flow.llm;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.UserContext;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.token.TokenData;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * SegmentDelegate 单元测试
 * 覆盖构造函数、元数据操作、内容操作、代理方法、状态重置及复制逻辑
 */
public class SegmentDelegateTest {

    @Test
    @DisplayName("测试构造函数与 WorkflowTask 的字段映射")
    public void testConstructorWithWorkflowTask() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        SegmentDelegate delegate = new SegmentDelegate(task);

        assertEquals(task.getQuery(), delegate.getContent());
        assertEquals(task.getWorkflow(), delegate.getWorkflow());
        assertEquals(task.getBiz(), delegate.getBiz());
        assertEquals(task.getUserContext(), delegate.getUserContext());
        assertEquals(task.getNotifier(), delegate.getNotifier());
        assertEquals(task.getDeepness(), delegate.getDeepness());
        assertEquals(task.getOriginal(), delegate.getOriginal());
        assertEquals(task.getPrevious(), delegate.getPrevious());
        assertEquals(task.getInitial(), delegate.getInitial());
    }

    @Test
    @DisplayName("isEntry 委托给内部 workTask")
    public void testIsEntryDelegatesToWorkTask() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        SegmentDelegate delegate = new SegmentDelegate(task);
        assertEquals(task.isEntry(), delegate.isEntry());
    }

    @Test
    @DisplayName("setDeepness 委托给内部 workTask")
    public void testSetDeepnessDelegatesToWorkTask() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        SegmentDelegate delegate = new SegmentDelegate(task);
        delegate.setDeepness(2);
        assertEquals(Integer.valueOf(2), task.getDeepness());
    }

    @Test
    @DisplayName("测试元数据操作：设置、合并、删除与懒加载")
    public void testMetadataOperations() {
        SegmentDelegate delegate = new SegmentDelegate();
        // 验证懒加载
        assertNotNull(delegate.getMetadata());
        assertTrue(delegate.getMetadata().isEmpty());

        // 单个 Key-Value 设置
        delegate.setMetadata("key1", "value1");
        assertEquals("value1", delegate.getMetadata().get("key1"));

        // Map 批量合并设置
        Map<String, Object> extra = new HashMap<>();
        extra.put("key2", "value2");
        delegate.setMetadata(extra);
        assertEquals("value1", delegate.getMetadata().get("key1"));
        assertEquals("value2", delegate.getMetadata().get("key2"));

        // 删除元数据
        delegate.delMetadata();
        // 再次获取应重新初始化为空 Map
        assertNotNull(delegate.getMetadata());
        assertTrue(delegate.getMetadata().isEmpty());
    }

    @Test
    @DisplayName("setMetadata(Map)：非空 map 与已有元数据合并（putAll）")
    public void setMetadataMap_nonEmpty_mergesWithExisting() {
        SegmentDelegate delegate = new SegmentDelegate();
        delegate.setMetadata("a", "1");
        Map<String, Object> extra = new HashMap<>();
        extra.put("b", "2");
        delegate.setMetadata(extra);
        assertEquals("1", delegate.getMetadata().get("a"));
        assertEquals("2", delegate.getMetadata().get("b"));
    }

    @Test
    @DisplayName("setMetadata(Map)：空 map 清空已有元数据")
    public void setMetadataMap_empty_clearsMetadata() {
        SegmentDelegate delegate = new SegmentDelegate();
        delegate.setMetadata("k", "v");
        assertFalse(delegate.getMetadata().isEmpty());
        delegate.setMetadata(new HashMap<>());
        assertTrue(delegate.getMetadata().isEmpty());
    }

    @Test
    @DisplayName("setMetadata(Map)：null 时断言失败")
    public void setMetadataMap_null_throws() {
        SegmentDelegate delegate = new SegmentDelegate();
        assertThrows(IllegalArgumentException.class, () -> delegate.setMetadata(null));
    }

    @Test
    @DisplayName("putMetadata(Map)：直接替换 metadata 引用")
    public void putMetadata_replacesMetadataReference() {
        SegmentDelegate delegate = new SegmentDelegate();
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("k", "v");

        delegate.putMetadata(metadata);

        assertSame(metadata, delegate.getMetadata());
        assertEquals("v", delegate.getMetadata().get("k"));
    }

    @Test
    @DisplayName("测试内容操作：设置、标记偏移量与缓冲区重置")
    public void testContentOperations() {
        SegmentDelegate delegate = new SegmentDelegate();
        delegate.setContent("Hello World");
        assertEquals("Hello World", delegate.getContent());
        assertEquals(0, delegate.getStart());

        // 标记当前位置作为后续读取的起点
        delegate.mark();
        assertEquals(11, delegate.getStart());
        assertEquals("", delegate.getContent());

        // 重新设置内容，应重置偏移量
        delegate.setContent("New Content");
        assertEquals("New Content", delegate.getContent());
        assertEquals(0, delegate.getStart());

        // 测试 contentBuffer 为 null 时的自动初始化
        delegate.setContentBuffer(null);
        delegate.setContent("Buffer Reinit");
        assertEquals("Buffer Reinit", delegate.getContent());
    }

    @Test
    @DisplayName("测试代理到 WorkflowTask 的方法")
    public void testDelegationMethods() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        SegmentDelegate delegate = new SegmentDelegate(task);

        assertEquals(task.getConversation(), delegate.getConversation());
        assertEquals(task.getDimension(), delegate.getDimension());
        assertEquals(task.getOriginal(), delegate.getOriginal());
        assertEquals(task.getCreated(), delegate.getTimestamp());
        assertEquals(task.getTrace(), delegate.getTrace());
        assertEquals(task.getChat(), delegate.getChat());

        if (task.getUserContext() != null) {
            assertEquals(task.getUserContext().getDevice(), delegate.getDevice());
        }
    }

    @Test
    @DisplayName("测试状态重置与初始化逻辑")
    public void testStatusAndReset() {
        SegmentDelegate delegate = new SegmentDelegate();
        assertFalse(delegate.isFinished());
        assertEquals(ProtocolCode.C200, delegate.getCode());

        delegate.setFinished(true);
        assertTrue(delegate.isFinished());

        // 测试 reset 方法
        delegate.reset(false, 10);
        assertFalse(delegate.isFinished());
        assertEquals(10, delegate.getIndex());

        // 测试 init 方法（清空关键状态）
        delegate.init();
        assertFalse(delegate.isFinished()); // null 映射为 false
        assertNull(delegate.getCode());
        assertNull(delegate.getNotifier());
        assertNull(delegate.getRole());
    }

    @Test
    @DisplayName("测试各种复制（Copy）方法")
    public void testCopyMethods() {
        SegmentDelegate delegate = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        delegate.setContent("Original Content");
        delegate.setWorkflow("wf_orig");
        delegate.setNotifier("notifier_orig");

        // 基础复制
        Segment copy = delegate.copy();
        assertEquals(delegate.getContent(), copy.getContent());
        assertEquals(delegate.getWorkflow(), copy.getWorkflow());

        // 指定 Workflow 复制
        Segment wfCopy = delegate.copyWithWorkflow("wf_new");
        assertEquals("wf_new", wfCopy.getWorkflow());
        assertEquals(delegate.getContent(), wfCopy.getContent());

        // 指定 Notifier 复制
        Segment nCopy = delegate.copyWithNotifier("notifier_new");
        assertEquals("notifier_new", nCopy.getNotifier());

        // 指定 Start 偏移量复制
        Segment sCopy = delegate.copyWithStart(5);
        assertEquals(5, sCopy.getStart());
        assertEquals(delegate.getContent().substring(5), sCopy.getContent());
    }

    @Test
    @DisplayName("copyWithId：保留原 id 与 workflow，但返回新对象")
    public void copyWithId_preservesIdAndWorkflow() {
        SegmentDelegate delegate = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        String originalId = delegate.getId();
        delegate.setWorkflow("wf-copy-id");

        Segment copy = delegate.copyWithId();

        assertInstanceOf(SegmentDelegate.class, copy);
        assertNotSame(delegate, copy);
        assertEquals(originalId, copy.getId());
        assertEquals("wf-copy-id", copy.getWorkflow());
    }

    @Test
    @DisplayName("测试 Usage 与其他 Getter/Setter 覆盖")
    public void testUsageAndOtherAccessors() {
        SegmentDelegate delegate = new SegmentDelegate();
        // 修复编译错误：使用 builder 模式构造 SegmentUsage
        SegmentUsage usage = new SegmentUsage(TokenData.builder().cache(1).total(1).build());
        delegate.setUsage(usage);
        assertEquals(usage, delegate.getUsage());

        delegate.setBiz("testBiz");
        assertEquals("testBiz", delegate.getBiz());

        delegate.setProtocol("HTTP");
        assertEquals("HTTP", delegate.getProtocol());

        delegate.setStream(false);
        assertFalse(delegate.getStream());

        delegate.setSilent(true);
        assertTrue(delegate.getSilent());

        delegate.setRole(SegmentDelegate.ROLE_QUERY);
        assertEquals(SegmentDelegate.ROLE_QUERY, delegate.getRole());
    }

    @Test
    @DisplayName("incrDeepness 委托给 WorkflowTask")
    public void testIncrDeepness_delegatesToWorkTask() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        Integer before = task.getDeepness();
        assertEquals(1, before);
        SegmentDelegate delegate = new SegmentDelegate(task);
        delegate.incrDeepness();
        assertEquals(2, task.getDeepness());
        assertEquals(2, task.getDeepness());
    }

    @Test
    @DisplayName("保留原有测试逻辑并转换为 JUnit 5")
    public void testOriginalLogic() {
        // 对应原 test()
        SegmentDelegate segmentDelegate = new SegmentDelegate(ObjectBuilder.buildLLMQuery());
        assertEquals("UNKNOWN", segmentDelegate.getContent());
        assertEquals(Integer.valueOf(200), Integer.valueOf(segmentDelegate.getCode()));
        UserContext userContext = UserContext.builder().build();
        segmentDelegate.setUserContext(userContext);
        assertEquals(userContext, segmentDelegate.getUserContext());
        assertFalse(segmentDelegate.isFinished());
        segmentDelegate.setFinished(true);
        assertTrue(segmentDelegate.isFinished());

        // 对应原 testDimension()
        SegmentDelegate dimensionDelegate = new SegmentDelegate(ObjectBuilder.buildWorkflowTask());
        assertEquals("UNKNOWN-UNKNOWN-UNKNOWN", dimensionDelegate.getDimension());

        // 对应原 testGetMetadataInit()
        SegmentDelegate metaDelegate = new SegmentDelegate();
        assertNotNull(metaDelegate.getMetadata());

        // 对应原 testSetContentNullBuffer()
        SegmentDelegate bufferDelegate = new SegmentDelegate();
        bufferDelegate.setContentBuffer(null);
        bufferDelegate.setContent("HELLO");
        assertEquals("HELLO", bufferDelegate.getContent());
    }

    @Test
    @DisplayName("getHistories：workTask 无 histories 时返回 null")
    public void getHistories_nullHistoriesOnTask_returnsNull() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        task.setHistories(null);
        SegmentDelegate delegate = new SegmentDelegate(task);
        assertNull(delegate.getHistories());
    }

    @Test
    @DisplayName("getHistories：空列表返回 null")
    public void getHistories_emptyHistories_returnsNull() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        task.setHistories(new ArrayList<>());
        SegmentDelegate delegate = new SegmentDelegate(task);
        assertNull(delegate.getHistories());
    }

    @Test
    @DisplayName("getHistories：仅返回 REFERENCE_INTERNAL 的历史")
    public void getHistories_returnsOnlyInternalHistories() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        History internal = new History();
        internal.setReference(History.REFERENCE_SERVER);
        History external = new History();
        external.setReference(History.REFERENCE_CLIENT);
        task.setHistories(Arrays.asList(internal, external));
        SegmentDelegate delegate = new SegmentDelegate(task);
        List<History> out = delegate.getHistories();
        assertNotNull(out);
        assertEquals(1, out.size());
        assertSame(internal, out.get(0));
    }

    @Test
    @DisplayName("getHistories：全部为外部引用时返回 null")
    public void getHistories_allExternal_returnsNull() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        History external = new History();
        external.setReference(History.REFERENCE_CLIENT);
        task.setHistories(Collections.singletonList(external));
        SegmentDelegate delegate = new SegmentDelegate(task);
        assertNull(delegate.getHistories());
    }
}
