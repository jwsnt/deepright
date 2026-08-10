package ai.open.right.workflow.flow.script;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;

@Setter
@Getter
@Builder
@AllArgsConstructor
public class ScriptResponse {

    protected Integer code;

    protected Object data;

    public ScriptResponse() {

    }

    public Boolean hasData() {
        return this.data != null;
    }
}
