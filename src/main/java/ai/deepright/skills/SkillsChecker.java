package ai.deepright.skills;

import ai.open.right.workflow.flow.WorkflowTask;

public interface SkillsChecker {

    public static final String PLUGIN_BROWSER_SKILL = "__internal_browser";

    public static final String PLUGIN_REMOTE_SKILL = "__internal_remote";

    public static final String PLUGIN_FEISHU_SKILL = "__internal_feishu";

    public static final String PLUGIN_EMAIL_SKILL = "__internal_email";

    public static final String PLUGIN_BROWSER_NAME = "browser";

    public static final String PLUGIN_REMOTE_NAME = "remote";

    public static final String PLUGIN_FEISHU_NAME = "feishu";

    public static final String PLUGIN_EMAIL_NAME = "email";

    public Boolean allowedSkill(WorkflowTask workTask, String skill) throws Exception;
}