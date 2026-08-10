package ai.open.right.workflow.flow.command;

import java.util.List;

// 用于替换Query部分内容的快捷指令
public interface QuickCommandStore {

    public void store(List<QuickCommand> commands, Integer expire, String biz, String chat, String device);

    public void store(List<QuickCommand> commands, String biz, String chat, String device);

    public List<QuickCommand> restore(String biz, String chat, String device);

    public void clear(String biz, String chat, String device);
}
