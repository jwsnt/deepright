package ai.open.right.workflow.skill;

import com.fasterxml.jackson.annotation.JsonIgnore;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Builder;
import lombok.Getter;

import java.util.Map;

@Getter
@Builder
// https://agentskills.io/specification
public class SkillMetadata {

    // 内部用途
    @JsonIgnore
    protected Map<String, Object> internal;

    @JsonProperty("metadata")
    protected Map<String, Object> metadata;

    @JsonProperty("allowed-tools")
    protected String[] allowedTools;

    protected String compatibility;

    protected String description;

    @JsonIgnore
    protected String path;

    protected String name;
}