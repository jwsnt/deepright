package ai.open.right.workflow.flow.llm.signal.impl;

import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.signal.SignalDistributor;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
@Setter
@Getter
public class SignalFactoryImpl implements SignalFactory {

    protected SignalDistributor signalDistributor;

    public SignalStream signal(WorkflowConfig workflowConfig) {
        if (!workflowConfig.hasSignals()) {
            return SignalStream.EMPTY;
        }
        return new SignalStreamImpl(workflowConfig.getSignalConfig(), this.signalDistributor);
    }

    @ConditionalOnProperty(name = "signal.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected SignalDistributor signalDistributor;

        @Bean
        @ConditionalOnMissingBean(value = SignalFactory.class)
        public SignalFactory signalFactory() throws Exception {
            SignalFactoryImpl signalFactory = new SignalFactoryImpl();
            BeanUtils.copyProperties(this, signalFactory);
            log.info("SignalFactoryImpl inited");
            return signalFactory;
        }
    }
}
