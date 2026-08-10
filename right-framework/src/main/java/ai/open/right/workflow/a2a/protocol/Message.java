package ai.open.right.workflow.a2a.protocol;

import lombok.Getter;
import lombok.Setter;

import java.util.List;
import java.util.Map;

// 表示客户端与客服人员之间的单次沟通或一条上下文信息
@Setter
@Getter
public class Message {

    // 可选元数据
    protected Map<String, Object> metadata;

    // 构成消息体的内容部分数组，消息可以由不同类型的多个部分组成（如文本和文件）
    protected List<Part> parts;

    // 消息的唯一标识符，通常是UUID，由发送者生成
    protected String messageId;

    // 服务器生成的唯一标识符（如UUID），用于在多个相关任务或交互中维护上下文（类似Chat）
    protected String contextId;

    // 此消息所属任务的ID，新任务第一条消息可省略
    protected String taskId;

    // 始终为`message`
    protected String kind = "message";

    // 标识消息的发送者，客户端为`user`，服务为`agent`
    protected String role;
}
    