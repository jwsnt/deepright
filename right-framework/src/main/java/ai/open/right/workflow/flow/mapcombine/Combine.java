package ai.open.right.workflow.flow.mapcombine;

import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;

@Setter
@Getter
public class Combine extends GlobalConfig {

    // 任一失败是否终止整个流程
    protected Boolean stopOnFailed;

    // 通知方式（Localhost/Endpoint/Source）
    protected String notifier;

    // Combine调用的下游思考链（Workflow）
    protected String dynamic;

    // 最大归并批次
    protected Integer batch;

    public Combine merge(Combine combine) throws Exception {
        super.merge(combine);
        if (combine != null) {
            this.stopOnFailed = this.stopOnFailed != null ? this.stopOnFailed : combine.stopOnFailed;
            this.notifier = this.notifier != null ? this.notifier : combine.notifier;
            this.dynamic = this.dynamic != null ? this.dynamic : combine.dynamic;
            this.batch = this.batch != null ? this.batch : combine.batch;
        }
        return this;
    }

    public Combine init(String notifier) {
        this.notifier = StringUtils.defaultString(this.notifier, notifier);
        return this;
    }

    public Boolean hasNotifier() {
        return !StringUtils.isEmpty(this.notifier);
    }

    public Boolean getStopOnFailed() {
        return this.stopOnFailed != null ? this.stopOnFailed : true;
    }

    public Integer getBatch() {
        return this.batch != null ? this.batch : Integer.MAX_VALUE;
    }
}
