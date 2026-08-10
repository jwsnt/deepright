package ai.open.right.workflow.flow.pubsub;

import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;

@Setter
@Getter
public class PubSubConfig extends GlobalConfig {

    // 是否开启多论会话记忆
    protected Boolean containHistories;

    protected Integer timeout4Llm;

    // 指定格式化PubSubFormatter
    protected String formatter;

    // 指定通知方式
    protected String notifier;

    // 是否指定特殊的响应Code
    protected Integer code;

    // 失败时固定答案
    protected String reply;

    public PubSubConfig merge(PubSubConfig pubSubConfig) throws Exception {
        super.merge(pubSubConfig);
        if (pubSubConfig != null) {
            this.containHistories = this.containHistories != null ? this.containHistories : pubSubConfig.containHistories;
            this.timeout4Llm = this.timeout4Llm != null ? this.timeout4Llm : pubSubConfig.timeout4Llm;
            this.formatter = StringUtils.defaultIfBlank(this.formatter, pubSubConfig.formatter);
            this.notifier = StringUtils.defaultIfBlank(this.notifier, pubSubConfig.notifier);
            this.reply = StringUtils.defaultIfBlank(this.reply, pubSubConfig.reply);
            this.code = this.code != null ? this.code : pubSubConfig.code;
        }
        return this;
    }

    public PubSubConfig init(String notifier) {
        this.notifier = StringUtils.defaultString(this.notifier, notifier);
        return this;
    }

    public Boolean getContainHistories() {
        return this.containHistories != null ? this.containHistories : false;
    }

    public Integer getTimeout4Llm(Integer timeout4llm) {
        return this.timeout4Llm != null ? this.timeout4Llm : timeout4llm;
    }

    public String getNotifier(String notifier) {
        return !StringUtils.isEmpty(this.notifier) ? this.notifier : notifier;
    }

    public Integer getCode() {
        return this.code != null ? this.code : ProtocolCode.C200;
    }

    public Boolean hasFormatter() {
        return StringUtils.isEmpty(this.formatter);
    }

    public Boolean hasReply() {
        return !StringUtils.isEmpty(this.reply);
    }
}
