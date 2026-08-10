package ai.open.right.netty.mcp;

import ai.open.right.netty.NettyInputBuffer;
import ai.open.right.netty.mcp.server.NettyMcpRequest;
import ai.open.right.utils.JsonUtils;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.FullHttpRequest;
import io.netty.handler.codec.http.HttpHeaders;
import io.netty.util.ReferenceCountUtil;
import lombok.Getter;
import lombok.extern.slf4j.Slf4j;

import java.io.BufferedInputStream;
import java.io.Closeable;
import java.util.HashMap;
import java.util.Map;

@Slf4j
@Getter
public class NettyInputProxy implements Closeable {

    protected Map<String, Object> content;

    protected FullHttpRequest request;

    public NettyInputProxy(FullHttpRequest request) throws Exception {
        try (BufferedInputStream input = new NettyInputBuffer((this.request = request.retain()).content())) {
            this.content = JsonUtils.read(input, Map.class);
            if (log.isInfoEnabled()) {
                log.info("Mcp request content={}", this.content);
            }
        } catch (Exception e) {
            this.close();
            throw e;
        }
    }

    // 构建MCP请求
    public NettyMcpRequest buildContent(ChannelHandlerContext context) throws Exception {
        return NettyMcpRequest.builder()
                .content(this.initContent())
                .headers(this.initHeaders())
                .context(context)
                .build().init();
    }

    protected Map<String, String> initHeaders() throws Exception {
        // 初始化Header
        Map<String, String> headers = new HashMap<String, String>();
        HttpHeaders netty = this.request.headers();
        if (netty != null) {
            for (Map.Entry<String, String> header : netty) {
                headers.put(header.getKey(), header.getValue());
            }
        }
        return headers;
    }

    protected Map<String, Object> initContent() throws Exception {
        return this.content;
    }

    public void close() {
        if (this.request != null && this.request.refCnt() > 0) {
            ReferenceCountUtil.release(this.request);
            if (log.isDebugEnabled()) {
                log.debug("Release request ...");
            }
        }
    }
}
