package ai.open.right.netty.a2a;

import ai.open.right.netty.NettyInputBuffer;
import ai.open.right.netty.a2a.server.NettyA2ARequest;
import ai.open.right.utils.DumpUtils;
import ai.open.right.utils.JsonUtils;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.FullHttpRequest;
import io.netty.handler.codec.http.HttpHeaders;
import io.netty.handler.codec.http.HttpMethod;
import io.netty.util.ReferenceCountUtil;
import lombok.Getter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;

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
        this(request, null);
    }

    public NettyInputProxy(FullHttpRequest request, String autodump) throws Exception {
        try {
            this.request = request.retain();
            if (this.request.method() == HttpMethod.POST) {
                // 由Netty上层控制大小，避免一次读取的OOM (GET/POST) 一体
                //
                try (BufferedInputStream input = new NettyInputBuffer(this.request.content())) {
                    this.content = JsonUtils.read(input, Map.class);
                    this.dumpInboundIfAutodump(autodump);
                    if (log.isInfoEnabled()) {
                        log.info("A2A request content={}", this.content);
                    }
                }
            } else {
                this.content = null;
            }
        } catch (Exception e) {
            this.close();
            throw e;
        }
    }

    // 构建MCP请求
    public NettyA2ARequest buildContent(ChannelHandlerContext context) throws Exception {
        return NettyA2ARequest.builder().content(this.initContent()).headers(this.initHeaders()).path(this.request.uri()).context(context).build().init();
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

    protected void dumpInboundIfAutodump(String directory) throws Exception {
        if (!StringUtils.isEmpty(directory)) {
            DumpUtils.dump("REQUEST_A2A", directory, ".json", JsonUtils.write(this.content));
        }
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
