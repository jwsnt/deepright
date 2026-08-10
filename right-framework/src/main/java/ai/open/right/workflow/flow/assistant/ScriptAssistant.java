package ai.open.right.workflow.flow.assistant;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.script.ScriptResponse;
import ai.open.right.workflow.flow.script.ScriptSegment;
import ai.open.right.workflow.flow.script.ScriptService;
import ai.open.right.workflow.flow.script.impl.ScriptServiceImpl;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

@Slf4j
@Setter
@Getter
// 脚本
public class ScriptAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-script";

    protected ScriptService scriptService;

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        Assert.isTrue(!StringUtils.isEmpty(workTask.getQuery()), "Query can not be empty");
        String response = this.scriptService.run(workflowConfig.getScriptConfig(), workTask);
        if (!StringUtils.isEmpty(response) && workflowConfig.hasScript() && workflowConfig.getScriptConfig().shouldWrap()) {
            // 将Response解析为ScriptResponse
            if (log.isInfoEnabled()) {
                log.info("Prepare to convert to the json format of ScriptResponse={}", response);
            }
            ScriptResponse scriptResponse = JsonUtils.read(response, ScriptResponse.class);
            ScriptSegment scriptSegment = new ScriptSegment(workTask, scriptResponse);
            // 指定Notifier
            scriptSegment.setNotifier(workflowConfig.getNotifier(scriptSegment.getNotifier()));
            if (!workflowConfig.getScriptConfig().isSuccessCode(scriptSegment.getCode())) {
                // 非200通知
                this.notifierService.notify(scriptSegment, workTask, workTask);
            } else {
                String content = JsonUtils.write(scriptSegment.getData());
                super.chainOr2Endpoint(workflowConfig, workTask, content);
            }
            return;
        }
        // Not wrap
        this.chainOr2Endpoint(workflowConfig, workTask, response);
    }

    @ConditionalOnProperty(name = "script.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        @Qualifier(ScriptServiceImpl.NAME)
        protected ScriptService scriptService;

        @Bean(ScriptAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = ScriptAssistant.WORKFLOW_NAME)
        public ScriptAssistant scriptAssistant() throws Exception {
            ScriptAssistant scriptAssistant = new ScriptAssistant();
            BeanUtils.copyProperties(this, scriptAssistant);
            log.info("ScriptAssistant inited");
            return scriptAssistant;
        }
    }
}
