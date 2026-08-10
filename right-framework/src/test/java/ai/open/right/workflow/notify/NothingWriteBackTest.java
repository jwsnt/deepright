package ai.open.right.workflow.notify;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import org.junit.Assert;
import org.junit.Test;

import static org.junit.jupiter.api.Assertions.assertThrows;

public class NothingWriteBackTest {

    @Test
    public void isClosed_defaultFalse() throws Exception {
        NothingWriteBack nwb = new NothingWriteBack();
        Assert.assertFalse(nwb.isClosed());
    }

    @Test
    public void isClosed_afterClose_returnsTrue() throws Exception {
        NothingWriteBack nwb = new NothingWriteBack();
        nwb.close();
        Assert.assertTrue(nwb.isClosed());
    }

    @Test
    public void close_setsClosed() throws Exception {
        NothingWriteBack nwb = new NothingWriteBack();
        Assert.assertFalse(nwb.isClosed());
        nwb.close();
        Assert.assertTrue(nwb.isClosed());
    }

    @Test
    public void checkClosed_whenClosed_throwsSilentCn1() throws Exception {
        NothingWriteBack nwb = new NothingWriteBack();
        nwb.close();
        WorkflowException exception = assertThrows(WorkflowException.class, nwb::checkClosed);
        Assert.assertEquals(ProtocolCode.CN1, exception.getCode());
        Assert.assertTrue(exception.getSilent());
    }
}
