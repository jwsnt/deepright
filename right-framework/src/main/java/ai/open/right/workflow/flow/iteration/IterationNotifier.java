package ai.open.right.workflow.flow.iteration;

import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;

@Setter
@Getter
public class IterationNotifier extends GlobalConfig {

    // 反思的通知方式（Localhost/Endpoint/Source）
    protected String refection;

    // 执行的通知方式（Localhost/Endpoint/Source）
    protected String processor;

    public IterationNotifier merge(IterationNotifier iterationNotifier) throws Exception {
        super.merge(iterationNotifier);
        if (iterationNotifier != null) {
            this.refection = StringUtils.defaultIfBlank(this.refection, iterationNotifier.refection);
            this.processor = StringUtils.defaultIfBlank(this.processor, iterationNotifier.processor);
        }
        return this;
    }

    public IterationNotifier init(String notifier) {
        this.refection = StringUtils.defaultString(this.refection, notifier);
        this.processor = StringUtils.defaultString(this.processor, notifier);
        return this;
    }

    public Boolean hasProcessor() {
        return !StringUtils.isEmpty(this.processor);
    }

    public Boolean hasRefection() {
        return !StringUtils.isEmpty(this.refection);
    }
}
