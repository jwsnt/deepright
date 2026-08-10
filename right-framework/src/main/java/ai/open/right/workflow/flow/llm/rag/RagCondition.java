package ai.open.right.workflow.flow.llm.rag;

import ai.open.right.workflow.condition.ConditionUtils;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.util.Assert;

@Slf4j
@Setter
@Getter
public class RagCondition {

    protected NotifierService notifierService;

    // Rag前置条件调用下游思考链（Workflow）超时
    protected Integer timeout4Condition;

    protected Boolean allowed(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!ragConfig.hasCondition()) {
            return true;
        }
        SyncConfig syncConfig = SyncConfig.builder()
                .timeout(ragConfig.getTimeout4Condition(this.timeout4Condition))
                .workflow(ragConfig.getCondition())
                .workTask(ragData.getQuery())
                .build();
        String response = StringUtils.lowerCase(SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get());
        if (log.isInfoEnabled()) {
            log.info("The condition response={}", response);
        }
        Assert.hasText(response, "Response can not be empty");
        // True: True/true/Yes/Y/1
        // False: False/false/No/N/0 or Other
        // Json: {...,"condition":true/false/0/1}
        return ConditionUtils.checkCondition(response).print().getCondition();
    }

    @Setter
    @Getter
    public static class ConditionInitConfig {

        @Autowired
        protected NotifierService notifierService;

        @Value("${rag.condition.timeout:1800000}")
        // Rag前置条件调用下游思考链（Workflow）超时
        protected Integer timeout4Condition;
    }
}
