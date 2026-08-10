package ai.open.right.workflow.flow.function;

import ai.open.right.protocol.ProtocolCode;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

import java.util.Map;

@Setter
@Getter
@Builder
@ToString
public class FunctionResponse {

    protected Map<String, Object> metadata;

    protected Object content;

    @Builder.Default
    protected Integer code = ProtocolCode.C200;
}
