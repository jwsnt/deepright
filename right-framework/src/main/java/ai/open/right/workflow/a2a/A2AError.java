package ai.open.right.workflow.a2a;

import lombok.Builder;
import lombok.Getter;
import lombok.Setter;

// A2A通用错误
@Setter
@Getter
@Builder
public class A2AError {

    // 未找到方法，请求的A2A RPC method不存在或不受支持
    public static final int METHOD_NOT_FOUND = -32601;

    // 无效请求，JSON有效负载是有效的JSON，但不是有效的JSON-RPC请求对象
    public static final int INVALID_REQUEST = -32600;

    // 参数无效，该params方法提供的信息无效（例如，类型错误、缺少必填字段）
    public static final int INVALID_PARAMS = -32602;

    // 内部错误，处理期间服务器发生意外错误
    public static final int INTERNAL_ERROR = -32603;

    // 解析错误，服务器收到的JSON格式不正确
    public static final int PARSE_ERROR = -32700;

    // 错误描述
    protected String message;

    // 错误码
    protected Integer code;

    // 任意数据
    protected Object data;
}
