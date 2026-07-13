package ai.deepright.cli;

import ai.open.right.WorkflowException;
import lombok.*;
import org.apache.commons.lang3.StringUtils;

@Builder
@Getter
@Setter
@AllArgsConstructor
@NoArgsConstructor
public class CliSubData {

    @Builder.Default
    protected Long created = System.currentTimeMillis();

    protected CliSubOps subOps;

    protected String conversation;

    protected String workspace;

    // 提供给下游的推荐超时
    protected Integer timeout;

    protected String agentId;

    protected Boolean unwind;

    protected String suffix;

    protected String router;

    protected String chat;

    protected String type = CliPubSub.CMD;

    protected String tid;

    protected String why;

    protected String cmd;

    protected Long ddl;

    public CliSubData check() throws Exception {
        WorkflowException.checkCondition(StringUtils.isEmpty(this.router), "The device can not be empty");
        WorkflowException.checkCondition(this.subOps == null, "The sub ops can not be empty");
        WorkflowException.checkCondition(this.unwind == null, "The unwind can not be empty");
        WorkflowException.checkCondition(StringUtils.isEmpty(this.cmd), "The cmd can not be empty");
        WorkflowException.checkCondition(this.ddl == null, "The ddl can not be empty");
        return this;
    }

    public Boolean isExpired() throws Exception {
        return System.currentTimeMillis() > this.ddl;
    }
}
