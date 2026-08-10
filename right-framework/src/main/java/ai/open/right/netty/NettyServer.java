package ai.open.right.netty;

import ai.open.right.WorkflowException;
import io.netty.bootstrap.ServerBootstrap;
import io.netty.buffer.PooledByteBufAllocator;
import io.netty.channel.ChannelHandler;
import io.netty.channel.ChannelInitializer;
import io.netty.channel.EventLoopGroup;
import io.netty.channel.nio.NioEventLoopGroup;
import io.netty.channel.socket.SocketChannel;
import io.netty.channel.socket.nio.NioServerSocketChannel;
import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.ApplicationListener;
import org.springframework.context.event.ContextRefreshedEvent;

/**
 * @author shenjiawei
 */
@Slf4j
@Setter
@Getter
abstract public class NettyServer implements ApplicationListener<ContextRefreshedEvent> {

    protected final ServerBootstrap boots = new ServerBootstrap();

    protected Integer bufferSend = -1;

    protected Integer bufferRecv = -1;

    @Override
    public void onApplicationEvent(ContextRefreshedEvent event) {
        try {
            // 必须同步等待，确保启动
            this.boots.bind(this.getBinding(), this.getPort()).sync();
            log.info("Server {}/{} started ...", this.getClass().getSimpleName(), this.getPort());
        } catch (Exception e) {
            log.error("Server {} start failed.", this.getPort(), e);
            throw WorkflowException.create(e);
        }
    }

    @PostConstruct
    public void init() throws Exception {
        int cLoop = this.getEventLoopChildren() == -1 ? Runtime.getRuntime().availableProcessors() * 2 : this.getEventLoopChildren();
        int pLoop = this.getEventLoopParent() == -1 ? Runtime.getRuntime().availableProcessors() : this.getEventLoopParent();
        log.info("Server threads={}/{}", pLoop, cLoop);
        this.boots.group(new NioEventLoopGroup(pLoop), new NioEventLoopGroup(cLoop));
        this.boots.channel(NioServerSocketChannel.class);
        this.boots.childHandler(this.buildHandler());
    }

    @PreDestroy
    public void destroy() throws Exception {
        try (EventLoopGroup child = this.boots.config().childGroup()) {
            child.shutdownGracefully().sync();
        }
        try (EventLoopGroup boots = this.boots.config().group()) {
            boots.shutdownGracefully().sync();
        }
        log.info("Server {} shutdown ...", this.getClass().getSimpleName());
    }

    abstract protected ProxyChannelInitializer buildHandler();

    abstract protected Integer getEventLoopChildren();

    abstract protected Integer getEventLoopParent();

    abstract protected String getBinding();

    abstract protected Integer getPort();

    abstract public static class ProxyChannelInitializer extends ChannelInitializer<SocketChannel> {

        protected final Integer bufferRecv;

        protected final Integer bufferSend;

        public ProxyChannelInitializer(Integer bufferRecv, Integer bufferSend) {
            this.bufferRecv = bufferRecv;
            this.bufferSend = bufferSend;
        }

        public void initChannel(SocketChannel ch) throws Exception {
            if (this.bufferRecv > 0) {
                ch.config().setReceiveBufferSize(this.bufferRecv);
            }
            if (this.bufferSend > 0) {
                ch.config().setSendBufferSize(this.bufferSend);
            }
            ch.config().setAllocator(PooledByteBufAllocator.DEFAULT);
            ch.config().setReuseAddress(true);
            ch.config().setTcpNoDelay(true);
            ch.config().setKeepAlive(true);
            ch.pipeline().addLast(this.getChannelHandler());
        }

        abstract public ChannelHandler[] getChannelHandler();
    }

    @Setter
    @Getter
    public static class NettyServerInitConfig {

        @Value("${netty.buffer.send:65536}")
        protected Integer bufferSend = -1;

        @Value("${netty.buffer.recv:65536}")
        protected Integer bufferRecv = -1;
    }
}
