package ai.open.right.netty.a2a.server;

import ai.open.right.netty.NettyWriter;
import ai.open.right.workflow.a2a.A2ARequest;
import ai.open.right.workflow.a2a.A2AResponse;
import com.fasterxml.jackson.annotation.JsonIgnore;
import io.netty.channel.ChannelHandlerContext;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
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
public class NettyA2ARequest implements A2ARequest {

    public static final String METHOD = "method";

    public static final String TRACE = "trace";

    public static final String ID = "id";

    @Builder.Default
    // 标记是否已经开启SSE模式
    protected volatile Boolean connectStream = false;

    @JsonIgnore
    protected ChannelHandlerContext context;

    protected Map<String, String> headers;

    protected Map<String, Object> content;

    protected String trace;

    protected String path;

    // 初始化
    public NettyA2ARequest init() {
        if (!CollectionUtils.isEmpty(this.headers)) {
            // 尝试从Header获取Trace
            this.trace = this.headers.get(NettyA2ARequest.TRACE);
        }
        return this;
    }

    public void setTrace(String trace) {
        this.trace = StringUtils.defaultString(this.trace, trace);
    }

    @Override
    public Map<String, String> getHeaders() {
        if (this.headers == null) {
            this.headers = new HashMap<String, String>();
        }
        return this.headers;
    }

    @Override
    public String getMethod() {
        // 从Content读取（Get Agent Card没有Content）
        if (!CollectionUtils.isEmpty(this.content)) {
            String method = String.class.cast(this.content.get(NettyA2ARequest.METHOD));
            Assert.notNull(method, "Method can not be empty");
            return StringUtils.upperCase(method);
        } else {
            return null;
        }
    }

    @Override
    public Object getId() {
        return this.content.get(NettyA2ARequest.ID);
    }

    @Override
    public void writeStream(A2AResponse response) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("A2a will stream the response={}", response);
        }
        // 开启SSE
        this.connectStream();
        NettyWriter.writeStream(this.context, response);
    }

    @Override
    // 回写报文
    public void writeOnce(Object response) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("A2a will directly write a response={}", response);
        }
        NettyWriter.writeOnce(this.context, response);
    }

    @Override
    public void close() throws Exception {
        NettyWriter.close(this.context);
    }

    // 开启SSE
    protected void connectStream() throws Exception {
        if (!this.connectStream) {
            NettyWriter.connectStream(this.context);
            this.connectStream = true;
        }
    }
}
