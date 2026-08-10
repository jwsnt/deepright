package ai.open.right.workflow.flow.media;

import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;

@Setter
@Getter
public class MediaConfig extends GlobalConfig {

    public static final String SPLIT = ";";

    // 调用下游解析思考链（Workflow）的超时
    protected Integer timeout4Llm;

    // 是否解析为Base64
    protected Boolean base64;

    // 调用下游解析思考链（Workflow）
    protected String dynamic;

    // 切分符
    protected String split;

    public MediaConfig merge(MediaConfig mediaConfig) throws Exception {
        super.merge(mediaConfig);
        if (mediaConfig != null) {
            this.timeout4Llm = this.timeout4Llm != null ? this.timeout4Llm : mediaConfig.timeout4Llm;
            this.dynamic = StringUtils.defaultIfEmpty(this.dynamic, mediaConfig.dynamic);
            this.split = StringUtils.defaultIfEmpty(this.split, mediaConfig.split);
            this.base64 = this.base64 != null ? this.base64 : mediaConfig.base64;
        }
        return this;
    }

    public Integer getTimeout4Llm(Integer timeout4Llm) {
        return this.timeout4Llm != null ? this.timeout4Llm : timeout4Llm;
    }

    public String getDynamic(String dynamic) {
        return StringUtils.defaultString(this.dynamic, dynamic);
    }

    public String getSplit(String split) {
        return StringUtils.defaultString(this.split, split);
    }

    public String getSplit() {
        return this.split != null ? this.split : MediaConfig.SPLIT;
    }

    public Boolean getBase64() {
        return this.base64 != null ? this.base64 : false;
    }
}
