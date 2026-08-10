package ai.open.right.workflow.flow.tools;

import ai.open.right.workflow.flow.command.QuickCommand;
import lombok.*;

import java.util.List;
import java.util.Map;

@Getter
@Setter
// Tools Response包装
public class ToolsBody {

    protected Map<String, Object> metadata;

    protected List<QuickCommand> command;

    protected String content;
}
