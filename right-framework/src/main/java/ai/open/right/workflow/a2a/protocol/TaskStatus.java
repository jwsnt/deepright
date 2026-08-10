package ai.open.right.workflow.a2a.protocol;

import lombok.*;

@Setter
@Getter
@Builder
@AllArgsConstructor
@NoArgsConstructor
public class TaskStatus {

    // 任务已暂停，等待用户输入（多轮交互）
    public static final String STATUS_INPUT_REQUIRED = "input-required";

    // 任务需要身份验证才能继续
    public static final String STATUS_AUTH_REQUIRED = "auth-required";

    // 任务已提交，正在等待执行
    public static final String STATUS_SUBMITTED = "submitted";

    // 任务已成功完成
    public static final String STATUS_COMPLETED = "completed";

    // 任务被代理拒绝，未启动
    public static final String STATUS_REJECTED = "rejected";

    // 任务已被用户取消
    public static final String STATUS_CANCELED = "canceled";

    // 代理正在处理任务
    public static final String STATUS_WORKING = "working";

    // 任务处于未知或不确定状态
    public static final String STATUS_UNKNOWN = "unknown";

    // 任务因执行期间出错而失败
    public static final String STATUS_FAILED = "failed";

    @Builder.Default
    protected String state = TaskStatus.STATUS_COMPLETED;
}
