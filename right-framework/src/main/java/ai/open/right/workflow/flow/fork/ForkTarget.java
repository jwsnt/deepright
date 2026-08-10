package ai.open.right.workflow.flow.fork;

import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;
import org.springframework.util.StringUtils;

@Setter
@Getter
public class ForkTarget extends GlobalConfig {

    // 用于判断条件的思考链（Workflow）
    protected String condition;

    // 条件判断匹配后思考链（Workflow）
    protected String dynamic;

    public Boolean hasCondition() {
        return StringUtils.hasText(this.condition);
    }
}
