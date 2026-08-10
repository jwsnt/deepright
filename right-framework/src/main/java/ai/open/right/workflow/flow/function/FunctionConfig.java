package ai.open.right.workflow.flow.function;

import ai.open.right.utils.CollectionsUtils;
import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.CollectionUtils;

import java.util.Map;

@Setter
@Getter
public class FunctionConfig extends GlobalConfig {

    // 传递给脚本的静态环境变量
    protected Map<String, String> environment;

    // Original=False则执行完毕后传递脚本响应，Original=True则执行完毕传递原始Query（脚本响应不传递）
    protected Boolean original;

    // 脚本加载的资源位置（URI格式）
    protected String resource;

    // 调用超时（覆盖默认超时）
    protected Integer timeout;

    // 脚本解析器名称
    protected String name;

    public FunctionConfig merge(FunctionConfig functionConfig) throws Exception {
        super.merge(functionConfig);
        if (functionConfig != null) {
            this.environment = CollectionsUtils.merge(this.environment, functionConfig.environment);
            this.resource = StringUtils.defaultIfBlank(this.resource, functionConfig.resource);
            this.original = this.original != null ? this.original : functionConfig.original;
            this.timeout = this.timeout != null ? this.timeout : functionConfig.timeout;
            this.name = StringUtils.defaultIfBlank(this.name, functionConfig.name);
        }
        return this;
    }

    public Boolean getOriginal() {
        return this.original != null ? this.original : false;
    }

    public Boolean hasEnvironment() {
        return !CollectionUtils.isEmpty(this.environment);
    }

    public Boolean hasResource() {
        return !StringUtils.isEmpty(this.resource);
    }

    public Integer getTimeout(Integer timeout) {
        return this.timeout != null ? this.timeout : timeout;
    }

    public String getName(String name) {
        return StringUtils.defaultIfBlank(this.name, name);
    }
}
