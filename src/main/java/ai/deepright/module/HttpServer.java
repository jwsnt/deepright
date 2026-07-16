package ai.deepright.module;

import ai.open.right.netty.chat.server.http.NettyHttpServer;
import io.netty.channel.ChannelHandler;
import io.netty.handler.stream.ChunkedWriteHandler;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.ArrayUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
public class HttpServer extends NettyHttpServer {

    @Override
    protected ProxyChannelInitializer buildHandler() {
        return new ChunkInitializer(this.bufferRecv, this.bufferSend);
    }

    public class ChunkInitializer extends HttpInitializer {

        public ChunkInitializer(Integer bufferRecv, Integer bufferSend) {
            super(bufferRecv, bufferSend);
        }

        public ChannelHandler[] getChannelHandler() {
            ChannelHandler[] handlers = super.getChannelHandler();
            int index = handlers.length - 1;
            while (index > 0 && handlers[index] == null) {
                index--;
            }
            return ArrayUtils.insert(index, handlers, new ChunkedWriteHandler());
        }
    }

    @ConditionalOnProperty(name = "chat.http.enable", havingValue = "true", matchIfMissing = true)
    @Setter
    @Getter
    @Configuration
    public static class ChunkInitConfig extends InitConfig {

        @Bean
        public NettyHttpServer nettyHttpServer() throws Exception {
            HttpServer httpServer = new HttpServer();
            BeanUtils.copyProperties(this, httpServer);
            log.info("HttpServer inited");
            return httpServer;
        }
    }
}
