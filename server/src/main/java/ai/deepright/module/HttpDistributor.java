package ai.deepright.module;

import static org.springframework.util.StringUtils.hasText;


import ai.open.right.protocol.ProtocolCode;

import ai.deepright.llm.notifier.MultiSourceNotifier;
import ai.open.right.WorkflowException;
import ai.open.right.netty.chat.NettyInputProxy;
import ai.open.right.netty.chat.distribute.NettyDistributor;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.deepright.llm.provider.RequestProviderUtils;
import io.netty.channel.ChannelHandlerContext;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
@Getter
@Setter
public class HttpDistributor extends NettyDistributor {

    public static final String MAIN = "/v1/chat/completions";

    public static final String GET = "/cli/get";

    public static final String PUB = "/cli/pub";

    @Override
    protected NettyRequest buildRequest(ChannelHandlerContext context, NettyInputProxy input) throws Exception {
        NettyRequest request = input.buildRequest(context, this.tokenMapping);
        String device = MapUtils.getString(request.getMetadata(), "deviceId");
        WorkflowException.checkCondition(StringUtils.isEmpty(device), "The request device must not be empty");
        request.getUserContext().setDevice(device);
        String[] part = SplitUtils.split(this.workflow(request));
        request.setWorkflow(part[1]);
        request.setBiz(part[0]);
        return this.provider(request);
    }

    protected NettyRequest provider(NettyRequest request) throws Exception {
        // 识别服务商并替换
        request.putMetadata(ProviderRequestService.KEY_PROVIDER, RequestProviderUtils.findProvider(request));
        return request;
    }

    protected String workflow(NettyRequest request) throws Exception {
        String uri = MapUtils.getString(request.getMetadata(), "uri");
        if (StringUtils.startsWithIgnoreCase(uri, HttpDistributor.MAIN)) {
            return MultiSourceNotifier.MAIN;
        } else if (StringUtils.startsWithIgnoreCase(uri, HttpDistributor.GET)) {
            return "cli@get";
        } else if (StringUtils.startsWithIgnoreCase(uri, HttpDistributor.PUB)) {
            return "cli@pub";
        }
        throw new WorkflowException("The request URI is invalid: " + uri);
    }

    @ConditionalOnProperty(name = "chat.enable", havingValue = "true", matchIfMissing = true)
    @Configuration
    @Setter
    @Getter
    public static class ModuleConfig extends InitConfig {

        @Override
        @Bean(NettyDistributor.NAME)
        @ConditionalOnMissingBean({NettyDistributor.class})
        public HttpDistributor distributor() throws Exception {
            HttpDistributor httpDistributor = new HttpDistributor();
            BeanUtils.copyProperties(this, httpDistributor);
            log.info("HttpDistributor inited");
            return httpDistributor;
        }
    }
}
