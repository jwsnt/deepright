package ai.deepright.skills;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.skill.SkillMetadata;
import com.google.common.collect.ImmutableMap;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;

import java.util.List;
import java.util.Map;

public class SkillsSelector {

    public static SkillMetadata select(List<Map<String, Object>> skills, String name) throws Exception {
        if (CollectionUtils.isEmpty(skills)) {
            return null;
        }
        for (Map<String, Object> skill : skills) {
            if (StringUtils.equalsIgnoreCase(name, MapUtils.getString(skill, "name"))) {
                return SkillMetadata.builder()
                        .internal(ImmutableMap.of("location", skill.get("location")))
                        .description(MapUtils.getString(skill, "description"))
                        .name(MapUtils.getString(skill, "name"))
                        .build();
            }
        }
        return null;
    }

    public static Boolean contain(List<Map<String, Object>> skills, String name) throws Exception {
        if (CollectionUtils.isEmpty(skills)) {
            return false;
        }
        for (Map<String, Object> skill : skills) {
            if (StringUtils.equalsIgnoreCase(name, MapUtils.getString(skill, "name"))) {
                return true;
            }
        }
        return false;
    }

    public static SkillMetadata select(WorkflowTask workTask, String name) throws Exception {
        return SkillsSelector.select(workTask.getMetadata("skills", List.class), name);
    }

    public static Boolean contain(WorkflowTask workTask, String name) throws Exception {
        return SkillsSelector.contain(workTask.getMetadata("skills", List.class), name);
    }
}
