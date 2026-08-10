package ai.open.right.workflow.a2a.protocol;

import lombok.Getter;
import lombok.Setter;

// 定义代理支持的可选功能
@Setter
@Getter
public class AgentCapabilities {

    // 指示代理是否提供任务的状态转换历史（TaskStatus List，不支持）
    protected Boolean stateTransitionHistory = false;

    // 指示代理是否支持发送异步任务更新的推送通知（不支持）
    protected Boolean pushNotifications = false;

    // 指示代理是否支持服务器发送事件（SSE）以进行流式响应
    protected Boolean streaming = false;
}
    