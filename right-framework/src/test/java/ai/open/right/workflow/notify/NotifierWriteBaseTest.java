package ai.open.right.workflow.notify;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.llm.Segment;
import org.junit.Assert;
import org.junit.Test;

import static org.junit.jupiter.api.Assertions.assertThrows;

public class NotifierWriteBaseTest {

    /** 用于测试 checkClosed 的已关闭分支，子类可设置 protected closed */
    private static class TestNotifierWriteBase extends NotifierWriteBase {
        void setClosedForTest(boolean value) {
            this.closed = value;
        }
    }

    @Test
    public void test() {
        NotifierWriteBase task = new NotifierWriteBase();
        Assert.assertFalse(task.containFunCallTrack());
        Assert.assertNull(task.getFunCallTrack());
        task.beginFunCallTrack("ABC");
        task.setCreated(10086L);
        Assert.assertEquals("ABC", task.getFunCallTrack());
        task.beginFunCallTrack();
        Assert.assertEquals(Integer.valueOf(36), Integer.valueOf(task.getFunCallTrack().length()));
        task.closeFunCallTrack();
        Assert.assertNull(task.getFunCallTrack());
        task.setTakeover("TK");
        Assert.assertEquals("TK", task.getTakeover());
        Assert.assertEquals(Long.valueOf(10086L), task.getCreated());
        task.setBiz("BIZ");
        task.setWorkflow("WORKFLOW");
        Assert.assertEquals("WORKFLOW", task.getWorkflow());
        Assert.assertEquals("BIZ", task.getBiz());
    }

    @Test
    public void testWrite() throws Exception {
        NotifierWriteBase task = new NotifierWriteBase();
        Segment segment = ObjectBuilder.buildSegment();
        task.writeSource(segment);
        task.writeBack(segment);
    }

    @Test
    public void testChat() {
        NotifierWriteBase task = new NotifierWriteBase();
        Assert.assertFalse(task.containChatTrack());
        task.beginChatTrack();
        Assert.assertTrue(task.containChatTrack());
    }

    @Test
    public void checkClosed_notClosed_doesNotThrow() throws Exception {
        NotifierWriteBase task = new NotifierWriteBase();
        task.checkClosed();
    }

    @Test
    public void checkClosed_whenClosed_throws() {
        TestNotifierWriteBase task = new TestNotifierWriteBase();
        task.setClosedForTest(true);
        WorkflowException exception = assertThrows(WorkflowException.class, () -> task.checkClosed());
        Assert.assertEquals(ProtocolCode.CN1, exception.getCode());
        Assert.assertTrue(exception.getSilent());
    }

    @Test
    public void ignoreClosed_thenCheckClosed_whenClosed_doesNotThrow() throws Exception {
        TestNotifierWriteBase task = new TestNotifierWriteBase();
        task.setClosedForTest(true);
        task.ignoreClosed();
        task.checkClosed();
    }

    @Test
    public void isClosed_defaultFalse() throws Exception {
        NotifierWriteBase task = new NotifierWriteBase();
        Assert.assertFalse(task.isClosed());
    }

    @Test
    public void isClosed_afterClose_returnsTrue() throws Exception {
        NotifierWriteBase task = new NotifierWriteBase();
        task.close();
        Assert.assertTrue(task.isClosed());
    }

    @Test
    public void close_setsClosed() throws Exception {
        NotifierWriteBase task = new NotifierWriteBase();
        Assert.assertFalse(task.isClosed());
        task.close();
        Assert.assertTrue(task.isClosed());
    }
}
