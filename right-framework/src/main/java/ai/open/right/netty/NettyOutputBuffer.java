package ai.open.right.netty;

import java.io.BufferedOutputStream;

import io.netty.buffer.ByteBuf;
import io.netty.buffer.ByteBufOutputStream;

/**
 * @author shenjiawei
 *
 */
public class NettyOutputBuffer extends BufferedOutputStream {

    public NettyOutputBuffer(ByteBuf byteBuf) {
        super(new ByteBufOutputStream(byteBuf));
    }

    public NettyOutputBuffer(ByteBuf byteBuf, int size) {
        super(new ByteBufOutputStream(byteBuf), size);
    }
}
