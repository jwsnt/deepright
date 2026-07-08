package ai.deepright.cli;

import static org.springframework.util.StringUtils.hasText;



import ai.open.right.protocol.ProtocolCode;

import ai.open.right.WorkflowException;

import lombok.*;

@Builder
@Getter
@Setter
@AllArgsConstructor
@NoArgsConstructor
public class CliSubData {

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
        WorkflowException.check(this.subOps == null, "The sub ops can not be empty", ProtocolCode.C400);
        WorkflowException.check(!hasText(this.router), "The device can not be empty", ProtocolCode.C400);
        WorkflowException.check(this.unwind == null, "The unwind can not be empty", ProtocolCode.C400);
        WorkflowException.check(!hasText(this.cmd), "The cmd can not be empty", ProtocolCode.C400);
        WorkflowException.check(this.ddl == null, "The ddl can not be empty", ProtocolCode.C400);
        return this;
    }

    public Boolean isExpired() throws Exception {
        return System.currentTimeMillis() > this.ddl;
    }
}
