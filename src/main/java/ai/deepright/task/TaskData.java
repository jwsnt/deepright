package ai.deepright.task;

import static org.springframework.util.ObjectUtils.isEmpty;

import static org.springframework.util.StringUtils.hasText;




import ai.open.right.protocol.ProtocolCode;

import ai.open.right.WorkflowException;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.lang3.StringUtils;

import java.util.List;

@Getter
@Setter
public class TaskData {

    protected List<TaskFunction.TaskArtifact> artifacts;

    @JsonProperty("timeout_seconds")
    protected Integer timeout;

    protected String content;

    @JsonProperty("async_execute")
    // 批里任意一个为true，整批都走异步返回
    protected Boolean async;

    @JsonProperty("target_device")
    protected String device;

    @JsonProperty("target_agent")
    protected String agent;

    @JsonProperty("why_do_this")
    protected String why;

    public TaskData check() throws Exception {
        // 会抛出模型，需要字段对齐
        WorkflowException.check(!hasText(this.device), "The target_device can not be empty", ProtocolCode.C400);
        WorkflowException.check(!hasText(this.agent), "The target_agent can not be empty", ProtocolCode.C400);
        WorkflowException.check(!hasText(this.content), "The content can not be empty", ProtocolCode.C400);
        return this;
    }

    public Boolean hasArtifacts() throws Exception {
        return !CollectionUtils.isEmpty(this.artifacts);
    }

    // 任一不为空
    public Boolean hasAnyBody() {
        return !StringUtils.isEmpty(this.getContent()) || !CollectionUtils.isEmpty(this.getArtifacts());
    }

    public Boolean isAsync() {
        return this.async != null ? this.async : false;
    }
}