package ai.open.right.workflow.a2a.server.cmd;

import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.a2a.A2AResponse;
import com.fasterxml.jackson.annotation.JsonIgnore;
import lombok.*;
import org.checkerframework.checker.units.qual.A;
import org.checkerframework.checker.units.qual.N;

// A2A基础Response
@Setter
@Getter
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class A2ACmdResponse implements A2AResponse {

    @Builder.Default
    @JsonIgnore
    protected Boolean finished = false;

    @Builder.Default
    protected String jsonrpc = "2.0";

    // Message | Task
    protected Object result;

    protected Object id;

    @Override
    public Boolean isFinished() {
        return this.finished;
    }

    @Override
    public Integer getCode() {
        return ProtocolCode.C200;
    }
}
