package ai.open.right.workflow.flow.llm.signal;

import ai.open.right.utils.CollectionsUtils;
import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;
import org.apache.commons.lang3.StringUtils;

import java.util.HashMap;
import java.util.Map;

@Setter
@Getter
@ToString
public class SignalConfig extends GlobalConfig {

    // Signal信号量配置
    private Map<String, String> configs = new HashMap<String, String>();

    // 调用下游思考链（Workflow）超时
    protected Integer timeout4Llm;

    // 多路Signal信号归并的思考链（Workflow）
    protected String synthesizer;

    // 当前思考链（Workflow）如果触发Signal，是否静默
    protected Boolean silent;

    public SignalConfig merge(SignalConfig signalConfig) throws Exception {
        super.merge(signalConfig);
        if (signalConfig != null) {
            this.timeout4Llm = this.timeout4Llm != null ? this.timeout4Llm : signalConfig.timeout4Llm;
            this.synthesizer = StringUtils.defaultIfBlank(this.synthesizer, signalConfig.synthesizer);
            this.configs = CollectionsUtils.merge(this.configs, signalConfig.configs);
            this.silent = this.silent != null ? this.silent : signalConfig.silent;
        }
        return this;
    }

    public Boolean hasSynthesizer() {
        return !StringUtils.isEmpty(this.synthesizer);
    }

    public String getDynamic(String key) {
        return this.configs.get(key);
    }

    public Integer getTimeout4Llm(Integer timeout4llm) {
        return this.timeout4Llm != null ? this.timeout4Llm : timeout4llm;
    }

    public Boolean getSilent() {
        return this.silent != null ? this.silent : true;
    }
}
