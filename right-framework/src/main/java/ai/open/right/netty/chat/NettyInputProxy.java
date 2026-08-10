package ai.open.right.netty.chat;

import ai.open.right.netty.NettyInputBuffer;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.utils.DumpUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.config.TokenMapping;
import io.netty.buffer.ByteBuf;
import io.netty.channel.ChannelHandlerContext;
import io.netty.util.ReferenceCountUtil;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;

import java.io.BufferedInputStream;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;

@Slf4j
public class NettyInputProxy implements AutoCloseable {

    protected ByteBuf byteBuf;

    protected String autoDump;

    public NettyInputProxy(ByteBuf buffer) {
        this(buffer, null);
    }

    public NettyInputProxy(ByteBuf buffer, String autoDump) {
        try {
            this.byteBuf = buffer.retain();
            this.autoDump = autoDump;
        } catch (Exception e) {
            this.close();
            throw e;
        }
    }

    // 构建NettyRequest
    public NettyRequest buildRequest(ChannelHandlerContext ctx, TokenMapping tokenMapping) throws Exception {
        try (BufferedInputStream input = new NettyInputBuffer(this.byteBuf)) {
            NettyRequest nettyRequest = !StringUtils.isEmpty(this.autoDump) ? JsonUtils.read(this.autoDump(IOUtils.toString(input, StandardCharsets.UTF_8)), NettyRequest.class) : JsonUtils.read(input, NettyRequest.class);
            if (log.isDebugEnabled()) {
                log.debug("Netty request={}", nettyRequest);
            }
            // 初始化
            return this.initRequest(ctx, nettyRequest);
        }
    }

    protected NettyRequest initRequest(ChannelHandlerContext ctx, NettyRequest request) throws Exception {
        // 绑定通道并初始化
        request.setChannel(ctx);
        return request.init();
    }

    protected NettyRequest buildRequest(InputStream inputstream) throws Exception {
        return JsonUtils.read(inputstream, NettyRequest.class);
    }

    protected NettyRequest buildRequest(String content) throws Exception {
        return JsonUtils.read(content, NettyRequest.class);
    }

    protected String autoDump(String content) throws Exception {
        if (!StringUtils.isEmpty(this.autoDump)) {
            DumpUtils.dump("REQUEST_CHAT", this.autoDump, ".json", content);
        }
        return content;
    }

    public void close() {
        if (this.byteBuf != null && this.byteBuf.refCnt() > 0) {
            ReferenceCountUtil.release(this.byteBuf);
            if (log.isDebugEnabled()) {
                log.debug("The release content ...");
            }
        }
    }
}
