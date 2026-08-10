package ai.open.right.netty.chat.distribute;

import ai.open.right.listener.EventListenerService;
import ai.open.right.netty.NettyCaptor;
import ai.open.right.netty.chat.NettyInputProxy;
import ai.open.right.trace.TraceService;
import ai.open.right.workflow.config.TokenMapping;
import ai.open.right.workflow.config.impl.TokenMappingImpl;
import ai.open.right.workflow.flow.Workflow;
import io.netty.channel.ChannelHandlerContext;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.scheduling.annotation.Async;

@Slf4j
@Setter
@Getter
public class NettyDistributor {

    public static final String NAME = "NettyGptDistributor";

    // 内部事件监听
    protected EventListenerService eventListenerService;

    // 用于链路跟踪
    protected TraceService traceService;

    // Token准入映射
    protected TokenMapping tokenMapping;

    // Chat Track的Netty回调
    protected NettyTrack nettyTrack;

    protected Workflow workflow;

    @Async("executor")
    public void distribute(ChannelHandlerContext context, NettyInputProxy proxy, NettyCaptor captor) throws Exception {
        try (NettyInputProxy input = proxy) {
            NettyRequest request = this.buildRequest(context, input);
            if (log.isDebugEnabled()) {
                log.debug("Received request={} ", request);
            }
            request.setTrace(this.traceService.getTrace(request.getTrace()));
            // Chat Track的Netty回调（写入报文时）
            request.setNettyTrack(this.nettyTrack);
            NettyRequest.NettyRequestChecker.check(request);
            if (this.eventListenerService != null) {
                this.eventListenerService.listen(new NettyEvent(request));
            }
            // sync
            this.workflow.sync(request);
        } catch (Exception e) {
            // 实际异常处理（回调）
            captor.exceptionCaught(context, e);
        }
    }

    protected NettyRequest buildRequest(ChannelHandlerContext context, NettyInputProxy input) throws Exception {
        return input.buildRequest(context, this.tokenMapping);
    }

    @ConditionalOnProperty(name = "chat.enable", havingValue = "true", matchIfMissing = true)
    @Configuration("NettyGptInitConfig")
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired(required = false)
        protected EventListenerService eventListenerService;

        @Autowired
        protected TraceService traceService;

        @Autowired
        @Qualifier(TokenMappingImpl.NAME)
        protected TokenMapping tokenMapping;

        @Autowired
        protected NettyTrack nettyTrack;

        @Autowired
        protected Workflow workflow;

        @Bean(name = NettyDistributor.NAME)
        @ConditionalOnMissingBean(value = NettyDistributor.class)
        public NettyDistributor distributor() throws Exception {
            NettyDistributor nettyDistributor = new NettyDistributor();
            BeanUtils.copyProperties(this, nettyDistributor);
            log.info("NettyDistributor inited");
            return nettyDistributor;
        }
    }
}
