package ai.open.right.workflow.flow.command;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;

// 快速指令
@Getter
@Setter
@Builder
@AllArgsConstructor
public class QuickCommand {

    // 相同匹配指令的优先级，越大优先级越高
    private Long priority = System.currentTimeMillis();

    // 需要替换的指令
    protected String command;

    // 用于替换的内容
    protected String content;

    public QuickCommand() {

    }
}
