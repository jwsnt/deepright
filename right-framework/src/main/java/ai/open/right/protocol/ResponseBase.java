package ai.open.right.protocol;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

@Setter
@Getter
@NoArgsConstructor
@AllArgsConstructor
public class ResponseBase<T> {

    public static final ResponseBase<Void> EMPTY = ResponseBase.build(null, ProtocolCode.C200, "success");

    protected Integer code;

    protected String msg;

    protected T data;

    public static <T> ResponseBase<T> build(T data, Integer code, String msg) {
        return new ResponseBase<T>(code, msg, data);
    }

    public static <T> ResponseBase<T> build(T data) {
        return new ResponseBase<T>(ProtocolCode.C200, "success", data);
    }

    public static ResponseBase<Void> build() {
        return ResponseBase.EMPTY;
    }
}
