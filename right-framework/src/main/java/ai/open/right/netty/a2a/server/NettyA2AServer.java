package ai.open.right.netty.a2a.server;

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
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

/**
 * @author shenjiawei
 */
@Slf4j
@Setter
@Getter
public class NettyA2AServer extends NettyServer {

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
    protected NettyA2AHandler a2aHandler;

    @Override
    protected ProxyChannelInitializer buildHandler() {
        return new A2AInitializer(this.bufferRecv, this.bufferSend);
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

    public class A2AInitializer extends ProxyChannelInitializer {

        public A2AInitializer(Integer bufferRecv, Integer bufferSend) {
            super(bufferRecv, bufferSend);
        }

        @Override
        public ChannelHandler[] getChannelHandler() {
            ChannelHandler[] handlers = new ChannelHandler[5];
            handlers[0] = new IdleStateHandler(NettyA2AServer.this.idle, NettyA2AServer.this.idle, NettyA2AServer.this.idle);
            handlers[1] = new HttpServerCodec(NettyA2AServer.this.maxInitialLineLength, NettyA2AServer.this.maxHeaderSize, NettyA2AServer.this.maxChunkSize);
            handlers[2] = new HttpObjectAggregator(NettyA2AServer.this.requestMax == -1 ? Integer.MAX_VALUE : NettyA2AServer.this.requestMax);
            handlers[3] = NettyA2AServer.this.a2aHandler;
            return handlers;
        }
    }

    @ConditionalOnProperty(name = "a2a.enable", havingValue = "true", matchIfMissing = false)
    @Setter
    @Getter
    @Configuration
    public static class InitConfig extends NettyServerInitConfig {

        @Value("${netty.a2a.eventloop.children:-1}")
        protected Integer eventLoopChildren = -1;

        @Value("${netty.a2a.eventloop.parent:-1}")
        protected Integer eventLoopParent = -1;

        @Value("${netty.a2a.max.initial.linelength:4096}")
        protected Integer maxInitialLineLength = 4096;

        @Value("${netty.a2a.max.header.size:8192}")
        protected Integer maxHeaderSize = 8192;

        @Value("${netty.a2a.max.chunk.size:8192}")
        protected Integer maxChunkSize = 8192;

        @Value("${netty.a2a.request.max:1024000}")
        protected Integer requestMax = -1;

        @Value("${netty.a2a.binding:0.0.0.0}")
        protected String binding;

        @Value("${netty.a2a.idle:3000}")
        protected Integer idle;

        @Value("${netty.a2a.port:9996}")
        protected Integer port;

        @Autowired
        protected NettyA2AHandler a2aHandler;

        @Bean
        public NettyA2AServer nettyA2AServer() throws Exception {
            NettyA2AServer nettyA2AServer = new NettyA2AServer();
            BeanUtils.copyProperties(this, nettyA2AServer);
            log.info("NettyA2AServer inited");
            return nettyA2AServer;
        }
    }
}
