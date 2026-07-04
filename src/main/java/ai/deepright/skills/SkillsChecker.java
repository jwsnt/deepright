package ai.deepright.skills;

import ai.open.right.workflow.flow.WorkflowTask;

public interface SkillsChecker {

    public static final String PLUGIN_BROWSER = "__internal_browser";

    public static final String PLUGIN_REMOTE = "__internal_remote";

    public Boolean allowedSkill(WorkflowTask workTask, String skill) throws Exception;
}