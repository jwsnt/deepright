package ai.open.right.workflow.flow.tools;

import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;
import org.springframework.util.StringUtils;

@Setter
@Getter
public class ToolsHeader extends GlobalConfig {

    // 失败时是否终止整个流程
    protected Boolean stopOnFailed = false;

    // 下游思考链（Workflow)
    protected String dynamic;

    // Http Header Key
    protected String key;

    // Http Header Val
    protected String val;

    public Boolean hasDynamic() {
        return StringUtils.hasText(this.dynamic);
    }
}
