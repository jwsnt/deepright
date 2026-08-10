package ai.open.right.netty.chat.server.ws;

import ai.open.right.netty.NettyServer;
import io.netty.channel.ChannelHandler;
import io.netty.handler.codec.http.HttpObjectAggregator;
import io.netty.handler.codec.http.HttpServerCodec;
import io.netty.handler.codec.http.websocketx.WebSocketServerProtocolHandler;
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
public class NettyWsServer extends NettyServer {

    protected Integer eventLoopChildren = -1;

    protected Integer eventLoopParent = -1;

    protected Integer maxInitialLineLength = 4096;

    protected Integer maxHeaderSize = 8192;

    protected Integer maxChunkSize = 8192;

    protected Integer requestMax = -1;

    protected String binding;

    protected Integer idle;

    protected Integer port;

    protected NettyWsHandler handler;

    @Override
    protected ProxyChannelInitializer buildHandler() {
        return new WsInitializer(this.bufferRecv, this.bufferSend);
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

    public class WsInitializer extends ProxyChannelInitializer {

        public WsInitializer(Integer bufferRecv, Integer bufferSend) {
            super(bufferRecv, bufferSend);
        }

        @Override
        public ChannelHandler[] getChannelHandler() {
            ChannelHandler[] handlers = new ChannelHandler[5];
            handlers[0] = new IdleStateHandler(NettyWsServer.this.idle, NettyWsServer.this.idle, NettyWsServer.this.idle);
            handlers[1] = new HttpServerCodec(NettyWsServer.this.maxInitialLineLength, NettyWsServer.this.maxHeaderSize, NettyWsServer.this.maxChunkSize);
            handlers[2] = new HttpObjectAggregator(NettyWsServer.this.requestMax == -1 ? Integer.MAX_VALUE : NettyWsServer.this.requestMax);
            handlers[3] = new WebSocketServerProtocolHandler("/ws", null, false, NettyWsServer.this.requestMax == -1 ? Integer.MAX_VALUE : NettyWsServer.this.requestMax);
            handlers[4] = NettyWsServer.this.handler;
            return handlers;
        }
    }

    @ConditionalOnProperty(name = "chat.ws.enable", havingValue = "true", matchIfMissing = false)
    @Setter
    @Getter
    @Configuration
    public static class InitConfig extends NettyServerInitConfig {

        @Value("${chat.ws.eventloop.children:-1}")
        protected Integer eventLoopChildren = -1;

        @Value("${chat.ws.eventloop.parent:-1}")
        protected Integer eventLoopParent = -1;

        @Value("${chat.ws.max.initial.linelength:4096}")
        protected Integer maxInitialLineLength = 4096;

        @Value("${chat.ws.max.header.size:8192}")
        protected Integer maxHeaderSize = 8192;

        @Value("${chat.ws.max.chunk.size:8192}")
        protected Integer maxChunkSize = 8192;

        @Value("${chat.ws.request.max:1024000}")
        protected Integer requestMax = -1;

        @Value("${chat.ws.binding:0.0.0.0}")
        protected String binding;

        @Value("${chat.ws.idle:3000}")
        protected Integer idle;

        @Value("${netty.chat.ws.port:9999}")
        protected Integer port;

        @Autowired
        protected NettyWsHandler handler;

        @Bean
        @ConditionalOnMissingBean(value = NettyWsServer.class)
        public NettyWsServer nettyWsServer() throws Exception {
            NettyWsServer nettyWsServer = new NettyWsServer();
            BeanUtils.copyProperties(this, nettyWsServer);
            log.info("NettyWsServer inited");
            return nettyWsServer;
        }
    }
}
