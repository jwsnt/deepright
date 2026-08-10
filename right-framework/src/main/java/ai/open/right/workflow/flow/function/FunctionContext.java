package ai.open.right.workflow.flow.function;

import ai.open.right.workflow.flow.WorkflowTask;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;

@Setter
@Getter
@Builder
public class FunctionContext {

    protected final FunctionConfig functionConfig;

    protected final WorkflowTask workTask;
}
