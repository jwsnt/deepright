package ai.open.right.netty.a2a.distribute;

import ai.open.right.netty.NettyCaptor;
import ai.open.right.netty.a2a.NettyInputProxy;
import ai.open.right.netty.a2a.server.NettyA2ARequest;
import ai.open.right.trace.TraceService;
import ai.open.right.workflow.a2a.server.A2ADistributor;
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

    public static final String NAME = "NettyA2ADistributor";

    // 用于A2A分发
    protected A2ADistributor a2aDistributor;

    // 生成Trace
    protected TraceService traceService;

    @Async("executor")
    public void distribute(ChannelHandlerContext context, NettyInputProxy proxy, NettyCaptor captor) throws Exception {
        try (NettyInputProxy input = proxy) {
            // 初始化
            NettyA2ARequest request = this.buildRequest(context, input);
            if (log.isDebugEnabled()) {
                log.debug("Received request={} ", request);
            }
            // 追加Trace
            request.setTrace(this.traceService.getTrace(request.getTrace()));
            this.a2aDistributor.distribute(request);
        } catch (Exception e) {
            captor.exceptionCaught(context, e);
        }
    }

    protected NettyA2ARequest buildRequest(ChannelHandlerContext context, NettyInputProxy input) throws Exception {
        return input.buildContent(context);
    }

    @ConditionalOnProperty(name = "a2a.enable", havingValue = "true", matchIfMissing = false)
    @Configuration("NettyA2AInitConfig")
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected A2ADistributor a2aDistributor;

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
