package ai.open.right.netty.mcp.server;

import ai.open.right.netty.NettyWriter;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.mcp.server.McpRequest;
import ai.open.right.workflow.mcp.server.McpResponse;
import com.fasterxml.jackson.annotation.JsonIgnore;
import com.google.common.collect.ImmutableMap;
import io.netty.channel.ChannelHandlerContext;
import lombok.*;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.util.HashMap;
import java.util.Map;

@Builder
@Setter
@Getter
@Slf4j
@NoArgsConstructor
@AllArgsConstructor
public class NettyMcpRequest implements McpRequest {

    public static final String METHOD = "method";

    public static final String TRACE = "trace";

    public static final String ID = "id";

    @JsonIgnore
    protected ChannelHandlerContext context;

    protected Map<String, String> headers;

    protected Map<String, Object> content;

    protected String trace;

    // 初始化
    public NettyMcpRequest init() {
        if (!CollectionUtils.isEmpty(this.headers)) {
            // 尝试从Header获取Trace
            this.trace = this.headers.get(NettyMcpRequest.TRACE);
        }
        return this;
    }

    public void setTrace(String trace) {
        this.trace = StringUtils.defaultString(this.trace, trace);
    }

    public Map<String, String> getHeaders() {
        if (this.headers == null) {
            this.headers = new HashMap<String, String>();
        }
        return this.headers;
    }

    public String getMethod() {
        String method = String.class.cast(this.content.get(NettyMcpRequest.METHOD));
        Assert.notNull(method, "Method can not be empty");
        return StringUtils.upperCase(method);
    }

    public Object getId() {
        return this.content.get(NettyMcpRequest.ID);
    }

    // 回写报文
    public void write(McpResponse response) throws Exception {
        // 仅响应非Notifier
        if (!response.getNotifier()) {
            // 是否需要包装Result
            Map<String, Object> result = response.getWrap() ? ImmutableMap.of("jsonrpc", "2.0", "id", this.getId(), "result", response.getResult()) : response.getResult();
            if (log.isDebugEnabled()) {
                log.debug("Mcp will write response={}", result);
            }
            NettyWriter.writeOnce(this.context, result);
        }
    }

    // 回写错误
    public void error(String message) throws Exception {
        Map<String, Object> result = ImmutableMap.of("jsonrpc", "2.0", "id", this.getId(), "error", ImmutableMap.of("code", ProtocolCode.C500, "message", message));
        if (log.isDebugEnabled()) {
            log.debug("Mcp will write error={}", result);
        }
        NettyWriter.writeOnce(this.context, result);
    }
}
