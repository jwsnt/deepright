package ai.open.right.workflow.a2a.protocol;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

import java.util.List;
import java.util.UUID;

public class AgentSkillTest {

    @Test
    void testGetSet() {
        AgentSkill skill = new AgentSkill();
        List<String> outputModes = List.of("output");
        skill.setOutputModes(outputModes);
        assertSame(outputModes, skill.getOutputModes());
        List<String> inputModes = List.of("input");
        skill.setInputModes(inputModes);
        assertSame(inputModes, skill.getInputModes());
        skill.setDescription("desc");
        assertEquals("desc", skill.getDescription());
        List<String> tags = List.of("tag");
        skill.setTags(tags);
        assertSame(tags, skill.getTags());
        skill.setName("name");
        assertEquals("name", skill.getName());
        String id = UUID.randomUUID().toString();
        skill.setId(id);
        assertEquals(id, skill.getId());
        assertEquals(List.of("right"), AgentSkill.TAG);

        AgentSkill built = AgentSkill.builder().build();
        assertEquals("right", built.getDescription());
        assertEquals(AgentSkill.TAG, built.getTags());
        assertEquals("right a2a server", built.getName());
        assertNotNull(built.getId());
    }
}