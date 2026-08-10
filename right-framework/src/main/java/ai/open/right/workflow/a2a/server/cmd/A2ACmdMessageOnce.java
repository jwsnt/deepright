package ai.open.right.workflow.a2a.server.cmd;

import ai.open.right.workflow.a2a.A2ARequest;
import ai.open.right.workflow.a2a.protocol.MessageRequest;
import ai.open.right.workflow.sync.SyncCallable;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Setter
@Getter
@Slf4j
public class A2ACmdMessageOnce extends A2ACmdExportMessage {

    public static final String METHOD = "message/send";

    @Override
    public SyncCallable buildSyncCallable(A2ARequest a2aRequest, MessageRequest messageRequest) throws Exception {
        return new A2ACmdCallableOnce(a2aRequest, messageRequest);
    }

    @Override
    public Boolean support(A2ARequest a2aRequest) throws Exception {
        return StringUtils.equalsIgnoreCase(a2aRequest.getMethod(), A2ACmdMessageOnce.METHOD);
    }

    @ConditionalOnProperty(name = "a2a.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends A2ACmdInitConfig {

        @Bean(name = A2ACmdMessageOnce.METHOD)
        @ConditionalOnMissingBean(name = A2ACmdMessageOnce.METHOD)
        public A2ACmdMessageOnce a2aCmdMessageOnce() throws Exception {
            A2ACmdMessageOnce a2aCmdMessageOnce = new A2ACmdMessageOnce();
            BeanUtils.copyProperties(this, a2aCmdMessageOnce);
            log.info("A2ACmdMessageOnce inited: timeout4Llm={}", a2aCmdMessageOnce.getTimeout4Llm());
            return a2aCmdMessageOnce;
        }
    }
}
