package ai.open.right.netty;

import io.netty.util.concurrent.Future;
import io.netty.util.concurrent.GenericFutureListener;
import lombok.extern.slf4j.Slf4j;

/**
 * 错误告警
 *
 * @author shenjiawei
 */
@Slf4j
public class NettyAlarm implements GenericFutureListener<Future<Void>> {

    public static final NettyAlarm INSTANCE = new NettyAlarm();

    protected NettyAlarm() {

    }

    @Override
    public void operationComplete(Future<Void> future) throws Exception {
        if (!future.isSuccess() && future.cause() != null) {
            if (log.isDebugEnabled()) {
                log.debug(future.cause().getMessage(), future.cause());
            }
        }
    }
}