package ai.open.right.workflow.flow.competition;

import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

@Setter
@Getter
@ToString
public class ConditionConfig extends GlobalConfig {

    // 用于判断条件的思考链（Workflow）
    protected String condition;

    // 条件匹配传递的思考链（Workflow）
    protected String dynamic;
}
