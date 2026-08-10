package ai.open.right.workflow.a2a.protocol;

import lombok.*;

import java.util.List;
import java.util.UUID;

// 表示代理可以执行的特定能力或功能
@Setter
@Getter
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class AgentSkill {

    public static final List<String> TAG = List.of("right");

    // 此技能支持的输出MIME类型集，覆盖代理的默认值
    protected List<String> outputModes;

    // 此技能支持的输入MIME类型集，覆盖代理的默认值
    protected List<String> inputModes;

    @Builder.Default
    // 技能的详细描述，旨在帮助客户端或用户理解其目的和功能
    protected String description = "right";

    @Builder.Default
    // 描述技能能力的一组关键字
    protected List<String> tags = AgentSkill.TAG;

    @Builder.Default
    // 技能的人类可读名称
    protected String name = "right a2a server";

    @Builder.Default
    // 代理技能的唯一标识符
    private String id = UUID.randomUUID().toString();
}
    