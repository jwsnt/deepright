package ai.open.right.workflow.flow.mapcombine;

import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;
import org.springframework.util.StringUtils;

@Setter
@Getter
public class MapCombineConfig extends GlobalConfig {

    // 调用LLM时的超时
    protected Integer timeout4Llm;

    protected Combine combine;

    protected Mapping mapping;

    public MapCombineConfig merge(MapCombineConfig mapCombineConfig) throws Exception {
        super.merge(mapCombineConfig);
        if (mapCombineConfig != null) {
            this.timeout4Llm = this.timeout4Llm != null ? this.timeout4Llm : mapCombineConfig.timeout4Llm;
            this.combine = this.combine != null ? this.combine.merge(mapCombineConfig.combine) : mapCombineConfig.combine;
            this.mapping = this.mapping != null ? this.mapping.merge(mapCombineConfig.mapping) : mapCombineConfig.mapping;
        }
        return this;
    }

    public MapCombineConfig init(String notifier) {
        if (this.mapping != null) {
            this.mapping.init(notifier);
        }
        if (this.combine != null) {
            this.combine.init(notifier);
        }
        return this;
    }

    public Boolean isValid() {
        // 包含Mapping或Combine的思考链配置，且Combine批次必须要大于0
        return (this.mapping != null && this.combine != null) && (this.combine.getBatch() > 0 && (StringUtils.hasText(this.combine.getDynamic()) && StringUtils.hasText(this.mapping.getDynamic()) && StringUtils.hasText(this.mapping.getSplit())));
    }

    public Integer getTimeout4Llm(Integer timeout4llm) {
        return this.timeout4Llm != null ? this.timeout4Llm : timeout4llm;
    }
}

