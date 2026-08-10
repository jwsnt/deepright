package ai.open.right.workflow.a2a.server.impl;

import ai.open.right.WorkflowException;
import ai.open.right.workflow.a2a.A2AError;
import ai.open.right.workflow.a2a.A2ARequest;
import ai.open.right.workflow.a2a.server.A2ACmdExportService;
import ai.open.right.workflow.a2a.server.A2ADistributor;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.List;

@Setter
@Getter
@Slf4j
public class A2ADistributorImpl implements A2ADistributor {

    @Autowired
    protected List<A2ACmdExportService> a2aCmdExportService;

    @Override
    public void distribute(A2ARequest a2aRequest) throws Exception {
        try {
            for (A2ACmdExportService a2ACmdExportService : this.a2aCmdExportService) {
                if (a2ACmdExportService.support(a2aRequest)) {
                    a2ACmdExportService.cmd(a2aRequest);
                    return;
                }
            }
            // 无法解析，通用错误
            a2aRequest.writeOnce(A2AError.builder().code(A2AError.METHOD_NOT_FOUND).build());
        } catch (Exception e) {
            WorkflowException.dolog(e);
            // 异常错误
            a2aRequest.writeOnce(A2AError.builder()
                    .code(WorkflowException.code(e))
                    .message(e.getMessage())
                    .build());
        }
    }

    @ConditionalOnProperty(name = "a2a.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected List<A2ACmdExportService> a2aCmdExportService;

        @Bean
        @ConditionalOnMissingBean(value = A2ADistributor.class)
        public A2ADistributor a2aDistributor() throws Exception {
            A2ADistributorImpl a2aDistributor = new A2ADistributorImpl();
            BeanUtils.copyProperties(this, a2aDistributor);
            if (log.isDebugEnabled()) {
                log.debug("A2ADistributorImpl inited");
            }
            return a2aDistributor;
        }
    }
}
