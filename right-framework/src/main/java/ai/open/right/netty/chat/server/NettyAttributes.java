package ai.open.right.netty.chat.server;

import io.netty.util.AttributeKey;

// Netty通道Flag Key
public interface NettyAttributes {

    // 通道类型（Once/Stream）
    public final static AttributeKey<Byte> CONNECTION_TYPE = AttributeKey.newInstance("CONNECTION_TYPE");

    // 服务类型（WS/HTTP）
    public final static AttributeKey<Byte> SERVER_TYPE = AttributeKey.newInstance("SERVER_TYPE");

    // 是否支持跨域
    public final static AttributeKey<Byte> CORS_TYPE = AttributeKey.newInstance("CORS_TYPE");

    // 是否为SSE
    public final static AttributeKey<Byte> SSE_TYPE = AttributeKey.newInstance("SSE_TYPE");

    public final static Byte CONNECTION_STREAM = 0;

    public final static Byte CONNECTION_ONCE = 1;

    public final static Byte SERVER_HTTP = 0;

    public final static Byte SERVER_WS = 1;

    public final static Byte HTTP_CORS = 1;

    public final static Byte HTTP_SSE = 1;
}
