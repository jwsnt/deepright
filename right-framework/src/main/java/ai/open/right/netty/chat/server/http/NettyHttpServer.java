package ai.open.right.netty.chat.server.http;

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
public class NettyHttpServer extends NettyServer {

    protected Integer eventLoopChildren = -1;

    protected Integer eventLoopParent = -1;

    protected Integer maxInitialLineLength = 4096;

    protected Integer maxHeaderSize = 8192;

    protected Integer maxChunkSize = 8192;

    protected Integer requestMax = -1;

    protected String binding;

    protected Integer idleW;

    protected Integer idleR;

    protected Integer idleA;

    protected Integer port;

    protected NettyHttpHandler httpHandler;

    protected NettyCorsHandler corsHandler;

    @Override
    protected ProxyChannelInitializer buildHandler() {
        return new HttpInitializer(this.bufferRecv, this.bufferSend);
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

    public class HttpInitializer extends ProxyChannelInitializer {

        public HttpInitializer(Integer bufferRecv, Integer bufferSend) {
            super(bufferRecv, bufferSend);
        }

        @Override
        public ChannelHandler[] getChannelHandler() {
            ChannelHandler[] handlers = new ChannelHandler[5];
            handlers[0] = new IdleStateHandler(NettyHttpServer.this.idleR, NettyHttpServer.this.idleW, NettyHttpServer.this.idleA);
            handlers[1] = new HttpServerCodec(NettyHttpServer.this.maxInitialLineLength, NettyHttpServer.this.maxHeaderSize, NettyHttpServer.this.maxChunkSize);
            handlers[2] = new HttpObjectAggregator(NettyHttpServer.this.requestMax == -1 ? Integer.MAX_VALUE : NettyHttpServer.this.requestMax);
            if (NettyHttpServer.this.corsHandler != null) {
                if (log.isDebugEnabled()) {
                    log.debug("Open the http cors handler");
                }
                handlers[3] = NettyHttpServer.this.corsHandler;
                handlers[4] = NettyHttpServer.this.httpHandler;
            } else {
                handlers[3] = NettyHttpServer.this.httpHandler;
            }
            return handlers;
        }
    }

    @ConditionalOnProperty(name = "chat.http.enable", havingValue = "true", matchIfMissing = true)
    @Setter
    @Getter
    @Configuration
    public static class InitConfig extends NettyServerInitConfig {

        @Value("${chat.http.eventloop.children:-1}")
        protected Integer eventLoopChildren = -1;

        @Value("${chat.http.eventloop.parent:-1}")
        protected Integer eventLoopParent = -1;

        @Value("${chat.http.max.initial.linelength:4096}")
        protected Integer maxInitialLineLength = 4096;

        @Value("${chat.http.max.header.size:8192}")
        protected Integer maxHeaderSize = 8192;

        @Value("${chat.http.max.chunk.size:8192}")
        protected Integer maxChunkSize = 8192;

        @Value("${chat.http.request.max:1024000}")
        protected Integer requestMax = -1;

        @Value("${chat.http.binding:0.0.0.0}")
        protected String binding;

        @Value("${chat.http.idle.write:3000}")
        protected Integer idleW;

        @Value("${chat.http.idle.read:3000}")
        protected Integer idleR;

        @Value("${chat.http.idle.all:3000}")
        protected Integer idleA;

        @Value("${chat.http.port:9998}")
        protected Integer port;

        @Autowired
        protected NettyHttpHandler httpHandler;

        @Autowired(required = false)
        protected NettyCorsHandler corsHandler;

        @Bean
        @ConditionalOnMissingBean(value = NettyHttpServer.class)
        public NettyHttpServer nettyHttpServer() throws Exception {
            NettyHttpServer nettyHttpServer = new NettyHttpServer();
            BeanUtils.copyProperties(this, nettyHttpServer);
            log.info("NettyHttpServer inited");
            return nettyHttpServer;
        }
    }
}
