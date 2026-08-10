package ai.open.right.workflow.flow.llm.rag.remote;

import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;
import org.springframework.util.CollectionUtils;

import java.util.List;

@Setter
@Getter
public class RagRemoteConfig extends GlobalConfig {

    // 需要追加的Header
    protected List<RagRemoteHeader> headers;

    public Boolean hasHeaders() {
        return !CollectionUtils.isEmpty(this.headers);
    }
}
