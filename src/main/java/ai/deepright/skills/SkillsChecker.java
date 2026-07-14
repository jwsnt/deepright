package ai.deepright.skills;

import ai.open.right.workflow.flow.WorkflowTask;

public interface SkillsChecker {

    public static final String PLUGIN_BROWSER_SKILL = "__internal_browser";

    public static final String PLUGIN_REMOTE_SKILL = "__internal_remote";

    public static final String PLUGIN_BROWSER_SWITCH = "browser";

    public static final String PLUGIN_REMOTE_SWITCH = "remote";

    public static final String PLUGIN_FEISHU_SWITCH = "feishu";

    public static final String PLUGIN_EMAIL_SWITCH = "email";

    public Boolean allowedSkill(WorkflowTask workTask, String skill) throws Exception;
}