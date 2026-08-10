package ai.open.right.workflow.flow.config;

import ai.open.right.utils.CollectionsUtils;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;
import org.springframework.util.CollectionUtils;

import java.util.Map;

@Getter
@Setter
public class GlobalConfig {

    @JsonProperty("global")
    // 全局配置，用于自定义Assistant的配置
    protected Map<String, Object> globalConfig;

    public GlobalConfig merge(GlobalConfig globalConfig) throws Exception {
        if (globalConfig != null) {
            this.globalConfig = CollectionsUtils.merge(this.globalConfig, globalConfig.globalConfig);
        }
        return this;
    }

    public Boolean hasGlobalConfig() {
        return !CollectionUtils.isEmpty(this.globalConfig);
    }
}
