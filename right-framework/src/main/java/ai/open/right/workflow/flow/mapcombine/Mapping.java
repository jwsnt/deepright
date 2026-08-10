package ai.open.right.workflow.flow.mapcombine;

import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;

@Setter
@Getter
public class Mapping extends GlobalConfig {

    // 任一失败是否终止整个流程
    protected Boolean stopOnFailed;

    // 通知方式（Localhost/Endpoint/Source）
    protected String notifier;

    // Map调用的下游思考链（Workflow）
    protected String dynamic;

    // Map Split调用的下游思考链（Workflow）
    protected String split;

    public Mapping merge(Mapping mapping) throws Exception {
        super.merge(mapping);
        if (mapping != null) {
            this.stopOnFailed = this.stopOnFailed != null ? this.stopOnFailed : mapping.stopOnFailed;
            this.notifier = this.notifier != null ? this.notifier : mapping.notifier;
            this.dynamic = this.dynamic != null ? this.dynamic : mapping.dynamic;
            this.split = this.split != null ? this.split : mapping.split;
        }
        return this;
    }

    public Mapping init(String notifier) {
        this.notifier = StringUtils.defaultString(this.notifier, notifier);
        return this;
    }

    public Boolean hasNotifier() {
        return !StringUtils.isEmpty(this.notifier);
    }

    public Boolean getStopOnFailed() {
        return this.stopOnFailed != null ? this.stopOnFailed : true;
    }
}

