package ai.open.right.netty.chat;

import ai.open.right.ObjectBuilder;
import ai.open.right.netty.chat.server.http.NettyErrorSegment;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.SegmentDelegate;
import ai.open.right.workflow.flow.script.ScriptResponse;
import ai.open.right.workflow.flow.script.ScriptSegment;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertTrue;

/**
 * NettySegment 在生产代码中的实现均需提供稳定的 getBiz 语义；此处按实现类分别覆盖。
 * <p>
 * 直接实现 {@link NettySegment}：{@link NettyErrorSegment}。<br>
 * 通过 {@link ai.open.right.workflow.flow.llm.Segment} 间接实现：{@link SegmentDelegate}、{@link ScriptSegment}。
 */
class NettySegmentGetBizTest {

    @Test
    @DisplayName("NettyErrorSegment#getBiz 固定为 null")
    void nettyErrorSegment_getBiz_returnsNull() {
        NettySegment segment = NettyErrorSegment.builder().content("e").code(500).build();
        assertNull(segment.getBiz());
    }

    @Test
    @DisplayName("SegmentDelegate#getBiz 与构造时 WorkflowTask.biz 一致")
    void segmentDelegate_getBiz_followsWorkflowTask() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        task.setBiz("cover-biz");
        NettySegment segment = new SegmentDelegate(task);
        assertEquals("cover-biz", segment.getBiz());
    }

    @Test
    @DisplayName("ScriptSegment#getBiz 委托内部 Segment")
    void scriptSegment_getBiz_delegatesToInnerSegment() throws Exception {
        WorkflowTask task = ObjectBuilder.buildWorkflowTaskWithTimestamp(1L);
        task.setBiz("script-biz");
        ScriptSegment scriptSegment = new ScriptSegment(task, ScriptResponse.builder()
                .data("x")
                .code(200)
                .build());
        assertTrue(scriptSegment instanceof NettySegment);
        assertEquals("script-biz", ((NettySegment) scriptSegment).getBiz());
    }

    @Test
    @DisplayName("ObjectBuilder 空 NettySegment 桩：getBiz 为 null（与 NettyErrorSegment 行为对齐）")
    void objectBuilderEmptyNettySegment_getBiz_returnsNull() {
        NettySegment segment = ObjectBuilder.buildEmptyNettySegment();
        assertNull(segment.getBiz());
    }
}
