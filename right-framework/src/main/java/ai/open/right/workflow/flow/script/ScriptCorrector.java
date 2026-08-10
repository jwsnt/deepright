package ai.open.right.workflow.flow.script;

import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;

@Setter
@Getter
public class ScriptCorrector extends GlobalConfig {

    // 用于校准的思考链（Workflow）
    protected String correction;

    // 校准最大重试次数
    protected Integer times = 1;
}
