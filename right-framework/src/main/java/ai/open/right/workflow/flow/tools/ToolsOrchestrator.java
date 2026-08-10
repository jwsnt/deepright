package ai.open.right.workflow.flow.tools;

import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;

@Setter
@Getter
public class ToolsOrchestrator extends GlobalConfig {

    // 在请求前调用的思考链（Workflow）
    protected String before;

    // 追加的Get参数
    protected String param;

    // 在响应后调用的思考链（Workflow）
    protected String after;

    public ToolsOrchestrator merge(ToolsOrchestrator toolsOrchestrator) throws Exception {
        super.merge(toolsOrchestrator);
        if (toolsOrchestrator != null) {
            this.before = StringUtils.defaultIfBlank(this.before, toolsOrchestrator.before);
            this.param = StringUtils.defaultIfBlank(this.param, toolsOrchestrator.param);
            this.after = StringUtils.defaultIfBlank(this.after, toolsOrchestrator.after);
        }
        return this;
    }

    public Boolean hasBefore() {
        return !StringUtils.isEmpty(this.before) || this.hasParam();
    }

    public Boolean hasParam() {
        return !StringUtils.isEmpty(this.param);
    }

    public Boolean hasAfter() {
        return !StringUtils.isEmpty(this.after);
    }
}
