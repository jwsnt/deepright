package ai.open.right.netty;

import ai.open.right.protocol.ProtocolCode;

// Netty Stream对象写入
public interface NettyStream {

    // 已完成对象
    public static final NettyStream SUCCESS = new NettyStream() {

        @Override
        public Boolean isFinished() {
            return true;
        }

        @Override
        public Integer getCode() {
            return ProtocolCode.C200;
        }
    };

    public Boolean isFinished();

    public Integer getCode();
}
