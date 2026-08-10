package ai.open.right.netty.mcp.distribute;

import ai.open.right.netty.NettyCaptor;
import ai.open.right.netty.mcp.NettyInputProxy;
import ai.open.right.netty.mcp.server.NettyMcpRequest;
import ai.open.right.trace.TraceService;
import ai.open.right.workflow.mcp.server.McpDistributor;
import io.netty.channel.ChannelHandlerContext;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.scheduling.annotation.Async;

@Slf4j
@Setter
@Getter
public class NettyDistributor {

    public static final String NAME = "NettyMcpDistributor";

    // 分发MCP请求
    protected McpDistributor mcpDistributor;

    // 生成Trace
    protected TraceService traceService;

    @Async("executor")
    public void distribute(ChannelHandlerContext context, NettyInputProxy proxy, NettyCaptor captor) throws Exception {
        try (NettyInputProxy input = proxy) {
            // 初始化
            NettyMcpRequest request = this.buildRequest(context, input);
            if (log.isDebugEnabled()) {
                log.debug("Received request={} ", request);
            }
            // 追加Trace
            request.setTrace(this.traceService.getTrace(request.getTrace()));
            this.mcpDistributor.distribute(request);
        } catch (Exception e) {
            captor.exceptionCaught(context, e);
        }
    }

    protected NettyMcpRequest buildRequest(ChannelHandlerContext context, NettyInputProxy input) throws Exception {
        return input.buildContent(context);
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Configuration("NettyMcpInitConfig")
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected McpDistributor mcpDistributor;

        @Autowired
        protected TraceService traceService;

        @Bean(name = NettyDistributor.NAME)
        @ConditionalOnMissingBean(value = NettyDistributor.class)
        public NettyDistributor nettyDistributor() throws Exception {
            NettyDistributor nettyDistributor = new NettyDistributor();
            BeanUtils.copyProperties(this, nettyDistributor);
            log.info("NettyDistributor inited");
            return nettyDistributor;
        }
    }
}
