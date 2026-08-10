package ai.open.right.netty.chat.server.http;

import ai.open.right.netty.NettyInputBuffer;
import ai.open.right.netty.NettyWriter;
import ai.open.right.netty.chat.NettyInputProxy;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.netty.chat.server.NettyAttributes;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.config.TokenEntry;
import ai.open.right.workflow.config.TokenMapping;
import io.netty.buffer.ByteBuf;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.HttpHeaders;
import io.netty.util.ReferenceCountUtil;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.ArrayUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;

import java.io.BufferedInputStream;
import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.HashMap;
import java.util.Map;

@Slf4j
public class NettyHttpProxy extends NettyInputProxy {

    protected final Map<String, String> headers = new HashMap<String, String>();

    protected final String device;

    protected final String token;

    protected final String chat;

    public NettyHttpProxy(ByteBuf buffer, String token) {
        this(null, buffer, token, null);
    }

    public NettyHttpProxy(HttpHeaders headers, ByteBuf buffer, String token, String autodump) {
        super(buffer, autodump);
        try {
            // 用于切分Token(token/device:chat)
            // token/device:chat或biz@workflow/device:chat
            String[] parts = StringUtils.split(token, SplitUtils.SPLIT_SLASH);
            this.token = !ArrayUtils.isEmpty(parts) ? parts[0] : "";
            if (log.isDebugEnabled()) {
                log.debug("Splitting token={}-{}", Arrays.toString(parts), this.token);
            }
            if (parts.length == 2) {
                // 拆解Device/Chat
                String[] info = StringUtils.split(parts[1], ":");
                Assert.isTrue(info.length == 2, "Device and chat can not be empty, please verify if it conforms to the pattern of `token/device:chat` or `biz@workflow/device:chat`");
                this.device = info[0];
                this.chat = info[1];
            } else {
                this.device = null;
                this.chat = null;
            }
            // 填充Header
            if (headers != null) {
                for (Map.Entry<String, String> header : headers) {
                    this.headers.put(header.getKey(), header.getValue());
                }
            }
        } catch (RuntimeException e) {
            // super 内已对 buffer retain，子类初始化失败时对称释放，避免泄漏
            ReferenceCountUtil.safeRelease(buffer);
            throw e;
        }
    }

    // Http通道报文
    @Override
    public NettyRequest buildRequest(ChannelHandlerContext ctx, TokenMapping tokenMapping) throws Exception {
        try (BufferedInputStream input = new NettyInputBuffer(this.byteBuf)) {
            NettyHttpRequest httpRequest = null;
            if (!StringUtils.isEmpty(this.autoDump)) {
                httpRequest = JsonUtils.read(this.autoDump(IOUtils.toString(input, StandardCharsets.UTF_8)), NettyHttpRequest.class);
            } else {
                httpRequest = JsonUtils.read(input, NettyHttpRequest.class);
            }
            // 标记需要响应的类型（Once/Stream）
            NettyWriter.flagConnection(ctx, httpRequest.getStream() ? NettyAttributes.CONNECTION_STREAM : NettyAttributes.CONNECTION_ONCE);
            // 转为抽象的WorkflowTask（NettyRequest实现）
            NettyRequest request = httpRequest.buildNettyRequest(this.chat, this.device, (Map<String, Object>) (Map<String, ?>) this.headers);
            // 从Token解析Workflow和Biz
            TokenEntry tokenEntry = tokenMapping.entry(request, this.token);
            request.setWorkflow(tokenEntry.getWorkflow());
            request.setBiz(tokenEntry.getBiz());
            // 初始化
            return this.initRequest(ctx, request);
        }
    }
}
