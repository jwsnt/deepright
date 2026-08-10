package ai.open.right.workflow.a2a;

import java.util.Map;

// A2A请求
public interface A2ARequest {

    // 报文内容
    public Map<String, Object> getContent();

    // Header请求
    public Map<String, String> getHeaders();

    public String getMethod();

    public String getTrace();

    // Http Path
    public String getPath();

    public Object getId();

    // 回写报文（Stream）
    public void writeStream(A2AResponse response) throws Exception;

    // 回写报文（Once）
    public void writeOnce(Object response) throws Exception;

    // 主动关闭
    public void close() throws Exception;
}
