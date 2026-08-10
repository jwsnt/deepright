package ai.open.right.workflow.flow.script.impl;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import lombok.Builder;
import lombok.Getter;
import lombok.ToString;
import org.apache.commons.lang3.StringUtils;

import java.util.HashMap;

@ToString
public class ScriptEnv extends HashMap<String, String> {

    public static final String KEY_METADATA = "__metadata__";

    public static final String KEY_WORKFLOW = "__workflow__";

    public static final String KEY_USERINFO = "__user__";

    public static final String KEY_DATA = "__data__";

    public static final String KEY_ENV = "__env__";

    public ScriptEnv(WorkflowTask workTask) throws Exception {
        this.put(ScriptEnv.KEY_USERINFO, StringUtils.defaultIfBlank(JsonUtils.write(workTask.getUserContext()), ""));
        this.put(ScriptEnv.KEY_METADATA, StringUtils.defaultIfBlank(JsonUtils.write(workTask.getMetadata()), ""));
        this.put(ScriptEnv.KEY_WORKFLOW, StringUtils.defaultIfBlank(JsonUtils.write(WorkflowInfo.builder()
                .workflow(workTask.getWorkflow())
                .biz(workTask.getBiz())
                .build()), ""));
        this.put(ScriptEnv.KEY_DATA, StringUtils.defaultIfBlank(JsonUtils.write(workTask.getQuery()), ""));
    }

    public ScriptEnv env(Object val) throws Exception {
        super.put(ScriptEnv.KEY_ENV, JsonUtils.write(val));
        return this;
    }

    @Getter
    @Builder
    public static class WorkflowInfo {

        protected String workflow;

        protected String biz;
    }
}
