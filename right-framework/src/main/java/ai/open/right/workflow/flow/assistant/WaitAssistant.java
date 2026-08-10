package ai.open.right.workflow.flow.assistant;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
@Getter
@Setter
public class WaitAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-wait";

    protected Integer max;

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        Integer wait = this.doWait(workTask, this.buildWait(workTask));
        this.chainOr2Endpoint(workflowConfig, workTask, this.buildContent(workTask, wait));
    }

    protected String buildContent(WorkflowTask workTask, Integer wait) throws Exception {
        return "Waited " + wait + " ms successfully";
    }

    protected Integer doWait(WorkflowTask workTask, Integer wait) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("start do wait task: {}", workTask);
        }
        Thread.sleep(wait);
        return wait;
    }

    protected Integer buildWait(WorkflowTask workTask) throws Exception {
        Integer wait = workTask.getObjectQuery(Integer.class);
        if (wait != null) {
            return Math.min(this.max, wait);
        } else {
            return this.max;
        }
    }

    @ConditionalOnProperty(name = "wait.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Value("${wait.max:5000}")
        protected Integer max;

        @Bean(WaitAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = WaitAssistant.WORKFLOW_NAME)
        public WaitAssistant waitAssistant() throws Exception {
            WaitAssistant waitAssistant = new WaitAssistant();
            BeanUtils.copyProperties(this, waitAssistant);
            log.info("WaitAssistant inited");
            return waitAssistant;
        }
    }
}
