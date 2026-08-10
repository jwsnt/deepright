package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.utils.JsonUtils;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;

@Setter
@Getter
@Slf4j
// 用于内部Event
public class ProviderData {

    protected StringBuffer responseBuffer = new StringBuffer();

    protected String response = "";

    protected String request;

    // 追加响应
    public void appendResponse(String response) throws Exception {
        this.responseBuffer.append(response);
    }

    // 追加请求
    public void appendRequest(Object request) throws Exception {
        this.request = JsonUtils.write(request);
    }

    public ProviderData init() throws Exception {
        this.response = this.responseBuffer.toString();
        return this;
    }
}
