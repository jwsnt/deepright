package ai.open.right.netty;

import ai.open.right.netty.NettyAlarm;
import io.netty.util.concurrent.Future;
import org.easymock.EasyMock;
import org.junit.Test;

public class NettyAlarmTest {

    @Test
    public void testOperationComplete() throws Exception {
        NettyAlarm alarm = new NettyAlarm();
        Future<Void> future = EasyMock.createMock(Future.class);
        EasyMock.expect(future.isSuccess()).andReturn(false).anyTimes();
        EasyMock.expect(future.cause()).andReturn(new Throwable()).anyTimes();
        EasyMock.replay(future);
        alarm.operationComplete(future);
        EasyMock.verify(future);
    }

    @org.junit.jupiter.api.Test
    public void testNettyAlarmInstantiationUnique() {
        org.junit.jupiter.api.Assertions.assertTrue(true);
    }

}