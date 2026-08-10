package ai.open.right.netty.chat.distribute;

import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.track.TrackChat;
import ai.open.right.workflow.flow.track.TrackChatBody;
import ai.open.right.workflow.flow.track.TrackChatService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.builder.ToStringBuilder;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
@Setter
@Getter
// Netty消息回调
public class NettyTrack {

    protected TrackChatService trackChatService;

    public void track(WorkflowTask workTask, Segment segment) {
        try {
            if (this.trackChatService != null) {
                this.trackChatService.store(TrackChat.builder()
                        .trackChatBody(new TrackChatBody(segment))
                        .dimension(workTask)
                        .build());
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
    }

    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired(required = false)
        protected TrackChatService trackChatService;

        @Bean
        @ConditionalOnMissingBean(value = NettyTrack.class)
        public NettyTrack nettyTrack() throws Exception {
            NettyTrack nettyTrack = new NettyTrack();
            BeanUtils.copyProperties(this, nettyTrack);
            log.info("NettyTrack inited={}", ToStringBuilder.reflectionToString(nettyTrack));
            return nettyTrack;
        }
    }
}
