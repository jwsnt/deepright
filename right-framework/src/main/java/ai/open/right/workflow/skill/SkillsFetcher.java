package ai.open.right.workflow.skill;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.AllowedConfig;

public interface SkillsFetcher {

    // 获取技能资源
    public String fetchResource(WorkflowTask workTask, String name, String path) throws Exception;

    public Skills fetchSkills(WorkflowTask workTask, AllowedConfig allowedConfig) throws Exception;

    public Skills fetchSkills(WorkflowTask workTask) throws Exception;
}
