package ai.open.right.workflow.flow.parallel;

import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;

@Setter
@Getter
public class ParallelFlow extends GlobalConfig {

    // 并行分支失败是否终止整个流程
    protected Boolean stopOnFailed = false;

    // 并行分支调用的思考链（Workflow）
    protected String dynamic;
}
