package ai.open.right.protocol;

public class ProtocolCode {

    // 范围常量
    public static final Integer RANGE = 100;

    // 内部Code
    // Workflow Fun Call接管Code
    public static final Integer I001 = -101;

    public static final Integer C200 = 200;

    // 解析错误
    public static final Integer C400 = 400;

    // 禁止访问
    public static final Integer C401 = 401;

    public static final Integer C404 = 404;

    // 访问限速
    public static final Integer C429 = 429;

    public static final Integer C430 = 430;

    // 内部错误
    public static final Integer C500 = 500;

    // 服务超时
    public static final Integer C502 = 502;

    // Service Unavailable
    public static final Integer C503 = 503;

    // HTTP Client异常
    public static final Integer C800 = 800;

    // 900-9xx 为静默区间
    public static final Integer C900 = 900;

    // LLM可重试错误：FunCall错误，响应为空（Gemini）
    public static final Integer C914 = 914;

    // 通用静默错误
    public static final Integer C915 = 915;

    // 被动关闭
    public static final Integer CN1 = -1;

    // 主动关闭
    public static final Integer C0 = 0;

    // 是否在指定范围内
    public static Boolean rangeCode(Integer code, Integer begin) {
        return code >= begin && code < begin + ProtocolCode.RANGE;
    }

    // 是否为200
    public static Boolean range2xx(Integer code) {
        return ProtocolCode.rangeCode(code, ProtocolCode.C200);
    }

    // 关闭型Code映射调整，小于等于0，复位为200
    public static Integer mapping(Integer code) {
        return code <= ProtocolCode.C0 ? ProtocolCode.C200 : code;
    }
}
