package ai.open.right.workflow.flow.llm.rag.remote;

import lombok.Getter;
import lombok.Setter;
import org.springframework.util.StringUtils;

@Setter
@Getter
public class RagRemoteHeader {

    // 追加失败是否终止整个流程
    protected Boolean stopOnFailed = false;

    // 用于获取Header值的下游思考链（Workflow）
    protected String dynamic;

    // Header key
    protected String key;

    // Header value
    protected String val;

    public Boolean hasDynamic() {
        return StringUtils.hasText(this.dynamic);
    }
}
