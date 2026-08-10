package ai.open.right.workflow.flow.tools;

import ai.open.right.protocol.ResponseBase;

import java.util.Map;

public class ToolsResponse extends ResponseBase<ToolsBody> {

    public Boolean hasData() {
        return this.getData() != null;
    }

    public Integer getCode(Integer code) {
        return this.getCode() != null ? this.getCode() : code;
    }

    public String getMsg(String msg) {
        return this.getMsg() != null ? this.getMsg() : msg;
    }

    public Map<String, Object> getMetadata() {
        return this.hasData() ? this.getData().getMetadata() : null;
    }

}
