package ai.deepright.cli;

import lombok.*;
import org.springframework.util.Assert;

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
        Assert.notNull(this.subOps, "The sub ops can not be empty");
        Assert.hasText(this.router, "The device can not be empty");
        Assert.notNull(this.unwind, "The unwind can not be empty");
        Assert.hasText(this.cmd, "The cmd can not be empty");
        Assert.notNull(this.ddl, "The ddl can not be empty");
        return this;
    }

    public Boolean isExpired() throws Exception {
        return System.currentTimeMillis() > this.ddl;
    }
}
