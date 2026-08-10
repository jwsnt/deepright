package ai.open.right.workflow.flow.select;


import ai.open.right.workflow.flow.function.FunctionConfig;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;

@Setter
@Getter
// COT模式动态选择下一个思考链（Workflow）
public class ChainSelectConfig extends FunctionConfig {

    // 用于动态选择下一个思考链（Workflow）的思考链（Workflow）
    protected String dynamic;

    // 失败时的默认Chain
    protected String chain;

    public ChainSelectConfig merge(ChainSelectConfig chainSelectConfig) throws Exception {
        super.merge(chainSelectConfig);
        if (chainSelectConfig != null) {
            this.dynamic = StringUtils.defaultIfBlank(this.dynamic, chainSelectConfig.dynamic);
            this.chain = StringUtils.defaultIfBlank(this.chain, chainSelectConfig.chain);
        }
        return this;
    }

    public Boolean hasFunction() {
        return !StringUtils.isEmpty(this.getName());
    }

    public Boolean hasDynamic() {
        return !StringUtils.isEmpty(this.dynamic);
    }

    public Boolean hasChain() {
        return !StringUtils.isEmpty(this.chain);
    }
}
