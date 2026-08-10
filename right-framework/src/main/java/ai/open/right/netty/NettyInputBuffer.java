package ai.open.right.netty;

import io.netty.buffer.ByteBuf;
import io.netty.buffer.ByteBufInputStream;
import lombok.extern.slf4j.Slf4j;

import java.io.BufferedInputStream;

/**
 * @author shenjiawei
 */
@Slf4j
public class NettyInputBuffer extends BufferedInputStream {

    // 主动释放ByteBuf
    public NettyInputBuffer(ByteBuf byteBuf) {
        super(new ByteBufInputStream(byteBuf, true));
    }

    public NettyInputBuffer(ByteBuf byteBuf, int size) {
        super(new ByteBufInputStream(byteBuf, true), size);
    }
}
