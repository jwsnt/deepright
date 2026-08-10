package ai.open.right.workflow.flow.llm.provider.coze;

import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import lombok.Getter;
import lombok.Setter;

// Coze的配置
@Setter
@Getter
public class CozeRequest extends ProviderRequest {

    // Coze botid
    protected String botId;
}
