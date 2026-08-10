package ai.open.right.workflow.flow.script.impl;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.condition.ConditionUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.script.ScriptConfig;
import ai.open.right.workflow.flow.script.ScriptService;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import ai.open.right.workflow.sync.impl.NotifierCallable;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.util.Assert;

@Slf4j
@Setter
@Getter
abstract public class AbstractScriptService implements ScriptService {

    protected NotifierService notifierService;

    // 脚本判断使用思考链（Workflow）超时
    protected Integer timeout4Condition;

    // 脚本校对使用思考链（Workflow）超时
    protected Integer timeout4Corrector;

    // 脚本执行超时
    protected Integer timeout;

    public String run(ScriptConfig scriptConfig, WorkflowTask workTask) throws Exception {
        ScriptEnv scriptEnv = new ScriptEnv(workTask);
        if (log.isDebugEnabled()) {
            log.debug("Script env={}", scriptEnv);
        }
        if (scriptConfig == null) {
            return this.run(scriptEnv, workTask.getQuery(), this.timeout);
        }
        if (scriptConfig.hasCondition()) {
            // 需要条件判断时
            SyncConfig syncConfig = SyncConfig.builder()
                    .syncCallable(scriptConfig.hasNotifier() ? new NotifierCallable(scriptConfig.getNotifier()) : null)
                    .timeout(scriptConfig.getTimeout4Condition(this.timeout4Condition))
                    .workflow(scriptConfig.getCondition())
                    .workTask(workTask)
                    .build();
            String conditionResponse = SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get();
            if (log.isInfoEnabled()) {
                log.info("Script condition={}-{}", workTask.getQuery(), conditionResponse);
            }
            Assert.hasText(conditionResponse, "Script condition can not be empty");
            // True: True/true/Yes/Y/1
            // False: False/false/No/N/0及其他任意无法解析
            // Json: {...,"condition":true/false/0/1}
            if (!ConditionUtils.checkCondition(conditionResponse).print().getCondition()) {
                throw new WorkflowException("Script can not be allowed to run", ProtocolCode.C401);
            }
        }
        try {
            return this.run(scriptEnv, workTask.getQuery(), scriptConfig.getTimeout(this.timeout));
        } catch (WorkflowException e) {
            // 执行失败则不需要校准时
            if (!(scriptConfig.hasCorrector() && e.getCode().equals(ProtocolCode.C500))) {
                throw e;
            }
            WorkflowException currentException = e;
            // 失败校准
            for (int index = 0; index < scriptConfig.getCorrector().getTimes(); index++) {
                try {
                    SyncConfig syncConfig = SyncConfig.builder()
                            .reQuery(workTask.getQuery() + System.lineSeparator() + e.getMessage())
                            .timeout(scriptConfig.getTimeout4Corrector(this.timeout4Corrector))
                            .workflow(scriptConfig.getCorrector().getCorrection())
                            .workTask(workTask)
                            .build();
                    String correctResponse = SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get();
                    if (log.isInfoEnabled()) {
                        log.info("Script correct={}-{}", workTask.getQuery(), correctResponse);
                    }
                    Assert.hasText(correctResponse, "Correct response can not be empty");
                    return this.run(scriptEnv, correctResponse, scriptConfig.getTimeout(this.timeout));
                } catch (WorkflowException inner) {
                    currentException = inner;
                } catch (Exception other) {
                    currentException = WorkflowException.create(other);
                }
            }
            throw currentException;
        }
    }

    abstract public String run(ScriptEnv scriptEnv, String script, Integer timeout) throws Exception;

    @Setter
    @Getter
    public static class ScriptInitConfig {

        @Autowired
        protected NotifierService notifierService;

        @Value("${script.condition.timeout:1800000}")
        // 脚本判断使用思考链（Workflow）超时
        protected Integer timeout4Condition;

        @Value("${script.corrector.timeout:1800000}")
        // 脚本校对使用思考链（Workflow）超时
        protected Integer timeout4Corrector;

        @Value("${script.timeout:1800000}")
        // 脚本执行超时
        protected Integer timeout;
    }
}
