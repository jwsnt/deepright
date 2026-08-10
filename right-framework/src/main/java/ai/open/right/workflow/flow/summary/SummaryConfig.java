package ai.open.right.workflow.flow.summary;

import ai.open.right.utils.CollectionsUtils;
import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.CollectionUtils;

import java.util.Arrays;
import java.util.List;

@Setter
@Getter
public class SummaryConfig extends GlobalConfig {

    // 摘要需要写入的记忆
    protected List<String> repositories;

    // 是否包含FunCall
    protected Boolean includeFunCall;

    protected Boolean includeReason;

    protected Boolean dropOnFailed;

    // 摘要调用下游思考链（Workflow）的超时
    protected Integer timeout4Llm;

    // 摘要判断条件使用的思考链（Workflow）
    protected String condition;

    // 摘要返回条数
    protected Integer maxsize;

    // 摘要ReStore的持久化时间
    protected Integer expired;

    // 摘要生成使用的思考链（Workflow）
    protected String dynamic;

    // 如果产生MediaContext，是否解析为Base64
    protected Boolean base64;

    // 是否需要ReStore
    protected Boolean store;

    // 是否拆分，默认True
    protected Boolean split;

    // 使用倒排序获取会话
    protected Boolean desc;

    // 摘要需要提取的记忆
    protected String scene;

    protected Long now;

    public SummaryConfig merge(SummaryConfig summaryConfig) throws Exception {
        super.merge(summaryConfig);
        if (summaryConfig != null) {
            this.includeFunCall = this.includeFunCall != null ? this.includeFunCall : summaryConfig.includeFunCall;
            this.includeReason = this.includeReason != null ? this.includeReason : summaryConfig.includeReason;
            this.dropOnFailed = this.dropOnFailed != null ? this.dropOnFailed : summaryConfig.dropOnFailed;
            this.timeout4Llm = this.timeout4Llm != null ? this.timeout4Llm : summaryConfig.timeout4Llm;
            this.repositories = CollectionsUtils.merge(this.repositories, summaryConfig.repositories);
            this.condition = StringUtils.defaultIfBlank(this.condition, summaryConfig.condition);
            this.dynamic = StringUtils.defaultIfBlank(this.dynamic, summaryConfig.dynamic);
            this.expired = this.expired != null ? this.expired : summaryConfig.expired;
            this.maxsize = this.maxsize != null ? this.maxsize : summaryConfig.maxsize;
            this.scene = StringUtils.defaultIfBlank(this.scene, summaryConfig.scene);
            this.store = this.store != null ? this.store : summaryConfig.store;
            this.split = this.split != null ? this.split : summaryConfig.split;
            this.desc = this.desc != null ? this.desc : summaryConfig.desc;
            this.now = this.now != null ? this.now : summaryConfig.now;
        }
        return this;
    }

    public List<String> getRepositories(String repository) {
        String scene = this.getScene(repository);
        if (CollectionUtils.isEmpty(this.repositories)) {
            return Arrays.asList(scene);
        }
        if (!this.repositories.contains(scene)) {
            this.repositories.add(scene);
        }
        return this.repositories;
    }

    public Integer getTimeout4Llm(Integer timeout4llm) {
        return this.timeout4Llm != null ? this.timeout4Llm : timeout4llm;
    }

    public String getScene(String scene) {
        return !StringUtils.isEmpty(this.scene) ? this.scene : scene;
    }

    public Boolean getIncludeFunCall() {
        return this.includeFunCall != null ? this.includeFunCall : true;
    }

    public Boolean getIncludeReason() {
        return this.includeReason != null ? this.includeReason : true;
    }

    public Boolean getDropOnFailed() {
        return this.dropOnFailed != null ? this.dropOnFailed : false;
    }

    public Integer getMaxsize() {
        return this.maxsize != null ? this.maxsize : Integer.MAX_VALUE;
    }

    public Boolean getBase64() {
        return this.base64 != null ? this.base64 : false;
    }

    public Boolean getStore() {
        return this.store != null ? this.store : true;
    }

    public Boolean getSplit() {
        return this.split != null ? this.split : true;
    }

    public Boolean getDesc() {
        return this.desc != null ? this.desc : false;
    }

    public Boolean hasCondition() {
        return !StringUtils.isEmpty(this.condition);
    }

    public Boolean shouldStore() {
        return this.getStore();
    }
}
