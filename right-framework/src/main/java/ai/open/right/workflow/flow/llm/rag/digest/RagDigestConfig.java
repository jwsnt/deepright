package ai.open.right.workflow.flow.llm.rag.digest;

import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;

import java.util.List;

@Setter
@Getter
public class RagDigestConfig extends GlobalConfig {

    public static final String MODE_JSON = "json";

    public static final String MODE_XML = "xml";

    // 允许摘要记忆的属性名称
    protected List<String> keys;

    // 下游思考链（Workflow）
    protected String dynamic;

    // 摘要记忆的存储Key（可选），用于多个摘要共享记忆
    protected String scene;

    // 解析模式（XML/JSON）
    protected String mode = RagDigestConfig.MODE_XML;

    public String getScene(String scene) {
        return StringUtils.defaultString(this.scene, scene);
    }

    public Boolean isMode(String mode) {
        return StringUtils.trim(this.mode).equals(mode);
    }
}
