package ai.open.right.netty.mcp.server;

import ai.open.right.netty.NettyServer;
import io.netty.channel.ChannelHandler;
import io.netty.handler.codec.http.HttpObjectAggregator;
import io.netty.handler.codec.http.HttpServerCodec;
import io.netty.handler.timeout.IdleStateHandler;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

/**
 * @author shenjiawei
 */
@Slf4j
@Setter
@Getter
public class NettyMcpServer extends NettyServer {

    protected Integer eventLoopChildren = -1;

    protected Integer eventLoopParent = -1;

    protected Integer maxInitialLineLength = 4096;

    protected Integer maxHeaderSize = 8192;

    protected Integer maxChunkSize = 8192;

    protected Integer requestMax = -1;

    protected String binding;

    protected Integer idle;

    protected Integer port;

    @Autowired
    protected NettyMcpHandler mcpHandler;

    @Override
    protected ProxyChannelInitializer buildHandler() {
        return new McpInitializer(this.bufferRecv, this.bufferSend);
    }

    @Override
    protected Integer getEventLoopChildren() {
        return this.eventLoopChildren;
    }

    @Override
    protected Integer getEventLoopParent() {
        return this.eventLoopParent;
    }

    @Override
    protected String getBinding() {
        return this.binding;
    }

    @Override
    protected Integer getPort() {
        return this.port;
    }

    public class McpInitializer extends ProxyChannelInitializer {

        public McpInitializer(Integer bufferRecv, Integer bufferSend) {
            super(bufferRecv, bufferSend);
        }

        @Override
        public ChannelHandler[] getChannelHandler() {
            ChannelHandler[] handlers = new ChannelHandler[5];
            handlers[0] = new IdleStateHandler(NettyMcpServer.this.idle, NettyMcpServer.this.idle, NettyMcpServer.this.idle);
            handlers[1] = new HttpServerCodec(NettyMcpServer.this.maxInitialLineLength, NettyMcpServer.this.maxHeaderSize, NettyMcpServer.this.maxChunkSize);
            handlers[2] = new HttpObjectAggregator(NettyMcpServer.this.requestMax == -1 ? Integer.MAX_VALUE : NettyMcpServer.this.requestMax);
            handlers[3] = NettyMcpServer.this.mcpHandler;
            return handlers;
        }
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Setter
    @Getter
    @Configuration
    public static class InitConfig extends NettyServerInitConfig {

        @Value("${mcp.eventloop.children:-1}")
        protected Integer eventLoopChildren = -1;

        @Value("${mcp.eventloop.parent:-1}")
        protected Integer eventLoopParent = -1;

        @Value("${mcp.max.initial.linelength:4096}")
        protected Integer maxInitialLineLength = 4096;

        @Value("${mcp.max.header.size:8192}")
        protected Integer maxHeaderSize = 8192;

        @Value("${mcp.max.chunk.size:8192}")
        protected Integer maxChunkSize = 8192;

        @Value("${mcp.request.max:1024000}")
        protected Integer requestMax = -1;

        @Value("${mcp.binding:0.0.0.0}")
        protected String binding;

        @Value("${mcp.idle:3000}")
        protected Integer idle;

        @Value("${mcp.port:9997}")
        protected Integer port;

        @Autowired
        protected NettyMcpHandler mcpHandler;

        @Bean
        @ConditionalOnMissingBean(value = NettyMcpServer.class)
        public NettyMcpServer nettyMcpServer() throws Exception {
            NettyMcpServer nettyMcpServer = new NettyMcpServer();
            BeanUtils.copyProperties(this, nettyMcpServer);
            log.info("NettyMcpServer inited");
            return nettyMcpServer;
        }
    }
}
